package odoo

import "fmt"

// POSConfig represents the fields we need from a pos.config record.
type POSConfig struct {
	ID               int          `json:"id"`
	Name             string       `json:"name"`
	PricelistID      OdooMany2one `json:"pricelist_id"`
	PaymentMethodIDs []int        `json:"payment_method_ids"`
}

var defaultPOSConfigFields = []string{
	"id", "name", "pricelist_id", "payment_method_ids",
}

// ReadPOSConfig calls /web/dataset/call_kw/pos.config/read for a single config ID.
func (o *odooClientImpl) ReadPOSConfig(configID int) (*POSConfig, error) {
	records, err := callKw[[]POSConfig](o, "pos.config", "read", []any{[]int{configID}}, readKWArgs{Fields: defaultPOSConfigFields})
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("odoo: pos.config %d not found", configID)
	}
	return &records[0], nil
}
