// Package validation holds small, reusable validation helpers shared between the
// API handlers and the service worker so the same rules are enforced everywhere.
package validation

import (
	"fmt"
	"net/url"
	"slices"
	"strings"

	"paperinator/src/shared/models"
)

// Feed validates a feed's user-supplied fields. It returns a descriptive error
// suitable for surfacing to API clients, or nil if the feed is valid.
func Feed(f *models.Feed) error {
	if strings.TrimSpace(f.Name) == "" {
		return fmt.Errorf("feed name is required")
	}
	u, err := url.Parse(strings.TrimSpace(f.URL))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("feed url must be a valid http(s) URL")
	}
	if f.FetchIntervalSec < 0 {
		return fmt.Errorf("fetch_interval_sec must not be negative")
	}
	return nil
}

// FieldMapping validates that a mapping points at a known Publication field.
func FieldMapping(m *models.FieldMapping) error {
	if strings.TrimSpace(m.SourceField) == "" {
		return fmt.Errorf("source_field is required")
	}
	if !slices.Contains(models.ValidTargetFields, m.TargetField) {
		return fmt.Errorf("target_field %q is not a valid publication field", m.TargetField)
	}
	return nil
}
