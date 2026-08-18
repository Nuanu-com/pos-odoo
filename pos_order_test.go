package odoo_test

import (
	odoo "github.com/Nuanu-com/pos-odoo"

	"github.com/Nuanu-com/pos-odoo/suite"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var sampleOrder = odoo.POSOrderData{
	Name:             "Order 00001-001-0001",
	POSSessionID:     5,
	UserID:           3,
	PartnerID:        false,
	CreationDate:     "2024-06-19T10:30:00",
	FiscalPositionID: false,
	PricelistID:      1,
	AmountPaid:       15.00,
	AmountTotal:      15.00,
	AmountTax:        1.36,
	AmountReturn:     0.00,
	ToInvoice:        false,
	ToShip:           false,
	IsTipped:         false,
	TipAmount:        0,
	AccessToken:      "",
	Lines: []any{
		odoo.NewPOSOrderLine(odoo.POSOrderLineData{
			ProductID:         101,
			Qty:               2,
			PriceUnit:         7.50,
			PriceSubtotal:     15.00,
			PriceSubtotalIncl: 15.00,
			Discount:          0,
			TaxIDs:            odoo.TaxIDs(5),
			PackLotIDs:        []any{},
		}),
	},
	StatementIDs: []any{
		odoo.NewPOSOrderPayment(odoo.POSOrderPaymentData{
			PaymentMethodID: 1,
			Amount:          15.00,
			Name:            "2024-06-19 10:30:00",
		}),
	},
}

var _ = Describe("OdooClient — pos.order", func() {

	Describe("CreatePOSOrder", func() {
		Context("with a finalised order (draft=false)", func() {
			It("returns the created order ID and reference", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_create_pos_order_success")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				results, err := client.CreatePOSOrder([]odoo.POSOrderData{sampleOrder}, false)

				Expect(err).To(BeNil())
				Expect(results).To(HaveLen(1))
				Expect(results[0].ID).To(Equal(42))
				Expect(results[0].PosReference).To(Equal("Order 00001-001-0001"))
				Expect(results[0].AccountMove).To(BeFalse())
			})
		})

		Context("when there is no active session", func() {
			It("returns the RPC error from Odoo", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_create_pos_order_rpc_error")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				results, err := client.CreatePOSOrder([]odoo.POSOrderData{sampleOrder}, false)

				Expect(results).To(BeNil())
				Expect(err).To(MatchError("No cash statement found for this session. Unable to record returned cash."))
			})
		})
	})

	Describe("WritePOSOrder", func() {
		Context("cancelling a draft order", func() {
			It("returns no error", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_write_pos_order_success")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				err := client.WritePOSOrder([]int{42}, map[string]any{"state": "cancel"})

				Expect(err).To(BeNil())
			})
		})

		Context("when trying to cancel a paid order", func() {
			It("returns the RPC error from Odoo", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_write_pos_order_rpc_error")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				err := client.WritePOSOrder([]int{42}, map[string]any{"state": "cancel"})

				Expect(err).To(MatchError("You cannot cancel a paid order. Create a refund instead."))
			})
		})
	})

	Describe("RefundPOSOrder", func() {
		Context("with a valid paid order", func() {
			It("returns the new refund order ID", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_refund_pos_order_success")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				refundID, err := client.RefundPOSOrder(42)

				Expect(err).To(BeNil())
				Expect(refundID).To(Equal(43))
			})
		})

		Context("when there is no open session for the refund", func() {
			It("returns the RPC error from Odoo", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_refund_pos_order_rpc_error")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				refundID, err := client.RefundPOSOrder(42)

				Expect(refundID).To(Equal(0))
				Expect(err).To(MatchError("No open POS session found for this configuration."))
			})
		})
	})
})
