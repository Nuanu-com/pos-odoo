package odoo_test

import (
	"encoding/json"
	"io"
	"net/http"

	odoo "github.com/Nuanu-com/pos-odoo"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type sentRequest struct {
	Path   string
	Params map[string]any
}

// fakeOdoo replays the given response bodies in order and records each request.
func fakeOdoo(responses ...string) (*http.Client, *[]sentRequest) {
	sent := &[]sentRequest{}
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		var payload struct {
			Params map[string]any `json:"params"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, err
		}
		i := len(*sent)
		*sent = append(*sent, sentRequest{Path: r.URL.Path, Params: payload.Params})
		Expect(i).To(BeNumerically("<", len(responses)), "unexpected extra request")
		return jsonResponse(responses[i]), nil
	})}
	return client, sent
}

var _ = Describe("OdooClient — API key authentication", func() {
	apiKeyAuth := odoo.AddAPIKeyAuthentication(func() (string, string, string) {
		return "admin", "secret-key", "mydb"
	})

	const authOK = `{"jsonrpc":"2.0","id":1,"result":2}`
	const taxOK = `{"jsonrpc":"2.0","id":2,"result":{"length":1,"records":[{"id":1,"name":"Tax 15%"}]}}`

	It("authenticates once via common.authenticate and calls execute_kw", func() {
		httpClient, sent := fakeOdoo(authOK, taxOK, taxOK)
		client := odoo.NewOdooClient(httpClient, "http://localhost:8069", apiKeyAuth)

		result, err := client.SearchReadAccountTax([]any{}, []string{"name"}, 80, 0)
		Expect(err).To(BeNil())
		Expect(result.Records[0].Name).To(Equal("Tax 15%"))
		Expect(client.AuthenticatedUserID()).To(Equal(2))

		_, err = client.SearchReadAccountTax([]any{}, []string{"name"}, 80, 0)
		Expect(err).To(BeNil())

		Expect(*sent).To(HaveLen(3), "uid should be cached after the first call")

		auth := (*sent)[0]
		Expect(auth.Path).To(Equal("/jsonrpc"))
		Expect(auth.Params["service"]).To(Equal("common"))
		Expect(auth.Params["method"]).To(Equal("authenticate"))
		Expect(auth.Params["args"]).To(Equal([]any{"mydb", "admin", "secret-key", map[string]any{}}))

		call := (*sent)[1]
		Expect(call.Path).To(Equal("/jsonrpc"))
		Expect(call.Params["service"]).To(Equal("object"))
		Expect(call.Params["method"]).To(Equal("execute_kw"))
		args := call.Params["args"].([]any)
		Expect(args[:6]).To(Equal([]any{"mydb", float64(2), "secret-key", "account.tax", "web_search_read", []any{}}))
		Expect(args[6]).To(HaveKeyWithValue("limit", float64(80)))
		Expect(args[6]).To(HaveKeyWithValue("fields", []any{"name"}))
	})

	It("returns an error when the API key is rejected", func() {
		httpClient, sent := fakeOdoo(`{"jsonrpc":"2.0","id":1,"result":false}`)
		client := odoo.NewOdooClient(httpClient, "http://localhost:8069", apiKeyAuth)

		result, err := client.SearchReadAccountTax(nil, nil, 80, 0)

		Expect(result).To(BeNil())
		Expect(err).To(MatchError("odoo: authentication failed — invalid credentials"))
		Expect(*sent).To(HaveLen(1))
		Expect(client.AuthenticatedUserID()).To(Equal(0))
	})

	It("returns the Odoo error message from execute_kw", func() {
		httpClient, _ := fakeOdoo(authOK,
			`{"jsonrpc":"2.0","id":2,"error":{"code":200,"message":"Odoo Server Error","data":{"name":"odoo.exceptions.AccessError","message":"You do not have access to this document."}}}`)
		client := odoo.NewOdooClient(httpClient, "http://localhost:8069", apiKeyAuth)

		result, err := client.SearchReadAccountTax(nil, nil, 80, 0)

		Expect(result).To(BeNil())
		Expect(err).To(MatchError("You do not have access to this document."))
	})

	It("passes kwargs context through (e.g. refund active_ids)", func() {
		httpClient, sent := fakeOdoo(authOK,
			`{"jsonrpc":"2.0","id":2,"result":{"res_id":99}}`)
		client := odoo.NewOdooClient(httpClient, "http://localhost:8069", apiKeyAuth)

		_, _ = client.RefundPOSOrder(42)

		args := (*sent)[1].Params["args"].([]any)
		Expect(args[3:5]).To(Equal([]any{"pos.order", "refund"}))
		Expect(args[6]).To(HaveKeyWithValue("context", HaveKeyWithValue("active_ids", []any{float64(42)})))
	})
})
