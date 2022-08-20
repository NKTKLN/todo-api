package tests

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestToDoAPI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ToDo Api Suite")
}
