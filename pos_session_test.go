package odoo_test

import (
	odoo "github.com/Nuanu-com/pos-odoo"

	"github.com/Nuanu-com/pos-odoo/suite"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("OdooClient — pos.session", func() {

	Describe("ReadPOSSession", func() {
		Context("with a valid session ID", func() {
			It("returns the session's id, name, sequence_number, and cash register fields", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_read_pos_session_success")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				info, err := client.ReadPOSSession(5)

				Expect(err).To(BeNil())
				Expect(info.ID).To(Equal(5))
				Expect(info.Name).To(Equal("Main/0001"))
				Expect(info.SequenceNumber).To(Equal(42))
				Expect(info.CashRegisterTotalEntryEncoding).To(Equal(50_000.0))
				Expect(info.CashRegisterBalanceEnd).To(Equal(198_000.0))
				Expect(info.CashRegisterBalanceEndReal).To(Equal(0.0))
				Expect(info.CashRegisterDifference).To(Equal(-198_000.0))
			})
		})

		Context("when the session does not exist", func() {
			It("returns a not found error", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_read_pos_session_not_found")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				info, err := client.ReadPOSSession(9999)

				Expect(info).To(BeNil())
				Expect(err).To(MatchError("odoo: pos.session 9999 not found"))
			})
		})
	})

	Describe("CreatePOSSession", func() {
		Context("with a valid config ID", func() {
			It("returns the new session ID", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_create_pos_session_success")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				sessionID, err := client.CreatePOSSession(1)

				Expect(err).To(BeNil())
				Expect(sessionID).To(Equal(5))
			})
		})

		Context("when the config does not exist", func() {
			It("returns the RPC error from Odoo", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_create_pos_session_rpc_error")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				sessionID, err := client.CreatePOSSession(999)

				Expect(sessionID).To(Equal(0))
				Expect(err).To(MatchError("You should assign a Point of Sale to your session."))
			})
		})
	})

	Describe("ClosePOSSession", func() {
		Context("when the session closes successfully", func() {
			It("returns no error", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_close_pos_session_success")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				err := client.ClosePOSSession(5, nil)

				Expect(err).To(BeNil())
			})
		})

		Context("when there are draft orders blocking the close", func() {
			It("returns the message from Odoo as an error", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_close_pos_session_failed")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				err := client.ClosePOSSession(5, nil)

				Expect(err).To(MatchError("There are still orders in draft state in the session."))
			})
		})

		Context("with bank payment method diffs to post", func() {
			It("returns no error", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_close_pos_session_with_bank_diff_success")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				err := client.ClosePOSSession(5, []odoo.BankPaymentMethodDiff{
					{PaymentMethodID: 3, DiffAmount: -2.5},
				})

				Expect(err).To(BeNil())
			})
		})
	})

	Describe("PostClosingCashDetails", func() {
		Context("when the cash details are stored successfully", func() {
			It("returns no error", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_post_closing_cash_details_success")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				err := client.PostClosingCashDetails(5, 100)

				Expect(err).To(BeNil())
			})
		})

		Context("when the session has no cash register", func() {
			It("returns the message from Odoo as an error", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_post_closing_cash_details_failed")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				err := client.PostClosingCashDetails(5, 100)

				Expect(err).To(MatchError("There is no cash register in this session."))
			})
		})
	})

	Describe("TryCashInOut", func() {
		Context("when the cash move is recorded successfully", func() {
			It("returns no error", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_try_cash_in_out_success")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				err := client.TryCashInOut(5, odoo.CashMoveOut, 50_000, "Supplier payment - bakery")

				Expect(err).To(BeNil())
			})
		})

		Context("when the session has no cash payment method", func() {
			It("returns the RPC error from Odoo", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_try_cash_in_out_no_cash_method")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				err := client.TryCashInOut(5, odoo.CashMoveIn, 200_000, "Float top-up")

				Expect(err).To(MatchError("There is no cash payment method for this PoS Session"))
			})
		})

		Context("with an unknown move type", func() {
			It("returns an error without calling Odoo", func() {
				client := odoo.NewOdooClient(nil, "http://localhost:8069")

				err := client.TryCashInOut(5, odoo.CashMoveType("withdraw"), 50_000, "Bank drop")

				Expect(err).To(MatchError(`odoo: invalid cash move type "withdraw", want "in" or "out"`))
			})
		})

		Context("with a non-positive amount", func() {
			It("returns an error without calling Odoo", func() {
				client := odoo.NewOdooClient(nil, "http://localhost:8069")

				err := client.TryCashInOut(5, odoo.CashMoveOut, -50_000, "Bank drop")

				Expect(err).To(MatchError("odoo: cash move amount must be positive, got -50000"))
			})
		})
	})

	Describe("UpdateClosingControlStateSession", func() {
		Context("when the session transitions to closing_control successfully", func() {
			It("returns no error", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_update_closing_control_state_session_success")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				err := client.UpdateClosingControlStateSession(5, "")

				Expect(err).To(BeNil())
			})
		})

		Context("when the session is already closed", func() {
			It("returns the RPC error from Odoo", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_update_closing_control_state_session_already_closed")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				err := client.UpdateClosingControlStateSession(5, "")

				Expect(err).To(MatchError("This session is already closed."))
			})
		})
	})
})
