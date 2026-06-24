package serviceWorker

import "testing"

const rssSample = `<?xml version="1.0"?>
<rss version="2.0" xmlns:dc="http://purl.org/dc/elements/1.1/">
  <channel>
    <title>Sample Journal</title>
    <item>
      <title>A Great Paper</title>
      <link>https://example.org/a</link>
      <description>An abstract.</description>
      <dc:creator>Ada Lovelace</dc:creator>
      <pubDate>Mon, 02 Jan 2006 15:04:05 -0700</pubDate>
    </item>
    <item>
      <title>Another Paper</title>
      <link>https://example.org/b</link>
      <description>Second abstract.</description>
      <author>grace@example.org (Grace Hopper)</author>
    </item>
  </channel>
</rss>`

const atomSample = `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Sample Atom Feed</title>
  <entry>
    <title>Atom Paper</title>
    <link rel="alternate" href="https://example.org/atom-paper"/>
    <summary>Atom abstract.</summary>
    <author><name>Alan Turing</name></author>
    <author><name>John von Neumann</name></author>
    <published>2020-05-01T12:00:00Z</published>
  </entry>
</feed>`

func TestParseRSS(t *testing.T) {
	items, err := Parse([]byte(rssSample))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	first := items[0]
	if first.Title != "A Great Paper" {
		t.Errorf("title = %q", first.Title)
	}
	if first.Authors != "Ada Lovelace" {
		t.Errorf("authors = %q (expected dc:creator)", first.Authors)
	}
	if first.Link != "https://example.org/a" {
		t.Errorf("link = %q", first.Link)
	}
	if first.Summary != "An abstract." {
		t.Errorf("summary = %q", first.Summary)
	}
	if first.Published == nil {
		t.Error("expected published date to be parsed")
	}
}

func TestParseAtom(t *testing.T) {
	items, err := Parse([]byte(atomSample))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(items))
	}
	e := items[0]
	if e.Title != "Atom Paper" {
		t.Errorf("title = %q", e.Title)
	}
	if e.Authors != "Alan Turing, John von Neumann" {
		t.Errorf("authors = %q (expected both names joined)", e.Authors)
	}
	if e.Link != "https://example.org/atom-paper" {
		t.Errorf("link = %q", e.Link)
	}
	if e.Published == nil {
		t.Error("expected published date to be parsed")
	}
}
