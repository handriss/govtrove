package bulkcsv

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/handriss/govtrove/pipeline/internal/config"
	"github.com/handriss/govtrove/pipeline/internal/database"
)

type mockBulkStore struct {
	hashFn    func(ctx context.Context, source string) (string, error)
	headersFn func(ctx context.Context, source string) (string, string, error)
	s3KeyFn   func(ctx context.Context, source string) (string, error)
	logFn     func(ctx context.Context, r *database.BulkCSVLogRecord) (int, error)
}

func (m *mockBulkStore) GetLatestBulkCSVHash(ctx context.Context, source string) (string, error) {
	if m.hashFn != nil {
		return m.hashFn(ctx, source)
	}
	return "", nil
}

func (m *mockBulkStore) GetLatestBulkCSVHeaders(ctx context.Context, source string) (string, string, error) {
	if m.headersFn != nil {
		return m.headersFn(ctx, source)
	}
	return "", "", nil
}

func (m *mockBulkStore) GetLatestBulkCSVS3Key(ctx context.Context, source string) (string, error) {
	if m.s3KeyFn != nil {
		return m.s3KeyFn(ctx, source)
	}
	return "", nil
}

func (m *mockBulkStore) InsertBulkCSVLog(ctx context.Context, r *database.BulkCSVLogRecord) (int, error) {
	if m.logFn != nil {
		return m.logFn(ctx, r)
	}
	return 1, nil
}

type mockS3Client struct {
	putFn  func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	headFn func(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
}

func (m *mockS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	if m.putFn != nil {
		return m.putFn(ctx, params, optFns...)
	}
	return &s3.PutObjectOutput{}, nil
}

func (m *mockS3Client) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	if m.headFn != nil {
		return m.headFn(ctx, params, optFns...)
	}
	return &s3.HeadObjectOutput{}, nil
}

