package odoo_test

import (
	odoo "github.com/Nuanu-com/pos-odoo"
	"github.com/Nuanu-com/pos-odoo/suite"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("OdooClient — automatic re-authentication", func() {
	Context("when the session expires mid-flight", func() {
		It("re-authenticates and retries the call transparently", func() {
			// Cassette has 3 interactions in order:
			//   0. POST /web/dataset/call_kw/…/web_search_read → SessionExpiredException
			//   1. POST /web/session/authenticate              → uid: 2  (automatic re-auth)
			//   2. POST /web/dataset/call_kw/…/web_search_read → records (success)
			httpClient := suite.UseCassette(GinkgoT(), "odoo_session_expired_reauth")
			client := odoo.NewOdooClient(httpClient, "http://localhost:8069",
				odoo.AddAuthentication(func() (string, string, string) {
					return "admin", "admin", "mydb"
				}),
			)

			result, err := client.SearchReadAccountTax(nil, nil, 80, 0)

			Expect(err).To(BeNil())
			Expect(result.Length).To(Equal(1))
			Expect(result.Records[0].Name).To(Equal("Tax 15%"))
		})
	})

	Context("when re-authentication itself fails", func() {
		It("returns the original session-expired error", func() {
			// Cassette has 2 interactions in order:
			//   0. POST /web/dataset/call_kw/…/web_search_read → SessionExpiredException
			//   1. POST /web/session/authenticate              → uid: false (bad credentials)
			httpClient := suite.UseCassette(GinkgoT(), "odoo_session_expired_reauth_failed")
			client := odoo.NewOdooClient(httpClient, "http://localhost:8069",
				odoo.AddAuthentication(func() (string, string, string) {
					return "admin", "admin", "mydb"
				}),
			)

			result, err := client.SearchReadAccountTax(nil, nil, 80, 0)

			Expect(result).To(BeNil())
			Expect(err).To(MatchError("Session Expired"))
		})
	})
})
