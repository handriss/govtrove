package samgov

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"
)

const testKey = "SAM-32cc7825-b77c-4272-8b34-bc656d29e7a2"

func reqURL() string {
	return baseURL + "?api_key=" + testKey + "&limit=1000&offset=1000"
}

func TestRedactAPIKeyRemovesCredential(t *testing.T) {
	got := redactAPIKey("Get \"" + reqURL() + "\": dial tcp: timeout")
	if strings.Contains(got, testKey) {
		t.Fatalf("credential survived redaction: %s", got)
	}
	if !strings.Contains(got, "api_key=REDACTED") {
		t.Fatalf("expected redaction marker, got: %s", got)
	}
	if !strings.Contains(got, "limit=1000") {
		t.Fatalf("redaction ate unrelated params: %s", got)
	}
}

// The real leak path: net/http returns *url.Error carrying the full request URL,
// and wrapping it verbatim is what put the key into Sentry and the database.
func TestRedactErrScrubsURLError(t *testing.T) {
	ue := &url.Error{Op: "Get", URL: reqURL(), Err: errors.New("dial tcp: i/o timeout")}
	wrapped := fmt.Errorf("execute request: %w", redactErr(ue))
	outer := fmt.Errorf("fetch window 01/01/2025-12/31/2025: page 2 (offset 1000): %w", wrapped)

	if strings.Contains(outer.Error(), testKey) {
		t.Fatalf("credential survived through the wrap chain: %s", outer)
	}

	// The structured field must be scrubbed too, or errors.As hands the raw URL back.
	var recovered *url.Error
	if !errors.As(outer, &recovered) {
		t.Fatal("errors.As no longer reaches the *url.Error")
	}
	if strings.Contains(recovered.URL, testKey) {
		t.Fatalf("credential readable via errors.As: %s", recovered.URL)
	}
}

func TestRedactErrPreservesErrorChain(t *testing.T) {
	sentinel := errors.New("sentinel")
	if !errors.Is(redactErr(fmt.Errorf("wrapped: %w", sentinel)), sentinel) {
		t.Fatal("errors.Is broken by redaction")
	}
	if redactErr(nil) != nil {
		t.Fatal("redactErr(nil) must stay nil")
	}
}
