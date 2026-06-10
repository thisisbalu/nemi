// Package sitemap writes a sitemap.xml for the rendered pages.
package sitemap

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/thisisbalu/nemi/internal/content"
)

type urlset struct {
	XMLName xml.Name `xml:"urlset"`
	Xmlns   string   `xml:"xmlns,attr"`
	URLs    []url    `xml:"url"`
}

type url struct {
	Loc     string `xml:"loc"`
	Lastmod string `xml:"lastmod,omitempty"`
}

// Write renders a sitemap.xml into dir for the given pages. URLs are made
// absolute against baseURL. The 404 page is excluded. Entries are sorted by
// location for stable output.
func Write(dir, baseURL string, pages []content.Page) error {
	base := strings.TrimRight(baseURL, "/")
	set := urlset{Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9"}
	for _, p := range pages {
		if p.Slug == "404" {
			continue
		}
		entry := url{Loc: base + p.URL}
		if !p.Date.IsZero() {
			entry.Lastmod = p.Date.Format("2006-01-02")
		}
		set.URLs = append(set.URLs, entry)
	}
	sort.Slice(set.URLs, func(i, j int) bool {
		return set.URLs[i].Loc < set.URLs[j].Loc
	})

	body, err := xml.MarshalIndent(set, "", "  ")
	if err != nil {
		return err
	}
	out := append([]byte(xml.Header), body...)
	out = append(out, '\n')
	return os.WriteFile(filepath.Join(dir, "sitemap.xml"), out, 0644)
}
