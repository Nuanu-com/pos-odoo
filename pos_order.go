package odoo

// POSOrderLineData is a single line item in a POS order.
type POSOrderLineData struct {
	ProductID         int     `json:"product_id"`
	Qty               float64 `json:"qty"`
	PriceUnit         float64 `json:"price_unit"`
	PriceSubtotal     float64 `json:"price_subtotal"`
	PriceSubtotalIncl float64 `json:"price_subtotal_incl"`
	Discount          float64 `json:"discount"`
	TaxIDs            []any   `json:"tax_ids"`      // build with TaxIDs()
	PackLotIDs        []any   `json:"pack_lot_ids"` // []any{} when unused
	FullProductName   string  `json:"full_product_name,omitempty"`
}

// POSOrderPaymentData is a single payment entry in a POS order.
type POSOrderPaymentData struct {
	PaymentMethodID int     `json:"payment_method_id"`
	Amount          float64 `json:"amount"`
	Name            string  `json:"name"` // payment datetime as string
	CardType        string  `json:"card_type,omitempty"`
	TransactionID   string  `json:"transaction_id,omitempty"`
}

// POSOrderData is the order payload passed to create_from_ui.
// Build Lines with NewPOSOrderLine and StatementIDs with NewPOSOrderPayment.
type POSOrderData struct {
	Name             string  `json:"name"`
	POSSessionID     int     `json:"pos_session_id"`
	SequenceNumber   int     `json:"sequence_number"`
	UserID           int     `json:"user_id"`
	PartnerID        any     `json:"partner_id"` // int or false
	CreationDate     string  `json:"creation_date"`
	FiscalPositionID any     `json:"fiscal_position_id"` // int or false
	PricelistID      int     `json:"pricelist_id"`
	AmountPaid       float64 `json:"amount_paid"`
	AmountTotal      float64 `json:"amount_total"`
	AmountTax        float64 `json:"amount_tax"`
	AmountReturn     float64 `json:"amount_return"`
	ToInvoice        bool    `json:"to_invoice"`
	ToShip           bool    `json:"to_ship"`
	IsTipped         bool    `json:"is_tipped"`
	TipAmount        float64 `json:"tip_amount"`
	AccessToken      string  `json:"access_token"`
	ServerID         int     `json:"server_id,omitempty"` // set when resuming a draft order
	Lines            []any   `json:"lines"`               // []any{NewPOSOrderLine(...)}
	StatementIDs     []any   `json:"statement_ids"`       // []any{NewPOSOrderPayment(...)}
}

// POSOrderResult is a single entry returned by create_from_ui.
type POSOrderResult struct {
	ID           int    `json:"id"`
	PosReference string `json:"pos_reference"`
	AccountMove  any    `json:"account_move"` // int ID or false
}

// NewPOSOrderLine wraps line data in Odoo's [0, 0, data] One2many create command.
func NewPOSOrderLine(data POSOrderLineData) []any {
	return []any{0, 0, data}
}

// NewPOSOrderPayment wraps payment data in Odoo's [0, 0, data] One2many create command.
func NewPOSOrderPayment(data POSOrderPaymentData) []any {
	return []any{0, 0, data}
}

// TaxIDs wraps tax IDs in Odoo's [6, 0, ids] Many2many replace command,
// ready to assign to POSOrderLineData.TaxIDs.
func TaxIDs(ids ...int) []any {
	return []any{[]any{6, 0, ids}}
}

type posOrderWrapper struct {
	Data POSOrderData `json:"data"`
}

type createFromUIKWArgs struct {
	Draft bool `json:"draft"`
}

type refundResult struct {
	ResID int `json:"res_id"`
}

// CreatePOSOrder calls /web/dataset/call_kw/pos.order/create_from_ui.
// Set draft=true to park the order without payment; false to finalise and move stock.
// The call is idempotent: retrying with the same pos_reference is safe.
func (o *odooClientImpl) CreatePOSOrder(orders []POSOrderData, draft bool) ([]POSOrderResult, error) {
	wrappers := make([]posOrderWrapper, len(orders))
	for i, order := range orders {
		wrappers[i] = posOrderWrapper{Data: order}
	}
	return callKw[[]POSOrderResult](o, "pos.order", "create_from_ui", []any{wrappers}, createFromUIKWArgs{Draft: draft})
}

// WritePOSOrder calls /web/dataset/call_kw/pos.order/write.
// Only draft orders can be cancelled (set state:"cancel").
// Paid or invoiced orders must be reversed via RefundPOSOrder.
func (o *odooClientImpl) WritePOSOrder(ids []int, fields map[string]any) error {
	_, err := callKw[bool](o, "pos.order", "write", []any{ids, fields}, struct{}{})
	return err
}

// RefundPOSOrder calls /web/dataset/call_kw/pos.order/refund.
// Returns the ID of the new negative (refund) order.
// Requires an open session on the same POS config as the original order.
func (o *odooClientImpl) RefundPOSOrder(orderID int) (int, error) {
	result, err := callKw[refundResult](o, "pos.order", "refund", []any{}, map[string]any{
		"context": map[string]any{"active_ids": []int{orderID}},
	})
	if err != nil {
		return 0, err
	}
	return result.ResID, nil
}
