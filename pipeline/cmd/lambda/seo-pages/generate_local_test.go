package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/handriss/govtrove/pipeline/internal/database"
)

// localS3 captures uploads to disk instead of S3 so a full generation run can be inspected
// before it reaches production. ListObjectsV2 returns nothing, which makes the prune step a
// no-op — this harness must never be able to delete live objects.
type localS3 struct {
	mu   sync.Mutex
	dir  string
	keys []string
}

func (l *localS3) PutObject(_ context.Context, in *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	l.mu.Lock()
	l.keys = append(l.keys, *in.Key)
	l.mu.Unlock()

	path := filepath.Join(l.dir, *in.Key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if _, err := f.ReadFrom(in.Body); err != nil {
		return nil, err
	}
	return &s3.PutObjectOutput{}, nil
}

func (l *localS3) ListObjectsV2(context.Context, *s3.ListObjectsV2Input, ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	return &s3.ListObjectsV2Output{}, nil
}

func (l *localS3) DeleteObjects(context.Context, *s3.DeleteObjectsInput, ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error) {
	panic("local harness must not delete objects")
}

type noopCF struct{}

func (noopCF) CreateInvalidation(context.Context, *cloudfront.CreateInvalidationInput, ...func(*cloudfront.Options)) (*cloudfront.CreateInvalidationOutput, error) {
	return &cloudfront.CreateInvalidationOutput{}, nil
}

// TestGenerateLocal renders the whole site against a real database and writes it to disk.
// Opt-in: SEO_LOCAL_DB=<dsn> SEO_LOCAL_OUT=<dir> go test -run TestGenerateLocal ./cmd/lambda/seo-pages/
func TestGenerateLocal(t *testing.T) {
	dsn := os.Getenv("SEO_LOCAL_DB")
	if dsn == "" {
		t.Skip("SEO_LOCAL_DB not set")
	}
	outDir := os.Getenv("SEO_LOCAL_OUT")
	if outDir == "" {
		outDir = t.TempDir()
	}

	ctx := context.Background()
	db, err := database.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	s3stub := &localS3{dir: outDir}
	h := &Handler{
		Store:    db,
		S3:       s3stub,
		CF:       noopCF{},
		S3Bucket: "local",
		Logger:   slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})),
	}

	out, err := h.Handle(ctx, []byte(`{}`))
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	if out.Failures > 0 {
		t.Errorf("run reported %d failures", out.Failures)
	}

	var naics, psc, pscGroup, sector int
	for _, k := range s3stub.keys {
		switch {
		case strings.HasPrefix(k, "contracts/naics/sector/"):
			sector++
		case strings.HasPrefix(k, "contracts/psc/group/"):
			pscGroup++
		case strings.HasPrefix(k, "contracts/psc/"):
			psc++
		case strings.HasPrefix(k, "contracts/naics/"):
			naics++
		}
	}
	t.Logf("out=%+v", out)
	t.Logf("written: naics=%d psc=%d psc_groups=%d sectors=%d total=%d", naics, psc, pscGroup, sector, len(s3stub.keys))
	t.Logf("output dir: %s", outDir)

	if psc == 0 || pscGroup == 0 || sector == 0 {
		t.Errorf("expected PSC, PSC group and sector pages to be generated")
	}
}
