package odoo

import (
	"encoding/json"
	"fmt"
)

// OdooMany2one represents a Many2one field returned as [id, "name"] or false.
type OdooMany2one struct {
	ID   int
	Name string
}

func (m *OdooMany2one) UnmarshalJSON(data []byte) error {
	s := string(data)
	if s == "false" || s == "null" {
		return nil
	}
	var pair []json.RawMessage
	if err := json.Unmarshal(data, &pair); err != nil {
		return err
	}
	if len(pair) < 2 {
		return nil
	}
	if err := json.Unmarshal(pair[0], &m.ID); err != nil {
		return err
	}
	return json.Unmarshal(pair[1], &m.Name)
}

// ProductProduct represents a product.product record (variant level).
type ProductProduct struct {
	ID            int          `json:"id"`
	Name          string       `json:"name"`
	DisplayName   string       `json:"display_name"`
	ListPrice     float64      `json:"list_price"`
	StandardPrice float64      `json:"standard_price"`
	CategID       OdooMany2one `json:"categ_id"`
	UomID         OdooMany2one `json:"uom_id"`
	ProductTmplID OdooMany2one `json:"product_tmpl_id"`
	Active        bool         `json:"active"`
}

// ProductTemplateInput holds the fields for creating a product.template.
type ProductTemplateInput struct {
	Name           string  `json:"name"`
	ListPrice      float64 `json:"list_price"`
	StandardPrice  float64 `json:"standard_price,omitempty"`
	Type           string  `json:"type"` // "consu", "service", "product"
	AvailableInPos bool    `json:"available_in_pos"`
	CategID        int     `json:"categ_id,omitempty"`
	UomID          int     `json:"uom_id,omitempty"`
	UomPoID        int     `json:"uom_po_id,omitempty"`
}

// ProductProductInput holds the fields for creating a product.product directly.
// Unlike CreateProductTemplate, this creates the variant record itself instead
// of letting Odoo auto-generate it from a product.template.
type ProductProductInput struct {
	Name           string  `json:"name"`
	ListPrice      float64 `json:"list_price"`
	StandardPrice  float64 `json:"standard_price,omitempty"`
	Type           string  `json:"type"` // "consu", "service", "product"
	AvailableInPos bool    `json:"available_in_pos"`
	CategID        int     `json:"categ_id,omitempty"`
	UomID          int     `json:"uom_id,omitempty"`
	UomPoID        int     `json:"uom_po_id,omitempty"`
	PosCategID     int     `json:"pos_categ_id,omitempty"`
	Barcode        string  `json:"barcode,omitempty"`
}

// Ptr returns a pointer to v, for setting individual ProductWriteInput fields.
func Ptr[T any](v T) *T {
	return &v
}

// ProductWriteInput holds the fields that can be updated on a product template
// or product variant via WriteProductTemplate/WriteProduct. Leave a field nil
// to leave it unchanged — only non-nil fields are sent to Odoo.
type ProductWriteInput struct {
	Name           *string  `json:"name,omitempty"`
	ListPrice      *float64 `json:"list_price,omitempty"`
	StandardPrice  *float64 `json:"standard_price,omitempty"`
	Type           *string  `json:"type,omitempty"`
	AvailableInPos *bool    `json:"available_in_pos,omitempty"`
	CategID        *int     `json:"categ_id,omitempty"`
	UomID          *int     `json:"uom_id,omitempty"`
	UomPoID        *int     `json:"uom_po_id,omitempty"`
	PosCategID     *int     `json:"pos_categ_id,omitempty"`
	Barcode        *string  `json:"barcode,omitempty"`
}

var defaultProductProductFields = []string{
	"id", "name", "list_price", "standard_price", "categ_id", "uom_id", "active",
}

var defaultProductVariantFields = []string{
	"id", "display_name", "categ_id",
}

type readKWArgs struct {
	Fields []string `json:"fields"`
}

// searchReadKWArgs are the kwargs for a plain search_read call. Unlike
// web_search_read (which returns {records, length}), search_read returns a bare
// array, so limit/offset/order are omitted when unset.
type searchReadKWArgs struct {
	Domain []any    `json:"domain"`
	Fields []string `json:"fields"`
	Limit  int      `json:"limit,omitempty"`
	Offset int      `json:"offset,omitempty"`
	Order  string   `json:"order,omitempty"`
}

// CreateProductTemplate calls /web/dataset/call_kw/product.template/create.
// Returns the ID of the newly created record. Odoo automatically creates the
// corresponding product.product variant.
func (o *odooClientImpl) CreateProductTemplate(input ProductTemplateInput) (int, error) {
	return callKw[int](o, "product.template", "create", []any{input}, struct{}{})
}

// CreateProduct calls /web/dataset/call_kw/product.product/create.
// Returns the ID of the newly created product.product record.
func (o *odooClientImpl) CreateProduct(input ProductProductInput) (int, error) {
	return callKw[int](o, "product.product", "create", []any{input}, struct{}{})
}

// WriteProductTemplate calls /web/dataset/call_kw/product.template/write.
// Leave fields nil in input to leave them unchanged.
func (o *odooClientImpl) WriteProductTemplate(ids []int, input ProductWriteInput) error {
	_, err := callKw[bool](o, "product.template", "write", []any{ids, input}, struct{}{})
	return err
}

// WriteProduct calls /web/dataset/call_kw/product.product/write.
// Leave fields nil in input to leave them unchanged.
func (o *odooClientImpl) WriteProduct(ids []int, input ProductWriteInput) error {
	_, err := callKw[bool](o, "product.product", "write", []any{ids, input}, struct{}{})
	return err
}

// SearchReadProduct calls /web/dataset/call_kw/product.product/search_read.
// Pass nil domain for no filter, nil fields for the variant field set
// (id, display_name, categ_id). limit/offset/order are omitted when zero/empty.
func (o *odooClientImpl) SearchReadProduct(domain []any, fields []string, limit, offset int) ([]ProductProduct, error) {
	if domain == nil {
		domain = []any{}
	}
	if len(fields) == 0 {
		fields = defaultProductVariantFields
	}
	kwargs := searchReadKWArgs{
		Domain: domain,
		Fields: fields,
		Limit:  limit,
		Offset: offset,
	}
	return callKw[[]ProductProduct](o, "product.product", "search_read", []any{}, kwargs)
}

// SearchReadProductsByTemplateID returns the product.product variants belonging
// to a product.template — used to resolve the variant Odoo auto-creates for a
// template. Pass nil fields to use the variant field set.
func (o *odooClientImpl) SearchReadProductsByTemplateID(templateID int, fields []string) ([]ProductProduct, error) {
	domain := []any{[]any{"product_tmpl_id", "=", templateID}}
	return o.SearchReadProduct(domain, fields, 0, 0)
}

// ReadProductByID calls /web/dataset/call_kw/product.product/read for a single ID.
// Pass nil fields to use the default field set.
func (o *odooClientImpl) ReadProductByID(id int, fields []string) (*ProductProduct, error) {
	if len(fields) == 0 {
		fields = defaultProductProductFields
	}
	records, err := callKw[[]ProductProduct](o, "product.product", "read", []any{[]int{id}}, readKWArgs{Fields: fields})
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("odoo: product %d not found", id)
	}
	return &records[0], nil
}
