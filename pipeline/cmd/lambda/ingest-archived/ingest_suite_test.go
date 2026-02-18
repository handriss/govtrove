package main

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestIngestArchivedHandler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ingest Archived Handler Suite")
}
