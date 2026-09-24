package odoo

// AccountTax represents an account.tax record returned by Odoo.
type AccountTax struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	Amount       float64 `json:"amount"`
	AmountType   string  `json:"amount_type"`   // "percent", "fixed", "division", "group"
	TypeTaxUse   string  `json:"type_tax_use"`  // "sale", "purchase", "none"
	PriceInclude bool    `json:"price_include"` // true = tax is included in the listed price
	Active       bool    `json:"active"`
	Description  any     `json:"description"`
	// CompanyID is only populated when "company_id" is requested in fields.
	CompanyID *OdooMany2one `json:"company_id,omitempty"`
}

var defaultAccountTaxFields = []string{
	"id", "name", "amount", "amount_type", "type_tax_use",
	"price_include", "active", "description",
}

type webSearchReadKWArgs struct {
	Domain []any    `json:"domain"`
	Fields []string `json:"fields"`
	Limit  int      `json:"limit"`
	Offset int      `json:"offset"`
	Order  string   `json:"order"`
	// Context is omitted when nil, so callers that don't set it send the same
	// payload as before it existed.
	Context map[string]any `json:"context,omitempty"`
}

// SearchReadAccountTaxOptions configures SearchReadAccountTaxWithOptions.
type SearchReadAccountTaxOptions struct {
	Domain []any    // nil for no filter
	Fields []string // nil for the default field set
	Limit  int      // 0 for no limit
	Offset int
	// Order is an Odoo order clause such as "id asc". Empty uses the model's
	// default order.
	Order string
	// Context is passed through as the call's context, e.g.
	// {"allowed_company_ids": []int{1}} to act within a specific company.
	Context map[string]any
}

// SearchReadAccountTax calls /web/dataset/call_kw/account.tax/web_search_read.
// Pass nil domain for no filter, nil fields for the default field set.
func (o *odooClientImpl) SearchReadAccountTax(domain []any, fields []string, limit, offset int) (*SearchReadResult[AccountTax], error) {
	return o.SearchReadAccountTaxWithOptions(SearchReadAccountTaxOptions{
		Domain: domain,
		Fields: fields,
		Limit:  limit,
		Offset: offset,
	})
}

// SearchReadAccountTaxWithOptions is SearchReadAccountTax with control over the
// order and the call context.
func (o *odooClientImpl) SearchReadAccountTaxWithOptions(opts SearchReadAccountTaxOptions) (*SearchReadResult[AccountTax], error) {
	domain := opts.Domain
	if domain == nil {
		domain = []any{}
	}
	fields := opts.Fields
	if len(fields) == 0 {
		fields = defaultAccountTaxFields
	}
	kwargs := webSearchReadKWArgs{
		Domain:  domain,
		Fields:  fields,
		Limit:   opts.Limit,
		Offset:  opts.Offset,
		Order:   opts.Order,
		Context: opts.Context,
	}
	result, err := callKw[SearchReadResult[AccountTax]](o, "account.tax", "web_search_read", []any{}, kwargs)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// AccountTaxComputeAllLine is a single tax line in the breakdown returned by compute_all.
type AccountTaxComputeAllLine struct {
	ID     int     `json:"id"`
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
	Base   float64 `json:"base"`
}

// AccountTaxComputeAllResult is returned by account.tax/compute_all.
type AccountTaxComputeAllResult struct {
	TotalExcluded float64                    `json:"total_excluded"`
	TotalIncluded float64                    `json:"total_included"`
	Taxes         []AccountTaxComputeAllLine `json:"taxes"`
}

type computeAllTaxKWArgs struct {
	Quantity float64 `json:"quantity"`
}

// ComputeAllTax calls /web/dataset/call_kw/account.tax/compute_all.
// It mirrors Odoo's own tax computation for priceUnit, returning the per-tax
// base/amount breakdown plus totals excluding/including tax. quantity defaults
// to 1 when it is zero, which is how an unset quantity arrives.
//
// A negative quantity is passed through. Odoo computes the base as priceUnit
// times quantity, so a refund line of minus one comes back with negative totals
// — which is how Odoo writes a refund itself. Defaulting those to 1 would answer
// for a sale of one unit instead, positive and, above one unit, the wrong size.
func (o *odooClientImpl) ComputeAllTax(taxIDs []int, priceUnit float64, quantity float64) (*AccountTaxComputeAllResult, error) {
	if quantity == 0 {
		quantity = 1
	}
	result, err := callKw[AccountTaxComputeAllResult](o, "account.tax", "compute_all", []any{taxIDs, priceUnit}, computeAllTaxKWArgs{Quantity: quantity})
	if err != nil {
		return nil, err
	}
	return &result, nil
}
