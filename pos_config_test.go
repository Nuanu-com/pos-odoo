package odoo_test

import (
	odoo "github.com/Nuanu-com/pos-odoo"
	"github.com/Nuanu-com/pos-odoo/suite"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("OdooClient — pos.config", func() {

	Describe("ReadPOSConfig", func() {
		Context("with a valid config ID", func() {
			It("returns the config with pricelist and payment method IDs", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_read_pos_config_success")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				config, err := client.ReadPOSConfig(1)

				Expect(err).To(BeNil())
				Expect(config.ID).To(Equal(1))
				Expect(config.Name).To(Equal("Shop"))
				Expect(config.PricelistID.ID).To(Equal(2))
				Expect(config.PricelistID.Name).To(Equal("Public Pricelist"))
				Expect(config.PaymentMethodIDs).To(ConsistOf(1, 3))
			})
		})

		Context("when the config does not exist", func() {
			It("returns a not found error", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_read_pos_config_not_found")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				config, err := client.ReadPOSConfig(9999)

				Expect(config).To(BeNil())
				Expect(err).To(MatchError("odoo: pos.config 9999 not found"))
			})
		})
	})
})
