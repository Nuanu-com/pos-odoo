package odoo_test

import (
	odoo "github.com/Nuanu-com/pos-odoo"
	"github.com/Nuanu-com/pos-odoo/suite"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("OdooClient — AuthenticatedUserID", func() {
	It("returns 0 before any authentication has occurred", func() {
		client := odoo.NewOdooClient(nil, "http://localhost:8069")

		Expect(client.AuthenticatedUserID()).To(Equal(0))
	})

	It("returns the uid captured during re-authentication", func() {
		// Cassette: session expires → re-auth (uid: 2) → retry succeeds
		httpClient := suite.UseCassette(GinkgoT(), "odoo_session_expired_reauth")
		client := odoo.NewOdooClient(httpClient, "http://localhost:8069",
			odoo.AddAuthentication(func() (string, string, string) {
				return "admin", "admin", "mydb"
			}),
		)

		_, _ = client.SearchReadAccountTax(nil, nil, 80, 0)

		Expect(client.AuthenticatedUserID()).To(Equal(2))
	})
})
