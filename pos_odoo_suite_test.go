package odoo_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPosOdoo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PosOdoo Suite")
}
