package odoo

import (
	"errors"
	"fmt"

	"github.com/leekchan/accounting"
)

// CashMoveType is the direction of a mid-session drawer movement.
type CashMoveType string

const (
	CashMoveIn  CashMoveType = "in"
	CashMoveOut CashMoveType = "out"
)

// BankPaymentMethodDiff pairs a bank-type payment method with the cash-count
// difference to post when closing a session via ClosePOSSession.
type BankPaymentMethodDiff struct {
	PaymentMethodID int
	DiffAmount      float64
}

type postClosingCashDetailsKWArgs struct {
	CountedCash float64 `json:"counted_cash"`
}

type closeSessionResult struct {
	Successful bool   `json:"successful"`
	Message    string `json:"message"`
	Redirect   bool   `json:"redirect"`
}

// POSSessionInfo holds the fields returned by pos.session/read.
type POSSessionInfo struct {
	ID                             int     `json:"id"`
	Name                           string  `json:"name"`
	SequenceNumber                 int     `json:"sequence_number"`
	CashRegisterStart              float64 `json:"cash_register_balance_start"`
	CashRegisterTotalEntryEncoding float64 `json:"cash_register_total_entry_encoding"`
	CashRegisterBalanceEnd         float64 `json:"cash_register_balance_end"`
	CashRegisterBalanceEndReal     float64 `json:"cash_register_balance_end_real"`
	CashRegisterDifference         float64 `json:"cash_register_difference"`
}

var defaultPOSSessionFields = []string{
	"id",
	"name",
	"sequence_number",
	"cash_register_total_entry_encoding",
	"cash_register_balance_end",
	"cash_register_balance_end_real",
	"cash_register_difference",
}

// ReadPOSSession calls /web/dataset/call_kw/pos.session/read and returns the
// session's ID, name, and current sequence_number.
func (o *odooClientImpl) ReadPOSSession(sessionID int) (*POSSessionInfo, error) {
	records, err := callKw[[]POSSessionInfo](o, "pos.session", "read", []any{[]int{sessionID}}, readKWArgs{Fields: defaultPOSSessionFields})
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("odoo: pos.session %d not found", sessionID)
	}
	return &records[0], nil
}

// CreatePOSSession calls /web/dataset/call_kw/pos.session/create.
// Returns the new session ID. The session starts in opening_control state if
// the config has cash control enabled, otherwise it transitions directly to opened.
func (o *odooClientImpl) CreatePOSSession(configID int) (int, error) {
	return callKw[int](o, "pos.session", "create", []any{map[string]any{"config_id": configID}}, struct{}{})
}

// ClosePOSSession calls /web/dataset/call_kw/pos.session/close_session_from_ui.
// bankPaymentMethodDiffs posts loss/profit for bank-type payment methods; pass nil if none.
// Returns an error if the session cannot be closed (e.g. draft orders still open).
// Odoo signals close failure via successful:false in the result rather than an RPC error.
func (o *odooClientImpl) ClosePOSSession(sessionID int, bankPaymentMethodDiffs []BankPaymentMethodDiff) error {
	pairs := make([]any, len(bankPaymentMethodDiffs))
	for i, d := range bankPaymentMethodDiffs {
		pairs[i] = []any{d.PaymentMethodID, d.DiffAmount}
	}
	result, err := callKw[closeSessionResult](o, "pos.session", "close_session_from_ui", []any{[]int{sessionID}, pairs}, struct{}{})
	if err != nil {
		return err
	}
	if !result.Successful {
		return errors.New(result.Message)
	}
	return nil
}

// PostClosingCashDetails calls /web/dataset/call_kw/pos.session/post_closing_cash_details.
// Sets cash_register_balance_end_real to countedCash. Only needed when the session's
// config has cash_control enabled — call this before ClosePOSSession in that case.
// Odoo signals failure via successful:false in the result rather than an RPC error.
func (o *odooClientImpl) PostClosingCashDetails(sessionID int, countedCash float64) error {
	result, err := callKw[closeSessionResult](o, "pos.session", "post_closing_cash_details", []any{[]int{sessionID}}, postClosingCashDetailsKWArgs{
		CountedCash: countedCash,
	})
	if err != nil {
		return err
	}
	if !result.Successful {
		return errors.New(result.Message)
	}
	return nil
}

// UpdateClosingControlStateSession calls /web/dataset/call_kw/pos.session/update_closing_control_state_session.
// Marks the session state as "closing_control" and stamps stop_at; pass notes="" if there are none.
// Unlike PostClosingCashDetails/ClosePOSSession, Odoo has no successful:false convention here —
// it returns nothing on success and raises a UserError (surfaced as an RPC error) on failure,
// e.g. when the session is already closed.
func (o *odooClientImpl) UpdateClosingControlStateSession(sessionID int, notes string) error {
	_, err := callKw[bool](o, "pos.session", "update_closing_control_state_session", []any{[]int{sessionID}, notes}, struct{}{})
	return err
}

// cashInOutExtras is the `extras` positional argument of try_cash_in_out.
// Odoo reads both keys unconditionally — a missing one raises a KeyError rather
// than a UserError, so both are always sent.
type cashInOutExtras struct {
	TranslatedType  string `json:"translatedType"`
	FormattedAmount string `json:"formattedAmount"`
}

// TryCashInOut calls /web/dataset/call_kw/pos.session/try_cash_in_out to record a
// mid-session drawer movement (cash taken out to pay a supplier, a float top-up, …).
// This is not the end-of-day count — that is PostClosingCashDetails.
//
// amount must be positive for both directions: Odoo applies the sign itself
// (sign = 1 if _type == 'in' else -1), so a negative amount with CashMoveOut posts
// a cash in. reason is appended to the statement line's payment_ref and is the only
// human-readable trace of the move — pass a non-empty one.
//
// The session's config must have a cash payment method (cash_journal_id set),
// otherwise Odoo raises "There is no cash payment method for this PoS Session".
// Odoo returns null on success, so only the RPC error matters here.
func (o *odooClientImpl) TryCashInOut(sessionID int, moveType CashMoveType, amount float64, reason string) error {
	if moveType != CashMoveIn && moveType != CashMoveOut {
		return fmt.Errorf("odoo: invalid cash move type %q, want %q or %q", moveType, CashMoveIn, CashMoveOut)
	}
	if amount <= 0 {
		return fmt.Errorf("odoo: cash move amount must be positive, got %v", amount)
	}

	ac := accounting.Accounting{Symbol: "Rp ", Precision: 0, Thousand: "."}
	extras := cashInOutExtras{
		TranslatedType:  string(moveType),
		FormattedAmount: ac.FormatMoney(amount),
	}

	_, err := callKw[bool](o, "pos.session", "try_cash_in_out", []any{
		[]int{sessionID},
		string(moveType),
		amount,
		reason,
		extras,
	}, struct{}{})
	return err
}
