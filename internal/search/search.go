// Package search builds a static JSON search index from a site's content so the
// theme's client-side search works with zero external dependencies — no Rust
// binary, no service. Suited to personal/portfolio sites (tens–hundreds of
// pages); the whole index is fetched once and filtered in the browser.
package search

import (
	"encoding/json"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/thisisbalu/nemi/internal/content"
)

// Entry is one searchable page in the index.
type Entry struct {
	Title   string   `json:"title"`
	URL     string   `json:"url"`
	Tags    []string `json:"tags,omitempty"`
	Summary string   `json:"summary,omitempty"`
	Body    string   `json:"body"`
}

var tagRe = regexp.MustCompile(`<[^>]*>`)

// plain strips HTML tags and collapses whitespace into searchable text.
func plain(htmlStr string) string {
	s := tagRe.ReplaceAllString(htmlStr, " ")
	s = html.UnescapeString(s)
	return strings.Join(strings.Fields(s), " ")
}

// Build returns the index entries for the given content pages, skipping the 404
// page and anything without a title. basePath is prepended to each entry URL so
// result links resolve under a subpath deploy (empty for root deploys).
func Build(pages []content.Page, basePath string) []Entry {
	entries := make([]Entry, 0, len(pages))
	for _, p := range pages {
		if p.Title == "" || p.Slug == "404" {
			continue
		}
		entries = append(entries, Entry{
			Title:   p.Title,
			URL:     basePath + p.URL,
			Tags:    p.Tags,
			Summary: p.Summary,
			Body:    plain(string(p.Content)),
		})
	}
	return entries
}

// Write marshals the index and writes it to publicDir/search-index.json.
func Write(publicDir, basePath string, pages []content.Page) error {
	b, err := json.Marshal(Build(pages, basePath))
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(publicDir, "search-index.json"), b, 0644)
}
