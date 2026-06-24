// Package frontend embeds the static web UI so the whole application ships as a
// single self-contained binary. The API package serves these assets.
package frontend

import "embed"

// Static holds the browser assets under static/. Access subpaths via fs.Sub.
//
//go:embed static
var Static embed.FS
