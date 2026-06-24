package serviceWorker

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

// Item is a feed entry in a format-agnostic shape. The parser flattens RSS 2.0
// and Atom into this common structure; the mapping layer then turns it into a
// models.Publication. Extra is kept so per-feed field mappings can reach
// less-common fields (e.g. "dc:creator") without changing this struct.
type Item struct {
	Title       string
	Authors     string
	Summary     string
	Link        string
	Published   *time.Time
	Extra       map[string]string // namespaced/raw fields keyed by local element name
}

// Parse detects the feed flavor (RSS 2.0 or Atom) and returns its items.
func Parse(data []byte) ([]Item, error) {
	trimmed := strings.TrimSpace(string(data))
	if strings.Contains(trimmed, "<feed") && strings.Contains(trimmed, "http://www.w3.org/2005/Atom") {
		return parseAtom(data)
	}
	return parseRSS(data)
}

// --- RSS 2.0 ---

type rssRoot struct {
	Channel struct {
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
}

type rssItem struct {
	Title   string    `xml:"title"`
	Link    string    `xml:"link"`
	Desc    string    `xml:"description"`
	Creator string    `xml:"creator"` // dc:creator (namespace stripped by encoding/xml local-name match)
	Author  string    `xml:"author"`
	PubDate string    `xml:"pubDate"`
	Date    string    `xml:"date"` // dc:date
	Extra   []rawField `xml:",any"`
}

type rawField struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
}

func parseRSS(data []byte) ([]Item, error) {
	var root rssRoot
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parse RSS: %w", err)
	}
	items := make([]Item, 0, len(root.Channel.Items))
	for _, it := range root.Channel.Items {
		authors := firstNonEmpty(it.Creator, it.Author)
		published := parseTime(firstNonEmpty(it.PubDate, it.Date))
		items = append(items, Item{
			Title:     strings.TrimSpace(it.Title),
			Authors:   strings.TrimSpace(authors),
			Summary:   strings.TrimSpace(it.Desc),
			Link:      strings.TrimSpace(it.Link),
			Published: published,
			Extra:     collectExtra(it.Extra),
		})
	}
	return items, nil
}

// --- Atom ---

type atomRoot struct {
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	Title   string `xml:"title"`
	Summary string `xml:"summary"`
	Content string `xml:"content"`
	Updated string `xml:"updated"`
	Pub     string `xml:"published"`
	Links   []struct {
		Href string `xml:"href,attr"`
		Rel  string `xml:"rel,attr"`
	} `xml:"link"`
	Authors []struct {
		Name string `xml:"name"`
	} `xml:"author"`
	Extra []rawField `xml:",any"`
}

func parseAtom(data []byte) ([]Item, error) {
	var root atomRoot
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parse Atom: %w", err)
	}
	items := make([]Item, 0, len(root.Entries))
	for _, e := range root.Entries {
		var names []string
		for _, a := range e.Authors {
			if n := strings.TrimSpace(a.Name); n != "" {
				names = append(names, n)
			}
		}
		items = append(items, Item{
			Title:     strings.TrimSpace(e.Title),
			Authors:   strings.Join(names, ", "),
			Summary:   strings.TrimSpace(firstNonEmpty(e.Summary, e.Content)),
			Link:      atomLink(e.Links),
			Published: parseTime(firstNonEmpty(e.Pub, e.Updated)),
			Extra:     collectExtra(e.Extra),
		})
	}
	return items, nil
}

func atomLink(links []struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}) string {
	// Prefer the "alternate" link, falling back to the first href.
	var fallback string
	for _, l := range links {
		if l.Rel == "alternate" || l.Rel == "" {
			return strings.TrimSpace(l.Href)
		}
		if fallback == "" {
			fallback = l.Href
		}
	}
	return strings.TrimSpace(fallback)
}

// --- helpers ---

func collectExtra(fields []rawField) map[string]string {
	if len(fields) == 0 {
		return nil
	}
	m := make(map[string]string, len(fields))
	for _, f := range fields {
		v := strings.TrimSpace(f.Value)
		if v != "" {
			m[f.XMLName.Local] = v
		}
	}
	return m
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// timeLayouts covers the date formats commonly seen across RSS and Atom feeds.
var timeLayouts = []string{
	time.RFC1123Z,
	time.RFC1123,
	time.RFC3339,
	time.RFC3339Nano,
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02 15:04:05",
	"2006-01-02",
	"Mon, 02 Jan 2006 15:04:05 -0700",
	"Mon, 2 Jan 2006 15:04:05 MST",
}

func parseTime(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	for _, layout := range timeLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return &t
		}
	}
	return nil
}
