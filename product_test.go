package odoo_test

import (
	odoo "github.com/Nuanu-com/pos-odoo"

	"github.com/Nuanu-com/pos-odoo/suite"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("OdooClient — product", func() {

	Describe("CreateProductTemplate", func() {
		input := odoo.ProductTemplateInput{
			Name:           "Iced Latte",
			ListPrice:      4.50,
			StandardPrice:  1.80,
			Type:           "consu",
			AvailableInPos: true,
			CategID:        1,
			UomID:          1,
			UomPoID:        1,
		}

		Context("with a successful response", func() {
			It("returns the new template ID", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_create_product_template_success")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				id, err := client.CreateProductTemplate(input)

				Expect(err).To(BeNil())
				Expect(id).To(Equal(55))
			})
		})

		Context("when Odoo returns an RPC error", func() {
			It("returns the error message from Odoo", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_create_product_template_rpc_error")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				id, err := client.CreateProductTemplate(input)

				Expect(id).To(Equal(0))
				Expect(err).To(MatchError("The operation cannot be completed: another record already exists with the same name."))
			})
		})
	})

	Describe("CreateProduct", func() {
		input := odoo.ProductProductInput{
			Name:           "Iced Latte",
			ListPrice:      4.50,
			StandardPrice:  1.80,
			Type:           "consu",
			AvailableInPos: true,
			CategID:        1,
			UomID:          1,
			UomPoID:        1,
			PosCategID:     2,
			Barcode:        "123456789",
		}

		Context("with a successful response", func() {
			It("returns the new product ID", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_create_product_success")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				id, err := client.CreateProduct(input)

				Expect(err).To(BeNil())
				Expect(id).To(Equal(77))
			})
		})

		Context("when Odoo returns an RPC error", func() {
			It("returns the error message from Odoo", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_create_product_rpc_error")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				id, err := client.CreateProduct(input)

				Expect(id).To(Equal(0))
				Expect(err).To(MatchError("The operation cannot be completed: another record already exists with the same barcode."))
			})
		})
	})

	Describe("WriteProductTemplate", func() {
		Context("with a successful response", func() {
			It("returns no error", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_write_product_template_success")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				err := client.WriteProductTemplate([]int{55}, odoo.ProductWriteInput{
					ListPrice: odoo.Ptr(5.00),
					Name:      odoo.Ptr("Iced Latte (Large)"),
				})

				Expect(err).To(BeNil())
			})
		})

		Context("when Odoo returns an RPC error", func() {
			It("returns the error message from Odoo", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_write_product_template_rpc_error")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				err := client.WriteProductTemplate([]int{55}, odoo.ProductWriteInput{
					ListPrice: odoo.Ptr(-1.0),
				})

				Expect(err).To(MatchError("Unable to modify this PoS Configuration because you can't modify Payment Methods while a session is open."))
			})
		})
	})

	Describe("WriteProduct", func() {
		Context("with a successful response", func() {
			It("returns no error", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_write_product_success")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				err := client.WriteProduct([]int{77}, odoo.ProductWriteInput{
					Barcode:   odoo.Ptr("987654321"),
					ListPrice: odoo.Ptr(5.00),
				})

				Expect(err).To(BeNil())
			})
		})

		Context("when Odoo returns an RPC error", func() {
			It("returns the error message from Odoo", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_write_product_rpc_error")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				err := client.WriteProduct([]int{77}, odoo.ProductWriteInput{
					ListPrice: odoo.Ptr(-1.0),
				})

				Expect(err).To(MatchError("The operation cannot be completed: list_price must be positive."))
			})
		})
	})

	Describe("SearchReadProductsByTemplateID", func() {
		Context("with a successful response", func() {
			It("returns the variants of the template", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_search_read_product_by_template_success")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				products, err := client.SearchReadProductsByTemplateID(55, nil)

				Expect(err).To(BeNil())
				Expect(products).To(HaveLen(1))
				Expect(products[0].ID).To(Equal(77))
				Expect(products[0].DisplayName).To(Equal("[TICKET] Iced Latte"))
				Expect(products[0].CategID.ID).To(Equal(5))
				Expect(products[0].CategID.Name).To(Equal("Beverages"))
			})
		})

		Context("when Odoo returns an RPC error", func() {
			It("returns the error message from Odoo", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_search_read_product_rpc_error")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				products, err := client.SearchReadProductsByTemplateID(9999, nil)

				Expect(products).To(BeNil())
				Expect(err).To(MatchError("Invalid field product.product.product_tmpl_idx in leaf"))
			})
		})
	})

	Describe("ReadProductByID", func() {
		Context("with a successful response", func() {
			It("returns the product with decoded fields", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_read_product_by_id_success")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				product, err := client.ReadProductByID(42, nil)

				Expect(err).To(BeNil())
				Expect(product.ID).To(Equal(42))
				Expect(product.Name).To(Equal("Coffee"))
				Expect(product.ListPrice).To(Equal(3.50))
				Expect(product.StandardPrice).To(Equal(1.20))
				Expect(product.Active).To(BeTrue())
				Expect(product.CategID.ID).To(Equal(5))
				Expect(product.CategID.Name).To(Equal("Beverages"))
				Expect(product.UomID.ID).To(Equal(1))
				Expect(product.UomID.Name).To(Equal("Units"))
			})
		})

		Context("when the product does not exist", func() {
			It("returns a not found error", func() {
				httpClient := suite.UseCassette(GinkgoT(), "odoo_read_product_by_id_not_found")
				client := odoo.NewOdooClient(httpClient, "http://localhost:8069")

				product, err := client.ReadProductByID(9999, nil)

				Expect(product).To(BeNil())
				Expect(err).To(MatchError("odoo: product 9999 not found"))
			})
		})
	})
})
