package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
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

		// Default mocks for bulk_csv_log
		store.GetBulkCSVLogByS3KeyFn = func(_ context.Context, s3Key string) (*database.BulkCSVLogRecord, error) {
			return &database.BulkCSVLogRecord{
				ID:     42,
				Source: "archived",
				Result: "new_file",
				S3Key:  &s3Key,
			}, nil
		}
		store.UpdateBulkCSVLogIngestionFn = func(_ context.Context, _ int, _ uuid.UUID, _ int, _ string) error {
			return nil
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
			store.CreateIngestionRunFn = func(_ context.Context, jt string, _ *uuid.UUID) (uuid.UUID, error) {
				Expect(jt).To(Equal("ingest-archived"))
				return runID, nil
			}

			var insertedCount int
			store.BulkInsertSnapCSVFn = func(_ context.Context, _ uuid.UUID, _ time.Time, _ int64, rows []database.SnapCSVRow) (int64, error) {
				insertedCount += len(rows)
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

		It("updates bulk_csv_log with correct count and status on success", func() {
			var finalStatus string
			var finalCount int
			store.UpdateBulkCSVLogIngestionFn = func(_ context.Context, _ int, _ uuid.UUID, count int, status string) error {
				finalStatus = status
				finalCount = count
				return nil
			}

			_, err := h.Handle(ctx, buildEvent("raw/archived-csv/FY2026/2026-02-18.csv.gz"))

			Expect(err).NotTo(HaveOccurred())
			Expect(finalStatus).To(Equal("completed"))
			Expect(finalCount).To(Equal(2))
		})
	})

	Context("idempotency — file already ingested", func() {
		It("skips S3 download when bulk_csv_log status is completed", func() {
			completed := "completed"
			rowCount := 100
			store.GetBulkCSVLogByS3KeyFn = func(_ context.Context, s3Key string) (*database.BulkCSVLogRecord, error) {
				return &database.BulkCSVLogRecord{
					ID:       42,
					Source:   "archived",
					Result:   "new_file",
					S3Key:    &s3Key,
					Status:   &completed,
					RowCount: &rowCount,
				}, nil
			}

			s3Called := false
			s3mock.GetObjectFn = func(_ context.Context, _ *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
				s3Called = true
				return nil, errors.New("should not be called")
			}

			store.CreateIngestionRunFn = func(_ context.Context, _ string, _ *uuid.UUID) (uuid.UUID, error) {
				return uuid.New(), nil
			}

			output, err := h.Handle(ctx, buildEvent("raw/archived-csv/FY2026/2026-02-18.csv.gz"))

			Expect(err).NotTo(HaveOccurred())
			Expect(output.Status).To(Equal("ok"))
			Expect(s3Called).To(BeFalse())
		})
	})

	Context("missing bulk_csv_log record", func() {
		It("returns an error", func() {
			store.GetBulkCSVLogByS3KeyFn = func(_ context.Context, _ string) (*database.BulkCSVLogRecord, error) {
				return nil, nil
			}
			store.CreateIngestionRunFn = func(_ context.Context, _ string, _ *uuid.UUID) (uuid.UUID, error) {
				return uuid.New(), nil
			}

			_, err := h.Handle(ctx, buildEvent("raw/archived-csv/FY2026/2026-02-18.csv.gz"))

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no bulk_csv_log record"))
		})
	})

	Context("when GetBulkCSVLogByS3Key returns a DB error", func() {
		It("returns the error (distinct from nil record)", func() {
			store.GetBulkCSVLogByS3KeyFn = func(_ context.Context, _ string) (*database.BulkCSVLogRecord, error) {
				return nil, errors.New("connection timeout")
			}

			_, err := h.Handle(ctx, buildEvent("raw/archived-csv/FY2026/2026-02-18.csv.gz"))

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("connection timeout"))
		})
	})

	Context("when CreateIngestionRun fails", func() {
		It("returns the error without proceeding", func() {
			store.CreateIngestionRunFn = func(_ context.Context, _ string, _ *uuid.UUID) (uuid.UUID, error) {
				return uuid.Nil, errors.New("db connection refused")
			}

			_, err := h.Handle(ctx, buildEvent("raw/archived-csv/FY2026/2026-02-18.csv.gz"))

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("db connection refused"))
		})
	})

	Context("when S3 GetObject fails", func() {
		It("marks the ingestion run and bulk_csv_log as failed", func() {
			s3mock.GetObjectFn = func(_ context.Context, _ *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
				return nil, errors.New("access denied")
			}

			failCalled := false
			store.FailIngestionRunFn = func(_ context.Context, _ uuid.UUID, errMsg string, _ int) error {
				failCalled = true
				Expect(errMsg).To(ContainSubstring("access denied"))
				return nil
			}

			bulkLogFailed := false
			store.UpdateBulkCSVLogIngestionFn = func(_ context.Context, _ int, _ uuid.UUID, _ int, status string) error {
				if status == "failed" {
					bulkLogFailed = true
				}
				return nil
			}

			_, err := h.Handle(ctx, buildEvent("raw/archived-csv/FY2026/2026-02-18.csv.gz"))

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("access denied"))
			Expect(failCalled).To(BeTrue())
			Expect(bulkLogFailed).To(BeTrue())
		})
	})

	Context("when S3 returns corrupt (non-gzip) content", func() {
		It("marks bulk_csv_log as failed and returns the error", func() {
			s3mock.GetObjectFn = func(_ context.Context, _ *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
				return &s3.GetObjectOutput{
					Body: io.NopCloser(strings.NewReader("this is not gzip data")),
				}, nil
			}

			bulkLogFailed := false
			store.UpdateBulkCSVLogIngestionFn = func(_ context.Context, _ int, _ uuid.UUID, _ int, status string) error {
				if status == "failed" {
					bulkLogFailed = true
				}
				return nil
			}

			_, err := h.Handle(ctx, buildEvent("raw/archived-csv/FY2026/2026-02-18.csv.gz"))

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("gzip"))
			Expect(bulkLogFailed).To(BeTrue())
		})
	})

	Context("when bulk insert fails", func() {
		It("updates bulk_csv_log to failed and fails the ingestion run", func() {
			s3mock.GetObjectFn = func(_ context.Context, _ *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
				return &s3.GetObjectOutput{Body: gzipCSV(csvData)}, nil
			}
			store.BulkInsertSnapCSVFn = func(_ context.Context, _ uuid.UUID, _ time.Time, _ int64, _ []database.SnapCSVRow) (int64, error) {
				return 0, errors.New("disk full")
			}

			bulkLogFailed := false
			store.UpdateBulkCSVLogIngestionFn = func(_ context.Context, _ int, _ uuid.UUID, _ int, status string) error {
				if status == "failed" {
					bulkLogFailed = true
				}
				return nil
			}

			_, err := h.Handle(ctx, buildEvent("raw/archived-csv/FY2026/2026-02-18.csv.gz"))

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("bulk insert"))
			Expect(bulkLogFailed).To(BeTrue())
		})
	})

	Context("pipeline_run_id propagation", func() {
		BeforeEach(func() {
			s3mock.GetObjectFn = func(_ context.Context, _ *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
				return &s3.GetObjectOutput{Body: gzipCSV(csvData)}, nil
			}
		})

		It("passes pipeline_run_id to CreateIngestionRun", func() {
			pipelineRunID := uuid.New()

			var receivedPipelineRunID *uuid.UUID
			store.CreateIngestionRunFn = func(_ context.Context, _ string, prid *uuid.UUID) (uuid.UUID, error) {
				receivedPipelineRunID = prid
				return uuid.New(), nil
			}

			input := Input{PipelineRunID: pipelineRunID.String()}
			input.File.Type = "archived"
			input.File.S3Key = "raw/archived-csv/FY2026/2026-02-18.csv.gz"
			input.File.Source = "archived_fy2026"
			b, _ := json.Marshal(input)

			_, err := h.Handle(ctx, b)

			Expect(err).NotTo(HaveOccurred())
			Expect(receivedPipelineRunID).NotTo(BeNil())
			Expect(*receivedPipelineRunID).To(Equal(pipelineRunID))
		})

		It("passes nil when pipeline_run_id is invalid", func() {
			var receivedPipelineRunID *uuid.UUID
			store.CreateIngestionRunFn = func(_ context.Context, _ string, prid *uuid.UUID) (uuid.UUID, error) {
				receivedPipelineRunID = prid
				return uuid.New(), nil
			}

			input := Input{PipelineRunID: "not-a-uuid"}
			input.File.Type = "archived"
			input.File.S3Key = "raw/archived-csv/FY2026/2026-02-18.csv.gz"
			input.File.Source = "archived_fy2026"
			b, _ := json.Marshal(input)

			_, err := h.Handle(ctx, b)

			Expect(err).NotTo(HaveOccurred())
			Expect(receivedPipelineRunID).To(BeNil())
		})
	})

	Context("batching with large CSV", func() {
		It("flushes multiple batches when rows exceed batchSize", func() {
			var sb strings.Builder
			sb.WriteString("NoticeId,Title\n")
			for i := 0; i < batchSize+2; i++ {
				sb.WriteString(fmt.Sprintf("OPP-%05d,Title %d\n", i, i))
			}

			s3mock.GetObjectFn = func(_ context.Context, _ *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
				return &s3.GetObjectOutput{Body: gzipCSV(sb.String())}, nil
			}

			insertCalls := 0
			totalInserted := 0
			store.BulkInsertSnapCSVFn = func(_ context.Context, _ uuid.UUID, _ time.Time, _ int64, rows []database.SnapCSVRow) (int64, error) {
				insertCalls++
				totalInserted += len(rows)
				return int64(len(rows)), nil
			}

			output, err := h.Handle(ctx, buildEvent("raw/archived-csv/FY2026/2026-02-18.csv.gz"))

			Expect(err).NotTo(HaveOccurred())
			Expect(output.Status).To(Equal("ok"))
			Expect(insertCalls).To(Equal(2))
			Expect(totalInserted).To(Equal(batchSize + 2))
		})
	})
})
