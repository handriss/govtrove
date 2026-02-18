package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/handriss/govtrove/pipeline/internal/database"
	"github.com/handriss/govtrove/pipeline/internal/testutil"
)

func gzipCSV(csv string) io.ReadCloser {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	gz.Write([]byte(csv))
	gz.Close()
	return io.NopCloser(&buf)
}

type mockS3 struct {
	GetObjectFn func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

func (m *mockS3) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if m.GetObjectFn != nil {
		return m.GetObjectFn(ctx, params, optFns...)
	}
	return nil, errors.New("not implemented")
}

var _ = Describe("Ingest Archived Handler", func() {
	var (
		h      *Handler
		store  *testutil.MockStore
		s3mock *mockS3
		ctx    context.Context
	)

	csvData := "NoticeId,Title,Type,Active\n" +
		"ARCH-001,Archived Opp,Award,No\n" +
		"ARCH-002,Another Archived,Award,No\n"

	BeforeEach(func() {
		ctx = context.Background()
		store = &testutil.MockStore{}
		s3mock = &mockS3{}
		h = &Handler{
			Store:  store,
			S3:     s3mock,
			Bucket: "test-bucket",
			Logger: slog.Default(),
		}
	})

	buildEvent := func(s3Key string) json.RawMessage {
		input := Input{PipelineRunID: uuid.New().String()}
		input.File.Type = "archived"
		input.File.S3Key = s3Key
		input.File.Source = "archived_fy2026"
		b, _ := json.Marshal(input)
		return b
	}

	Context("processing a new archived CSV file", func() {
		BeforeEach(func() {
			s3mock.GetObjectFn = func(_ context.Context, _ *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
				return &s3.GetObjectOutput{Body: gzipCSV(csvData)}, nil
			}
		})

		It("creates an ingestion run, parses CSV, and bulk inserts", func() {
			runID := uuid.New()
			store.CreateIngestionRunFn = func(_ context.Context, jt string) (uuid.UUID, error) {
				Expect(jt).To(Equal("ingest-archived"))
				return runID, nil
			}

			var insertedCount int
			store.BulkInsertSnapCSVFn = func(_ context.Context, _ uuid.UUID, _ time.Time, _ int64, rows []database.SnapCSVRow) (int64, error) {
				insertedCount = len(rows)
				return int64(len(rows)), nil
			}

			output, err := h.Handle(ctx, buildEvent("raw/archived-csv/FY2026/2026-02-18.csv.gz"))

			Expect(err).NotTo(HaveOccurred())
			Expect(output.Status).To(Equal("ok"))
			Expect(output.RunID).To(Equal(runID.String()))
			Expect(output.JobType).To(Equal("ingest-archived"))
			Expect(insertedCount).To(Equal(2))
		})

		It("does NOT perform change detection", func() {
			detectCalled := false
			store.DetectChangesFn = func(_ context.Context, _, _ uuid.UUID, _ time.Time, _ *slog.Logger) (int, int, error) {
				detectCalled = true
				return 0, 0, nil
			}

			output, err := h.Handle(ctx, buildEvent("raw/archived-csv/FY2026/2026-02-18.csv.gz"))

			Expect(err).NotTo(HaveOccurred())
			Expect(output.Status).To(Equal("ok"))
			Expect(detectCalled).To(BeFalse())
		})
	})

	Context("when S3 GetObject fails", func() {
		It("marks the ingestion run as failed and returns the error", func() {
			s3mock.GetObjectFn = func(_ context.Context, _ *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
				return nil, errors.New("access denied")
			}

			failCalled := false
			store.FailIngestionRunFn = func(_ context.Context, _ uuid.UUID, errMsg string, _ int) error {
				failCalled = true
				return nil
			}

			_, err := h.Handle(ctx, buildEvent("raw/archived-csv/FY2026/2026-02-18.csv.gz"))

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("access denied"))
			Expect(failCalled).To(BeTrue())
		})
	})

	Context("when bulk insert fails", func() {
		It("fails the CSV download entry and the ingestion run", func() {
			s3mock.GetObjectFn = func(_ context.Context, _ *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
				return &s3.GetObjectOutput{Body: gzipCSV(csvData)}, nil
			}
			store.BulkInsertSnapCSVFn = func(_ context.Context, _ uuid.UUID, _ time.Time, _ int64, _ []database.SnapCSVRow) (int64, error) {
				return 0, errors.New("disk full")
			}

			downloadFailed := false
			store.FailCSVDownloadEntryFn = func(_ context.Context, _ int64, errMsg string) error {
				downloadFailed = true
				return nil
			}

			_, err := h.Handle(ctx, buildEvent("raw/archived-csv/FY2026/2026-02-18.csv.gz"))

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("bulk insert"))
			Expect(downloadFailed).To(BeTrue())
		})
	})
})
