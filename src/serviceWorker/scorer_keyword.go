package serviceWorker

import (
	"context"
	"sort"
	"strings"
)

// KeywordScorer scores publications by term overlap between the publication's
// title+abstract and the user's interest profile. It is pure Go with no
// external dependencies and requires no network access.
//
// Scoring: terms extracted from the profile are matched against the publication
// text. Title matches count double (title is more signal-dense than abstract).
// The final score = unique matched terms / total profile terms, clamped to 1.0.
type KeywordScorer struct{}

func (KeywordScorer) Score(_ context.Context, title, abstract, profile string) (float64, string, error) {
	profileTerms := tokenize(profile)
	if len(profileTerms) == 0 {
		return 0, "", nil
	}

	// Build a weighted term frequency map over title (×2) + abstract (×1).
	pubText := strings.ToLower(title) + " " + strings.ToLower(title) + " " + strings.ToLower(abstract)
	pubWords := strings.Fields(pubText)
	pubWordSet := make(map[string]struct{}, len(pubWords))
	for _, w := range pubWords {
		pubWordSet[stripPunct(w)] = struct{}{}
	}

	var matched []string
	seen := make(map[string]bool)
	for _, term := range profileTerms {
		if seen[term] {
			continue
		}
		if _, ok := pubWordSet[term]; ok {
			matched = append(matched, term)
			seen[term] = true
		}
	}

	score := float64(len(matched)) / float64(len(profileTerms))
	if score > 1.0 {
		score = 1.0
	}

	sort.Strings(matched)
	notes := ""
	if len(matched) > 0 {
		notes = "Matched: " + strings.Join(matched, ", ")
	}
	return score, notes, nil
}

// tokenize lowercases, strips punctuation, removes stopwords, and deduplicates
// the terms in s. The resulting slice represents the unique meaningful terms
// extracted from the user's interest profile.
func tokenize(s string) []string {
	words := strings.Fields(strings.ToLower(s))
	seen := make(map[string]bool)
	var out []string
	for _, w := range words {
		w = stripPunct(w)
		if w == "" || isStopword(w) || seen[w] {
			continue
		}
		seen[w] = true
		out = append(out, w)
	}
	return out
}

// stripPunct removes leading/trailing ASCII punctuation from a word.
func stripPunct(s string) string {
	return strings.TrimFunc(s, func(r rune) bool {
		return r == '.' || r == ',' || r == ';' || r == ':' || r == '!' ||
			r == '?' || r == '"' || r == '\'' || r == '(' || r == ')' ||
			r == '[' || r == ']' || r == '{' || r == '}' || r == '-'
	})
}

// isStopword reports whether w is a common English word that carries no
// research-relevance signal.
func isStopword(w string) bool {
	_, ok := stopwords[w]
	return ok
}

// stopwords is a minimal set of common English words with no domain signal.
var stopwords = map[string]struct{}{
	"a": {}, "an": {}, "the": {}, "and": {}, "or": {}, "but": {}, "not": {},
	"in": {}, "on": {}, "at": {}, "to": {}, "for": {}, "of": {}, "with": {},
	"by": {}, "from": {}, "as": {}, "is": {}, "are": {}, "was": {}, "were": {},
	"be": {}, "been": {}, "being": {}, "have": {}, "has": {}, "had": {},
	"do": {}, "does": {}, "did": {}, "will": {}, "would": {}, "could": {},
	"should": {}, "may": {}, "might": {}, "shall": {}, "can": {},
	"i": {}, "we": {}, "you": {}, "he": {}, "she": {}, "it": {}, "they": {},
	"my": {}, "our": {}, "your": {}, "his": {}, "her": {}, "its": {}, "their": {},
	"this": {}, "that": {}, "these": {}, "those": {}, "such": {}, "also": {},
	"about": {}, "than": {}, "more": {}, "very": {}, "so": {}, "if": {},
	"then": {}, "when": {}, "where": {}, "which": {}, "who": {},
	"how": {}, "what": {}, "paper": {}, "study": {}, "research": {}, "work": {},
	"propose": {}, "show": {}, "present": {}, "use": {}, "used": {}, "using": {},
	"based": {}, "approach": {}, "method": {}, "new": {}, "novel": {},
}
