package serviceWorker

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// fetchClient is a package-level client with a sane timeout, reused across feeds.
var fetchClient = &http.Client{Timeout: 30 * time.Second}

// maxFeedBytes caps how much we read from a single feed response to avoid
// unbounded memory use on a misbehaving or hostile endpoint (8 MiB).
const maxFeedBytes = 8 << 20

// fetch retrieves the raw bytes of a feed URL.
func fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "paper-inator/0.1 (+https://github.com/)")
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml;q=0.9")

	resp, err := fetchClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxFeedBytes))
}
