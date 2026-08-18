package odoo_test

import (
	"github.com/Nuanu-com/pos-odoo/suite"

	odoo "github.com/Nuanu-com/pos-odoo"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("OdooClient", func() {
	odooUsername := "username"
	odooPassword := "password"
	odooDB := "woodenfish"
	odooUrl := "http://localhost:8069"

	Describe("SearchReadAccountTax", func() {
		Context("with a successful response", func() {
			It("returns the records and total length", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_search_read_account_tax_success")

				auth := odoo.AddAuthentication(func() (username string, password string, db string) {
					username = odooUsername
					password = odooPassword
					db = odooDB

					return username, password, db
				})

				client := odoo.NewOdooClient(httpClient, odooUrl, auth)

				result, err := client.SearchReadAccountTax(nil, nil, 80, 0)

				Expect(err).To(BeNil())
				Expect(result.Length).To(Equal(40))
				Expect(result.Records).To(HaveLen(40))

				first := result.Records[0]
				Expect(first.ID).To(Equal(1))
				Expect(first.Name).To(Equal("PPN-S Non Luxury Good (2025)"))
				Expect(first.Amount).To(Equal(11.0))
				Expect(first.AmountType).To(Equal("percent"))
				Expect(first.TypeTaxUse).To(Equal("sale"))
				Expect(first.PriceInclude).To(BeFalse())
				Expect(first.Active).To(BeTrue())
				Expect(first.Description).To(Equal("PPN (Non Luxury Goods)"))
			})
		})

		Context("when Odoo returns an RPC error", func() {
			It("returns the error message from Odoo", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_search_read_account_tax_rpc_error")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				result, err := client.SearchReadAccountTax(nil, nil, 80, 0)

				Expect(result).To(BeNil())
				Expect(err).To(MatchError("You do not have access to this document."))
			})
		})
	})

	Describe("ComputeAllTax", func() {
		Context("with a successful response", func() {
			It("returns the tax breakdown and totals", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_compute_all_tax_success")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				result, err := client.ComputeAllTax([]int{384}, 987, 1)

				Expect(err).To(BeNil())
				Expect(result.TotalExcluded).To(Equal(889.19))
				Expect(result.TotalIncluded).To(Equal(987.0))
				Expect(result.Taxes).To(HaveLen(1))
				Expect(result.Taxes[0].ID).To(Equal(384))
				Expect(result.Taxes[0].Name).To(Equal("PPN-S Non Luxury Good (2025)"))
				Expect(result.Taxes[0].Amount).To(Equal(97.81))
				Expect(result.Taxes[0].Base).To(Equal(889.19))
			})
		})

		Context("when quantity is not provided", func() {
			It("defaults to 1", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_compute_all_tax_success")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				result, err := client.ComputeAllTax([]int{384}, 987, 0)

				Expect(err).To(BeNil())
				Expect(result.TotalIncluded).To(Equal(987.0))
			})
		})

		Context("when Odoo returns an RPC error", func() {
			It("returns the error message from Odoo", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_compute_all_tax_rpc_error")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				result, err := client.ComputeAllTax([]int{9999}, 987, 1)

				Expect(result).To(BeNil())
				Expect(err).To(MatchError("Record does not exist or has been deleted."))
			})
		})
	})
})