var _ = Describe("BulkCSV Download", func() {
	var (
		ctx       context.Context
		store     *mockBulkStore
		s3Client  *mockS3Client
		cfg       *config.Config
		origSrcs  []Source
	)

	BeforeEach(func() {
		ctx = context.Background()
		store = &mockBulkStore{}
		s3Client = &mockS3Client{}
		cfg = &config.Config{
			S3Bucket:         "test-bucket",
			S3ArchiveEnabled: false, // disable S3 upload by default to simplify tests
		}
		origSrcs = Sources
	})

	AfterEach(func() {
		Sources = origSrcs
	})

	Context("conditional GET returns 304 Not Modified", func() {
		It("reports not_modified outcome without downloading", func() {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("If-None-Match") == `"etag-123"` {
					w.WriteHeader(http.StatusNotModified)
					return
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			Sources = []Source{{Key: "test-active", URL: server.URL, S3Prefix: "raw/csv"}}
			store.headersFn = func(_ context.Context, _ string) (string, string, error) {
				return `"etag-123"`, "", nil
			}

			results, err := Run(ctx, cfg, store, s3Client, slog.Default())

			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Outcome).To(Equal("not_modified"))
		})
	})

	Context("downloaded file hash matches previous hash", func() {
		It("reports hash_match outcome without uploading to S3", func() {
			csvBody := "NoticeId,Title\nID-001,Test\n"

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, csvBody)
			}))
			defer server.Close()

			Sources = []Source{{Key: "test-active", URL: server.URL, S3Prefix: "raw/csv"}}

			// First download to get the hash
			results1, err := Run(ctx, cfg, store, s3Client, slog.Default())
			Expect(err).NotTo(HaveOccurred())
			Expect(results1).To(HaveLen(1))
			Expect(results1[0].Outcome).To(Equal("new_file"))

			// Now set the previous hash to match
			store.hashFn = func(_ context.Context, _ string) (string, error) {
				return results1[0].S3Key, nil // wrong value, let me use a capture
			}

			// Actually, let's compute the expected hash from the test
			// Run again with the hash from the first run captured differently
			var capturedHash string
			store.logFn = func(_ context.Context, r *database.BulkCSVLogRecord) (int, error) {
				if r.SHA256Hash != nil {
					capturedHash = *r.SHA256Hash
				}
				return 1, nil
			}

			// Re-run to capture the hash
			_, err = Run(ctx, cfg, store, s3Client, slog.Default())
			Expect(err).NotTo(HaveOccurred())
			Expect(capturedHash).NotTo(BeEmpty())

			// Now set the hash and run again
			store.hashFn = func(_ context.Context, _ string) (string, error) {
				return capturedHash, nil
			}

			results3, err := Run(ctx, cfg, store, s3Client, slog.Default())
			Expect(err).NotTo(HaveOccurred())
			Expect(results3).To(HaveLen(1))
			Expect(results3[0].Outcome).To(Equal("hash_match"))
		})
	})

	Context("file is new (no previous hash)", func() {
		It("downloads and reports new_file", func() {
			csvBody := "NoticeId,Title\nID-001,New File\n"

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("ETag", `"new-etag"`)
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, csvBody)
			}))
			defer server.Close()

			Sources = []Source{{Key: "test-active", URL: server.URL, S3Prefix: "raw/csv"}}

			results, err := Run(ctx, cfg, store, s3Client, slog.Default())

			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Outcome).To(Equal("new_file"))
			Expect(results[0].Source).To(Equal("test-active"))
			Expect(results[0].Size).To(BeNumerically(">", 0))
			Expect(results[0].Rows).To(BeNumerically(">", 0))
		})
	})

	Context("one source fails, another succeeds", func() {
		It("continues processing remaining sources and includes error outcome", func() {
			goodServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, "NoticeId,Title\nID-001,Good\n")
			}))
			defer goodServer.Close()

			badServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			}))
			defer badServer.Close()

			Sources = []Source{
				{Key: "failing-source", URL: badServer.URL, S3Prefix: "raw/bad"},
				{Key: "good-source", URL: goodServer.URL, S3Prefix: "raw/good"},
			}

			results, err := Run(ctx, cfg, store, s3Client, slog.Default())

			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(2))

			Expect(results[0].Source).To(Equal("failing-source"))
			Expect(results[0].Outcome).To(Equal("error"))

			Expect(results[1].Source).To(Equal("good-source"))
			Expect(results[1].Outcome).To(Equal("new_file"))
		})
	})

	Context("S3 upload is disabled", func() {
		It("skips upload and sets empty S3 key", func() {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, "NoticeId,Title\nID-001,No Upload\n")
			}))
			defer server.Close()

			cfg.S3ArchiveEnabled = false
			Sources = []Source{{Key: "test-active", URL: server.URL, S3Prefix: "raw/csv"}}

			putCalled := false
			s3Client.putFn = func(_ context.Context, _ *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
				putCalled = true
				return &s3.PutObjectOutput{}, nil
			}

			results, err := Run(ctx, cfg, store, s3Client, slog.Default())

			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Outcome).To(Equal("new_file"))
			Expect(results[0].S3Key).To(BeEmpty())
			Expect(putCalled).To(BeFalse())
		})
	})

	Context("S3 upload is enabled", func() {
		It("uploads to S3 and records the S3 key", func() {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, "NoticeId,Title\nID-001,With Upload\n")
			}))
			defer server.Close()

			cfg.S3ArchiveEnabled = true
			Sources = []Source{{Key: "test-active", URL: server.URL, S3Prefix: "raw/csv"}}

			var uploadedKey string
			s3Client.putFn = func(_ context.Context, params *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
				uploadedKey = *params.Key
				return &s3.PutObjectOutput{}, nil
			}

			results, err := Run(ctx, cfg, store, s3Client, slog.Default())

			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Outcome).To(Equal("new_file"))
			Expect(results[0].S3Key).NotTo(BeEmpty())
			Expect(uploadedKey).To(ContainSubstring("raw/csv/"))
			Expect(uploadedKey).To(HaveSuffix(".csv.gz"))
		})
	})

	Context("FormatSummary", func() {
		It("builds a human-readable summary", func() {
			results := []SourceResult{
				{Source: "active", Outcome: "new_file", Size: 10 * 1024 * 1024, Rows: 50000},
				{Source: "archived", Outcome: "not_modified"},
			}
			summary := FormatSummary(results)

			Expect(summary).To(ContainSubstring("active: new_file"))
			Expect(summary).To(ContainSubstring("50000 rows"))
			Expect(summary).To(ContainSubstring("archived: not_modified"))
		})
	})
})
