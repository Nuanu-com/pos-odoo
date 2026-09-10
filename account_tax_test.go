package odoo_test

import (
	"io"
	"net/http"
	"strings"

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

		// A refund line is a negative quantity at the price it sold for. Odoo computes the
		// base as price times quantity, so it is the quantity that has to carry the sign all
		// the way to the wire: clamping it to 1 answers for a sale of one unit instead.
		Context("when quantity is negative", func() {
			It("sends the negative quantity through untouched", func() {
				var sent string
				httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					body, err := io.ReadAll(r.Body)
					if err != nil {
						return nil, err
					}
					sent = string(body)
					return jsonResponse(`{"jsonrpc":"2.0","id":1,"result":` +
						`{"total_excluded":-1778.38,"total_included":-1974.0,"taxes":[]}}`), nil
				})}
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				result, err := client.ComputeAllTax([]int{384}, 987, -2)

				Expect(err).To(BeNil())
				Expect(sent).To(ContainSubstring(`"quantity":-2`))
				Expect(result.TotalIncluded).To(Equal(-1974.0))
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

// roundTripFunc serves one canned reply and keeps the request, for the cases where the
// assertion is about what was sent rather than what came back.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     http.Header{"Content-Type": []string{"application/json; charset=utf-8"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
