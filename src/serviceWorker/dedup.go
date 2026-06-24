package serviceWorker

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// DedupKey produces a deterministic, explainable identifier for a publication
// based on its title and authors. Two items that normalize to the same title and
// author set produce the same key and are therefore treated as duplicates.
//
// Normalization (lowercase, trim, collapse internal whitespace) makes the key
// robust to trivial formatting differences between feeds while remaining easy to
// reason about: the key is the SHA-256 of "normalized_title|normalized_authors".
func DedupKey(title, authors string) string {
	h := sha256.Sum256([]byte(normalize(title) + "|" + normalize(authors)))
	return hex.EncodeToString(h[:])
}

// normalize lowercases, trims, and collapses runs of whitespace into a single
// space so that cosmetic differences do not defeat deduplication.
func normalize(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}
