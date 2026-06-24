package serviceWorker

import "testing"

func TestDedupKeyDeterministic(t *testing.T) {
	a := DedupKey("Attention Is All You Need", "Vaswani et al.")
	b := DedupKey("Attention Is All You Need", "Vaswani et al.")
	if a != b {
		t.Fatalf("same input produced different keys:\n%s\n%s", a, b)
	}
}

func TestDedupKeyNormalizes(t *testing.T) {
	// Cosmetic differences (case, extra/leading whitespace) must not defeat dedup.
	cases := [][2]string{
		{"Deep Learning", "deep   learning"},
		{"  A Study  ", "a study"},
		{"Multi\tWord\nTitle", "multi word title"},
	}
	for _, c := range cases {
		if DedupKey(c[0], "X") != DedupKey(c[1], "X") {
			t.Errorf("expected %q and %q to dedup to the same key", c[0], c[1])
		}
	}
}

func TestDedupKeyDistinguishes(t *testing.T) {
	// Different titles or authors must produce different keys.
	if DedupKey("Title A", "Author") == DedupKey("Title B", "Author") {
		t.Error("different titles produced the same key")
	}
	if DedupKey("Title", "Author A") == DedupKey("Title", "Author B") {
		t.Error("different authors produced the same key")
	}
}
