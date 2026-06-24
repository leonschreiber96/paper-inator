package serviceWorker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// LLMScorer scores publications by sending title + abstract to an
// OpenAI-compatible chat completions endpoint. The response is expected to be
// JSON with a numeric "score" (0–10) and a "reasoning" string; a regex
// fallback extracts the first integer in [0,10] if JSON parsing fails.
type LLMScorer struct {
	BaseURL string // e.g. "https://api.openai.com/v1"
	APIKey  string
	Model   string
	client  *http.Client
}

// NewLLMScorer creates a scorer with a 30-second per-request timeout.
func NewLLMScorer(baseURL, apiKey, model string) *LLMScorer {
	return &LLMScorer{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
		Model:   model,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *LLMScorer) Score(ctx context.Context, title, abstract, profile string) (float64, string, error) {
	systemPrompt := fmt.Sprintf(`You are a research relevance classifier. The user's research interests are:

%s

Score the following academic paper for relevance to these research interests on a scale of 0 to 10:
- 0–2: Not relevant
- 3–5: Tangentially related
- 6–8: Relevant
- 9–10: Highly relevant, directly addresses core interests

Respond with JSON only (no markdown): {"score": <integer 0-10>, "reasoning": "<one sentence>"}`, profile)

	userMessage := fmt.Sprintf("Title: %s\n\nAbstract: %s", title, abstract)

	body, err := json.Marshal(map[string]any{
		"model": s.Model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userMessage},
		},
		"response_format": map[string]string{"type": "json_object"},
		"max_tokens":      150,
		"temperature":     0.1,
	})
	if err != nil {
		return 0, "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return 0, "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.APIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("llm request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 32*1024))
	if err != nil {
		return 0, "", fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return 0, "", fmt.Errorf("llm returned %d: %s", resp.StatusCode, string(raw))
	}

	content, err := extractContent(raw)
	if err != nil {
		return 0, "", err
	}

	return parseScoreResponse(content)
}

// extractContent pulls the assistant message text out of the OpenAI response
// envelope: choices[0].message.content
func extractContent(raw []byte) (string, error) {
	var envelope struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return "", fmt.Errorf("parse llm envelope: %w", err)
	}
	if envelope.Error != nil {
		return "", fmt.Errorf("llm api error: %s", envelope.Error.Message)
	}
	if len(envelope.Choices) == 0 {
		return "", fmt.Errorf("llm returned no choices")
	}
	return envelope.Choices[0].Message.Content, nil
}

// parseScoreResponse tries JSON first, then falls back to regex extraction.
// Returns an error only when no integer in [0,10] can be found at all.
func parseScoreResponse(content string) (float64, string, error) {
	// Primary: JSON decode
	var result struct {
		Score     json.Number `json:"score"`
		Reasoning string      `json:"reasoning"`
	}
	// Strip markdown code fences if present
	cleaned := strings.TrimSpace(content)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	if err := json.Unmarshal([]byte(cleaned), &result); err == nil {
		if n, err := result.Score.Int64(); err == nil && n >= 0 && n <= 10 {
			return float64(n) / 10.0, result.Reasoning, nil
		}
	}

	// Fallback: find first integer 0–10 in the response
	if n, ok := extractFirstInt(content); ok {
		return float64(n) / 10.0, "", nil
	}

	return 0, "", fmt.Errorf("could not extract score from llm response: %q", content)
}

var intPattern = regexp.MustCompile(`\b(10|[0-9])\b`)

func extractFirstInt(s string) (int, bool) {
	m := intPattern.FindString(s)
	if m == "" {
		return 0, false
	}
	n, err := strconv.Atoi(m)
	if err != nil {
		return 0, false
	}
	return n, true
}
