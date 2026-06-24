package serviceWorker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func mockLLMServer(t *testing.T, responseContent string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": responseContent}},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func TestLLMScorerValidJSON(t *testing.T) {
	srv := mockLLMServer(t, `{"score": 8, "reasoning": "Directly addresses mechanistic interpretability."}`)
	defer srv.Close()

	sc := NewLLMScorer(srv.URL, "test-key", "test-model")
	score, notes, err := sc.Score(context.Background(), "Attention Heads", "We study attention patterns.", "interpretability")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if score != 0.8 {
		t.Errorf("expected 0.8, got %.2f", score)
	}
	if notes != "Directly addresses mechanistic interpretability." {
		t.Errorf("unexpected notes: %q", notes)
	}
}

func TestLLMScorerMarkdownFencedJSON(t *testing.T) {
	srv := mockLLMServer(t, "```json\n{\"score\": 5, \"reasoning\": \"Tangentially related.\"}\n```")
	defer srv.Close()

	sc := NewLLMScorer(srv.URL, "test-key", "test-model")
	score, _, err := sc.Score(context.Background(), "T", "A", "profile")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if score != 0.5 {
		t.Errorf("expected 0.5, got %.2f", score)
	}
}

func TestLLMScorerFallbackRegex(t *testing.T) {
	// LLM returns free text instead of JSON
	srv := mockLLMServer(t, "I would give this paper a score of 7 out of 10.")
	defer srv.Close()

	sc := NewLLMScorer(srv.URL, "test-key", "test-model")
	score, _, err := sc.Score(context.Background(), "T", "A", "profile")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if score != 0.7 {
		t.Errorf("expected 0.7 from regex fallback, got %.2f", score)
	}
}

func TestLLMScorerUnparseableResponse(t *testing.T) {
	srv := mockLLMServer(t, "This paper is very interesting and relevant.")
	defer srv.Close()

	sc := NewLLMScorer(srv.URL, "test-key", "test-model")
	_, _, err := sc.Score(context.Background(), "T", "A", "profile")
	if err == nil {
		t.Error("expected error when no integer can be extracted from response")
	}
}

func TestLLMScorerHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"invalid api key"}}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	sc := NewLLMScorer(srv.URL, "bad-key", "test-model")
	_, _, err := sc.Score(context.Background(), "T", "A", "profile")
	if err == nil {
		t.Error("expected error on HTTP 401")
	}
}

func TestParseScoreResponse(t *testing.T) {
	cases := []struct {
		input     string
		wantScore float64
		wantOK    bool
	}{
		{`{"score": 9, "reasoning": "Highly relevant."}`, 0.9, true},
		{`{"score": 0, "reasoning": "Not relevant."}`, 0.0, true},
		{`{"score": 10, "reasoning": "Perfect match."}`, 1.0, true},
		{"Score: 6/10", 0.6, true},
		{"I rate it 3.", 0.3, true},
		{"no numbers here", 0, false},
	}
	for _, c := range cases {
		score, _, err := parseScoreResponse(c.input)
		if c.wantOK && err != nil {
			t.Errorf("input %q: unexpected error: %v", c.input, err)
			continue
		}
		if !c.wantOK && err == nil {
			t.Errorf("input %q: expected error, got score %.2f", c.input, score)
			continue
		}
		if c.wantOK && score != c.wantScore {
			t.Errorf("input %q: expected %.1f, got %.2f", c.input, c.wantScore, score)
		}
	}
}
