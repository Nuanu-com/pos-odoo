package odoo

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
)

type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	ID      uint64 `json:"id"`
	Params  any    `json:"params"`
}

type jsonRPCResponse[T any] struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      uint64        `json:"id"`
	Result  T             `json:"result,omitempty"`
	Error   *jsonRPCError `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int              `json:"code"`
	Message string           `json:"message"`
	Data    jsonRPCErrorData `json:"data"`
}

type jsonRPCErrorData struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}

type callKwParams struct {
	Model  string `json:"model"`
	Method string `json:"method"`
	Args   []any  `json:"args"`
	KWArgs any    `json:"kwargs"`
}

// rpcError carries both the Odoo exception class name and human-readable message.
// Using a distinct type (rather than errors.New) lets callers inspect the name
// with errors.As — used internally to detect session expiry.
type rpcError struct {
	name    string
	message string
}

func (e *rpcError) Error() string { return e.message }

func isSessionExpired(err error) bool {
	var e *rpcError
	return errors.As(err, &e) && strings.Contains(e.name, "SessionExpiredException")
}

// SearchReadResult is returned by web_search_read calls.
type SearchReadResult[T any] struct {
	Records []T `json:"records"`
	Length  int `json:"length"`
}

// OdooClient wraps Odoo's JSON-RPC API.
type OdooClient interface {
	AuthenticatedUserID() int
	SearchReadAccountTax(domain []any, fields []string, limit, offset int) (*SearchReadResult[AccountTax], error)
	SearchReadAccountTaxWithOptions(opts SearchReadAccountTaxOptions) (*SearchReadResult[AccountTax], error)
	ComputeAllTax(taxIDs []int, priceUnit float64, quantity float64) (*AccountTaxComputeAllResult, error)
	CreateProductTemplate(input ProductTemplateInput) (int, error)
	CreateProduct(input ProductProductInput) (int, error)
	WriteProductTemplate(ids []int, input ProductWriteInput) error
	WriteProduct(ids []int, input ProductWriteInput) error
	ReadProductByID(id int, fields []string) (*ProductProduct, error)
	SearchReadProduct(domain []any, fields []string, limit, offset int) ([]ProductProduct, error)
	SearchReadProductsByTemplateID(templateID int, fields []string) ([]ProductProduct, error)
	ReadPOSConfig(configID int) (*POSConfig, error)
	ReadPOSSession(sessionID int) (*POSSessionInfo, error)
	CreatePOSSession(configID int) (int, error)
	ClosePOSSession(sessionID int, bankPaymentMethodDiffs []BankPaymentMethodDiff) error
	PostClosingCashDetails(sessionID int, countedCash float64) error
	UpdateClosingControlStateSession(sessionID int, notes string) error
	TryCashInOut(sessionID int, moveType CashMoveType, amount float64, reason string) error
	CreatePOSOrder(orders []POSOrderData, draft bool) ([]POSOrderResult, error)
	WritePOSOrder(ids []int, fields map[string]any) error
	RefundPOSOrder(orderID int) (int, error)
}

type odooClientImpl struct {
	httpClient       *http.Client
	baseURL          string
	idCounter        atomic.Uint64
	mu               sync.Mutex
	authUsername     string
	authPassword     string
	authDB           string
	authenticatedUID atomic.Int64
}

type OdooOption func(*odooClientImpl)

func AddAuthentication(pred func() (username, password, db string)) func(*odooClientImpl) {
	authUsername, authPassword, authDB := pred()

	return func(oci *odooClientImpl) {
		oci.authUsername = authUsername
		oci.authPassword = authPassword
		oci.authDB = authDB
	}
}

// NewOdooClient creates a new Odoo client. The caller is responsible for
// supplying an *http.Client with a cookie jar when session auth is needed:
//
//	jar, _ := cookiejar.New(nil)
//	client := odoo.NewOdooClient(&http.Client{Jar: jar}, baseURL)
//
// In tests, pass a cassette-backed client directly.
func NewOdooClient(httpClient *http.Client, baseURL string, opts ...OdooOption) OdooClient {
	oddoClient := &odooClientImpl{
		httpClient: httpClient,
		baseURL:    baseURL,
	}

	if len(opts) > 0 {
		for _, opt := range opts {
			opt(oddoClient)
		}
	}

	return oddoClient
}

// AuthenticatedUserID returns the Odoo user ID captured during the last
// successful authentication. Returns 0 before the first auth completes.
func (o *odooClientImpl) AuthenticatedUserID() int {
	return int(o.authenticatedUID.Load())
}

func (o *odooClientImpl) nextID() uint64 {
	return o.idCounter.Add(1)
}

func (o *odooClientImpl) doAuthenticate(username, password, db string) error {
	type authParams struct {
		DB       string `json:"db"`
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	type authResult struct {
		UID json.RawMessage `json:"uid"`
	}

	payload := jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "call",
		ID:      o.nextID(),
		Params:  authParams{DB: db, Login: username, Password: password},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := o.httpClient.Post(o.baseURL+"/web/session/authenticate", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var rpcResp jsonRPCResponse[authResult]
	if err := json.Unmarshal(raw, &rpcResp); err != nil {
		return err
	}
	if rpcResp.Error != nil {
		return errors.New(rpcResp.Error.Data.Message)
	}
	// Odoo returns false (not an int uid) when credentials are wrong.
	if string(rpcResp.Result.UID) == "false" {
		return errors.New("odoo: authentication failed — invalid credentials")
	}
	var uid int64
	if err := json.Unmarshal(rpcResp.Result.UID, &uid); err == nil {
		o.authenticatedUID.Store(uid)
	}
	return nil
}

// callKw sends a JSON-RPC /web/dataset/call_kw request. On session expiry it
// re-authenticates once (using stored credentials) and retries the call.
func callKw[T any](o *odooClientImpl, model, method string, args []any, kwargs any) (T, error) {
	result, err := doCallKw[T](o, model, method, args, kwargs)
	if err == nil || !isSessionExpired(err) || o.authDB == "" {
		return result, err
	}
	// Acquire the lock so concurrent goroutines don't all re-authenticate at once.
	o.mu.Lock()
	authErr := o.doAuthenticate(o.authUsername, o.authPassword, o.authDB)
	o.mu.Unlock()
	if authErr != nil {
		return result, err // return the original session-expired error
	}
	return doCallKw[T](o, model, method, args, kwargs)
}

func doCallKw[T any](o *odooClientImpl, model, method string, args []any, kwargs any) (T, error) {
	var zero T
	payload := jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "call",
		ID:      o.nextID(),
		Params: callKwParams{
			Model:  model,
			Method: method,
			Args:   args,
			KWArgs: kwargs,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return zero, err
	}
	resp, err := o.httpClient.Post(
		o.baseURL+"/web/dataset/call_kw/"+model+"/"+method,
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return zero, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, err
	}
	var rpcResp jsonRPCResponse[T]
	if err := json.Unmarshal(raw, &rpcResp); err != nil {
		return zero, err
	}
	if rpcResp.Error != nil {

		b, _ := json.Marshal(rpcResp)
		slog.Error("Error calling Odoo", slog.Any("E", b))
		return zero, &rpcError{
			name:    rpcResp.Error.Data.Name,
			message: rpcResp.Error.Data.Message,
		}
	}
	return rpcResp.Result, nil
}
