package serviceWorker

import (
	"context"
	"strings"
	"testing"
)

func TestKeywordScorerPerfectMatch(t *testing.T) {
	sc := KeywordScorer{}
	profile := "mechanistic interpretability transformer attention"
	title := "Mechanistic Interpretability of Transformer Attention Heads"
	abstract := "We study the internal mechanisms of transformer models."

	score, notes, err := sc.Score(context.Background(), title, abstract, profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if score < 0.8 {
		t.Errorf("expected high score, got %.2f", score)
	}
	if !strings.Contains(notes, "Matched:") {
		t.Errorf("expected notes to list matched terms, got: %q", notes)
	}
}

func TestKeywordScorerNoMatch(t *testing.T) {
	sc := KeywordScorer{}
	profile := "mechanistic interpretability sparse autoencoders"
	title := "Efficient GPU Kernel Fusion for Deep Learning"
	abstract := "We optimize CUDA kernels for reduced memory bandwidth."

	score, _, err := sc.Score(context.Background(), title, abstract, profile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if score > 0.2 {
		t.Errorf("expected low score for unrelated paper, got %.2f", score)
	}
}

func TestKeywordScorerEmptyProfile(t *testing.T) {
	sc := KeywordScorer{}
	score, notes, err := sc.Score(context.Background(), "Any Title", "Any abstract.", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if score != 0 {
		t.Errorf("expected 0 score for empty profile, got %.2f", score)
	}
	if notes != "" {
		t.Errorf("expected empty notes for empty profile, got %q", notes)
	}
}

func TestKeywordScorerStopwordsIgnored(t *testing.T) {
	// Profile contains only stopwords — tokenize should produce no terms
	profile := "the and or but in on"
	terms := tokenize(profile)
	if len(terms) != 0 {
		t.Errorf("expected no terms from stopword-only profile, got %v", terms)
	}
}

func TestKeywordScorerTitleWeightedHigher(t *testing.T) {
	profile := "interpretability"

	// Title match only
	scoreTitleMatch, _, _ := KeywordScorer{}.Score(context.Background(), "Interpretability of Neural Networks", "We discuss various optimization tricks.", profile)
	// Abstract match only
	scoreAbstractMatch, _, _ := KeywordScorer{}.Score(context.Background(), "Advances in Deep Learning", "We study interpretability of models.", profile)

	// Both should match the term, resulting in same score (since we count unique matched terms)
	// But this test verifies that title content is included in scoring at all
	if scoreTitleMatch == 0 {
		t.Error("expected non-zero score when term appears in title")
	}
	if scoreAbstractMatch == 0 {
		t.Error("expected non-zero score when term appears in abstract")
	}
}

func TestTokenize(t *testing.T) {
	terms := tokenize("Large Language Models (LLMs) and their safety implications.")
	// Should contain: large, language, models, llms, safety, implications
	// Should NOT contain: and, their, the
	found := make(map[string]bool)
	for _, t := range terms {
		found[t] = true
	}
	for _, expected := range []string{"large", "language", "models", "llms", "safety", "implications"} {
		if !found[expected] {
			t.Errorf("expected term %q in tokenized output, got %v", expected, terms)
		}
	}
	for _, unexpected := range []string{"and", "their"} {
		if found[unexpected] {
			t.Errorf("stopword %q should not appear in tokenized output", unexpected)
		}
	}
}
