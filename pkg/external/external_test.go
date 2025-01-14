package external

import (
	"testing"

	_ "github.com/joho/godotenv/autoload"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestExternal(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "External Suite")
}
