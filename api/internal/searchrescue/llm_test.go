package searchrescue

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/handriss/govtrove/api/internal/models"
)

// fakeOpenRouter scripts an OpenAI-style conversation: first a count_search
// tool call, then a final answer containing one probed and one fabricated
// suggestion. Enforcement must keep only the probed one.
func fakeOpenRouter(t *testing.T) *httptest.Server {
	round := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req orRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("bad request body: %v", err)
		}
		round++
		w.Header().Set("Content-Type", "application/json")

		switch round {
		case 1:
			if req.Messages[0].Role != "system" || !strings.Contains(req.Messages[0].Content, "search rescuer") {
				t.Errorf("system prompt missing, got role %q", req.Messages[0].Role)
			}
			w.Write([]byte(`{"choices":[{"message":{
				"role":"assistant",
				"tool_calls":[{"id":"call_1","type":"function","function":{"name":"count_search","arguments":"{\"params\":{\"q\":\"custodial services\"}}"}}]
			}}]}`))
		default:
			var toolResult string
			for _, m := range req.Messages {
				if m.Role == "tool" {
					toolResult = m.Content
				}
			}
			if !strings.Contains(toolResult, `"total": 33`) {
				t.Errorf("tool result not fed back, got %q", toolResult)
			}
			final := `{"cause":"no-market","explanation":"Government notices call this custodial services.","suggestions":[` +
				`{"label":"Search custodial services","params":{"q":"custodial services"}},` +
				`{"label":"Fabricated","params":{"q":"unicorn wrangling"}}]}`
			resp := map[string]any{
				"choices": []map[string]any{{"message": map[string]any{"role": "assistant", "content": final}}},
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
}

func TestLLMRescueEnforcement(t *testing.T) {
	srv := fakeOpenRouter(t)
	defer srv.Close()

	prober := &fakeProber{countFn: func(p models.SearchParams) int {
		if p.Query == "custodial services" {
			return 33
		}
		return 0
	}}
	llm := NewLLMClient("test-key", "test-model")
	llm.baseURL = srv.URL
	s := newTestService(prober, llm)

	res := s.Rescue(context.Background(), models.SearchParams{Query: "cleaning lady"}, 0, "")
	if res == nil || res.Stage != "llm" {
		t.Fatalf("result = %+v", res)
	}
	if len(res.Suggestions) != 1 {
		t.Fatalf("expected exactly the probed suggestion to survive, got %+v", res.Suggestions)
	}
	sug := res.Suggestions[0]
	if sug.Params["q"] != "custodial services" || sug.VerifiedTotal != 33 || sug.Rule != "llm" {
		t.Errorf("suggestion = %+v", sug)
	}
}

func TestLLMDisabledFallsThroughToNone(t *testing.T) {
	prober := &fakeProber{countFn: func(models.SearchParams) int { return 0 }}
	s := newTestService(prober, nil)
	res := s.Rescue(context.Background(), models.SearchParams{Query: "cleaning lady"}, 0, "")
	if res == nil || res.Stage != "none" || len(res.Suggestions) != 0 {
		t.Fatalf("result = %+v", res)
	}
}

func TestEnforceRejectsUnparseable(t *testing.T) {
	s := newTestService(&fakeProber{countFn: func(models.SearchParams) int { return 0 }}, nil)
	if res := s.enforce("this is not json", newProbeLog(), 0); res != nil {
		t.Fatalf("expected nil for unparseable output, got %+v", res)
	}
}

func TestEnforceUsesProbedCountNotClaimed(t *testing.T) {
	s := newTestService(&fakeProber{countFn: func(models.SearchParams) int { return 0 }}, nil)
	pl := newProbeLog()
	probed := s.normalizeForProbe(models.SearchParams{Query: "custodial"})
	pl.seen[canonical(probed)] = 12

	out := `{"cause":"typo","explanation":"x","suggestions":[{"label":"a","params":{"q":"custodial"}}]}`
	res := s.enforce(out, pl, 0)
	if res == nil || len(res.Suggestions) != 1 || res.Suggestions[0].VerifiedTotal != 12 {
		t.Fatalf("result = %+v", res)
	}
}
