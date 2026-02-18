package samgov_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSamgov(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SAM.gov Suite")
}
