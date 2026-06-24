package serviceWorker

import (
	"encoding/json"
	"time"

	"paperinator/src/shared/models"
)

// MapItem converts a parsed feed Item into a normalized Publication for the given
// feed, applying any per-feed field mappings on top of sensible defaults.
//
// By default the parser's normalized fields are used directly. A FieldMapping
// overrides a target field by naming a source: either one of the standard
// normalized names ("title", "authors", "summary"/"abstract", "link",
// "published_at") or a key present in the item's Extra map (e.g. "dc:creator"
// arrives as "creator"). This keeps mapping configurable per feed for
// heterogeneous RSS formats without special-casing each one.
func MapItem(feedID int64, it Item, mappings []models.FieldMapping) models.Publication {
	pub := models.Publication{
		FeedID:      feedID,
		Title:       it.Title,
		Authors:     it.Authors,
		Abstract:    it.Summary,
		Link:        it.Link,
		PublishedAt: it.Published,
		FetchedAt:   time.Now().UTC(),
	}

	for _, m := range mappings {
		applyMapping(&pub, it, m)
	}

	pub.DedupKey = DedupKey(pub.Title, pub.Authors)
	pub.Raw = rawJSON(it)
	return pub
}

func applyMapping(pub *models.Publication, it Item, m models.FieldMapping) {
	value := resolveSource(it, m.SourceField)
	if value == "" && m.TargetField != "published_at" {
		return // don't clobber a good default with an empty override
	}
	switch m.TargetField {
	case "title":
		pub.Title = value
	case "authors":
		pub.Authors = value
	case "abstract":
		pub.Abstract = value
	case "link":
		pub.Link = value
	case "published_at":
		if t := parseTime(value); t != nil {
			pub.PublishedAt = t
		}
	}
}

// resolveSource returns the value of a named source field, checking the standard
// normalized fields first, then the item's Extra map.
func resolveSource(it Item, source string) string {
	switch source {
	case "title":
		return it.Title
	case "authors":
		return it.Authors
	case "summary", "abstract", "description":
		return it.Summary
	case "link":
		return it.Link
	}
	if it.Extra != nil {
		return it.Extra[source]
	}
	return ""
}

func rawJSON(it Item) string {
	b, err := json.Marshal(it)
	if err != nil {
		return ""
	}
	return string(b)
}
