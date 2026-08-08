package searchrescue

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/handriss/govtrove/api/internal/models"
)

//go:embed prompt.md
var systemPrompt string

const (
	defaultBaseURL = "https://openrouter.ai/api/v1"
	llmTimeout     = 20 * time.Second
	llmMaxRounds   = 6
)

type LLMClient struct {
	apiKey  string
	model   string
	baseURL string
	httpc   *http.Client
}

func NewLLMClient(apiKey, model string) *LLMClient {
	return &LLMClient{
		apiKey:  apiKey,
		model:   model,
		baseURL: defaultBaseURL,
		httpc:   &http.Client{Timeout: 25 * time.Second},
	}
}

// OpenAI-compatible wire types (OpenRouter speaks this dialect).
type orMessage struct {
	Role       string       `json:"role"`
	Content    string       `json:"content,omitempty"`
	ToolCalls  []orToolCall `json:"tool_calls,omitempty"`
	ToolCallID string       `json:"tool_call_id,omitempty"`
}

type orToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type orRequest struct {
	Model       string      `json:"model"`
	Messages    []orMessage `json:"messages"`
	Tools       []orTool    `json:"tools,omitempty"`
	Temperature float64     `json:"temperature"`
	MaxTokens   int         `json:"max_tokens"`
}

type orTool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Parameters  json.RawMessage `json:"parameters"`
	} `json:"function"`
}

type orResponse struct {
	Choices []struct {
		Message orMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func countSearchTool() orTool {
	var t orTool
	t.Type = "function"
	t.Function.Name = "count_search"
	t.Function.Description = "Count the results the app would show for a candidate search. Params use the same flat string keys as the input search (q, type, set_aside, naics, naics_prefix, psc, state, agency, department, deadline_from, deadline_to, posted_from, posted_to, sol_num, pop_city). Omit a key to drop that filter."
	t.Function.Parameters = json.RawMessage(`{
		"type": "object",
		"properties": {"params": {"type": "object", "additionalProperties": {"type": "string"}}},
		"required": ["params"]
	}`)
	return t
}

type llmInput struct {
	Query              string            `json:"query"`
	Filters            map[string]string `json:"filters"`
	OriginalTotal      int               `json:"original_total"`
	SpellingSuggestion string            `json:"spelling_suggestion,omitempty"`
	ProbesAlreadyRun   []probeEntry      `json:"probes_already_run"`
	SuggestionsBudget  int               `json:"remaining_probe_budget"`
}

type llmOutput struct {
	Cause       string `json:"cause"`
	Explanation string `json:"explanation"`
	Suggestions []struct {
		Label  string            `json:"label"`
		Params map[string]string `json:"params"`
	} `json:"suggestions"`
}

func (s *Service) llmRescue(ctx context.Context, b *budget, pl *probeLog, orig models.SearchParams, origTotal int, spellFix string) *Result {
	ctx, cancel := context.WithTimeout(ctx, llmTimeout)
	defer cancel()

	filters := paramsToMap(orig)
	delete(filters, "q")
	input, err := json.Marshal(llmInput{
		Query:              orig.Query,
		Filters:            filters,
		OriginalTotal:      origTotal,
		SpellingSuggestion: spellFix,
		ProbesAlreadyRun:   pl.entries,
		SuggestionsBudget:  b.remaining,
	})
	if err != nil {
		return nil
	}

	messages := []orMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: string(input)},
	}

	for round := 0; round < llmMaxRounds; round++ {
		resp, err := s.llm.chat(ctx, messages)
		if err != nil {
			s.logger.Warn("rescue LLM call failed", "round", round, "error", err)
			return nil
		}

		if len(resp.ToolCalls) == 0 {
			return s.enforce(resp.Content, pl, origTotal)
		}

		messages = append(messages, *resp)
		for _, tc := range resp.ToolCalls {
			messages = append(messages, orMessage{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    s.runCountTool(ctx, b, pl, tc.Function.Arguments),
			})
		}
	}
	return nil
}

func (s *Service) runCountTool(ctx context.Context, b *budget, pl *probeLog, rawArgs string) string {
	var args struct {
		Params map[string]any `json:"params"`
	}
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return `{"error": "invalid arguments"}`
	}
	strParams := make(map[string]string, len(args.Params))
	for k, v := range args.Params {
		strParams[k] = fmt.Sprintf("%v", v)
	}
	if b.remaining <= 0 {
		return `{"error": "probe budget exhausted — produce your final answer from the counts you already have"}`
	}
	n, ok := s.probe(ctx, b, pl, paramsFromMap(strParams))
	if !ok {
		return `{"error": "count failed"}`
	}
	return fmt.Sprintf(`{"total": %d}`, n)
}

// enforce applies the same rules to the model's answer that the
// deterministic tiers live by: a suggestion survives only if its exact
// params were probed in this conversation with a significant count, and the
// count shown is the probed one — never the model's claim.
func (s *Service) enforce(content string, pl *probeLog, origTotal int) *Result {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")

	var out llmOutput
	if err := json.Unmarshal([]byte(strings.TrimSpace(content)), &out); err != nil {
		s.logger.Warn("rescue LLM output unparseable", "error", err)
		return nil
	}

	result := &Result{
		Stage:       "llm",
		Cause:       out.Cause,
		Explanation: out.Explanation,
		Suggestions: []Suggestion{},
	}
	for _, sug := range out.Suggestions {
		if len(result.Suggestions) >= maxSuggestions {
			break
		}
		p := paramsFromMap(sug.Params)
		total, probed := pl.seen[canonical(s.normalizeForProbe(p))]
		if !probed || !significant(origTotal, total) {
			s.logger.Warn("rescue LLM suggestion rejected", "label", sug.Label, "probed", probed, "total", total)
			continue
		}
		result.Suggestions = append(result.Suggestions, Suggestion{
			Label:         sug.Label,
			Params:        sug.Params,
			VerifiedTotal: total,
			Rule:          "llm",
		})
	}
	if result.Cause == "" {
		result.Cause = "no-market"
	}
	return result
}

func (c *LLMClient) chat(ctx context.Context, messages []orMessage) (*orMessage, error) {
	body, err := json.Marshal(orRequest{
		Model:       c.model,
		Messages:    messages,
		Tools:       []orTool{countSearchTool()},
		Temperature: 0.2,
		MaxTokens:   1500,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("HTTP-Referer", "https://govtrove.com")
	req.Header.Set("X-Title", "GovTrove Search Rescue")

	resp, err := c.httpc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var parsed orResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decoding response (status %d): %w", resp.StatusCode, err)
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("openrouter: %s", parsed.Error.Message)
	}
	if resp.StatusCode != http.StatusOK || len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("openrouter: status %d, %d choices", resp.StatusCode, len(parsed.Choices))
	}
	return &parsed.Choices[0].Message, nil
}
