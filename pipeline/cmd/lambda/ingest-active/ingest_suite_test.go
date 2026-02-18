package main

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestIngestActiveHandler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ingest Active Handler Suite")
}
