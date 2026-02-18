package main

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestReconcileHandler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Reconcile Handler Suite")
}
