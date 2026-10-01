package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/handriss/govtrove/pipeline/internal/bulkcsv"
	"github.com/handriss/govtrove/pipeline/internal/database"
	"github.com/handriss/govtrove/pipeline/internal/testutil"
)

type memObjects struct {
	objects map[string][]byte
}

func (m *memObjects) GetObject(_ context.Context, p *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	b, ok := m.objects[*p.Key]
	if !ok {
		return nil, errors.New("AccessDenied")
	}
	return &s3.GetObjectOutput{Body: io.NopCloser(bytes.NewReader(b))}, nil
}

func (m *memObjects) PutObject(_ context.Context, p *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	b, _ := io.ReadAll(p.Body)
	m.objects[*p.Key] = b
	return &s3.PutObjectOutput{}, nil
}

func upstream(t *testing.T, etag string) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", etag)
		w.Write([]byte("Notice ID\n1\n"))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func withSource(t *testing.T, url string) {
	orig := bulkcsv.Sources
	t.Cleanup(func() { bulkcsv.Sources = orig })
	bulkcsv.Sources = []bulkcsv.Source{{Key: "active", Type: bulkcsv.SourceTypeActive, URL: url, S3Prefix: "raw/csv"}}
}

func lazyHandler(t *testing.T, objects *memObjects, store *testutil.MockStore) (*Handler, *int) {
	h, _ := stubHandler(t)
	h.Store = nil
	h.Validators = objects
	connects := 0
	h.Connect = func(context.Context) (database.Store, error) {
		connects++
		return store, nil
	}
	return h, &connects
}

func TestHandle_UnchangedUpstreamNeverConnectsToDatabase(t *testing.T) {
	withSource(t, upstream(t, `"v1"`).URL)
	marker, _ := json.Marshal(bulkcsv.Validators{ETag: `"v1"`})
	objects := &memObjects{objects: map[string][]byte{"raw/csv-validators/active.json": marker}}
	h, connects := lazyHandler(t, objects, &testutil.MockStore{})

	out, err := h.Handle(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Files) != 0 || *connects != 0 {
		t.Fatalf("files=%d connects=%d, want 0 and 0", len(out.Files), *connects)
	}
}

func TestHandle_MissingMarkerFallsBackToDatabaseAndWritesMarker(t *testing.T) {
	withSource(t, upstream(t, `"v1"`).URL)
	objects := &memObjects{objects: map[string][]byte{}}
	store := &testutil.MockStore{
		GetLatestBulkCSVHeadersFn: func(context.Context, string) (string, string, error) { return `"v1"`, "", nil },
	}
	h, connects := lazyHandler(t, objects, store)

	if _, err := h.Handle(context.Background(), json.RawMessage(`{}`)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *connects != 1 {
		t.Fatalf("connects = %d, want 1", *connects)
	}
	var v bulkcsv.Validators
	if err := json.Unmarshal(objects.objects["raw/csv-validators/active.json"], &v); err != nil || v.ETag != `"v1"` {
		t.Fatalf("marker = %+v, %v", v, err)
	}
}

func TestHandle_ChangedUpstreamTakesDatabasePath(t *testing.T) {
	withSource(t, upstream(t, `"v2"`).URL)
	marker, _ := json.Marshal(bulkcsv.Validators{ETag: `"v1"`})
	objects := &memObjects{objects: map[string][]byte{"raw/csv-validators/active.json": marker}}
	h, connects := lazyHandler(t, objects, &testutil.MockStore{})

	out, err := h.Handle(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *connects != 1 || len(out.Files) != 1 {
		t.Fatalf("connects=%d files=%d, want 1 and 1", *connects, len(out.Files))
	}
}

func TestHandle_DatabaseConnectFailureIsAnError(t *testing.T) {
	withSource(t, upstream(t, `"v2"`).URL)
	h, _ := stubHandler(t)
	h.Store = nil
	h.Connect = func(context.Context) (database.Store, error) { return nil, errors.New("neon down") }

	if _, err := h.Handle(context.Background(), json.RawMessage(`{}`)); err == nil {
		t.Fatal("want error when the database is unreachable")
	}
}
