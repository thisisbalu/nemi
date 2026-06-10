// Package feed writes an RSS 2.0 feed for the blog section.
package feed

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/thisisbalu/nemi/internal/config"
	"github.com/thisisbalu/nemi/internal/content"
)

// section is the content section published as a feed, and maxItems caps how
// many of its most recent posts appear.
const (
	section  = "blog"
	maxItems = 20
)

type rss struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel channel  `xml:"channel"`
}

type channel struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	LastBuild   string `xml:"lastBuildDate,omitempty"`
	Items       []item `xml:"item"`
}

type item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	GUID        string `xml:"guid"`
	PubDate     string `xml:"pubDate,omitempty"`
	Description string `xml:"description"`
}

// Write emits public/feed.xml for the most recent blog posts. It is a no-op
// (no file written) when there are no posts in the blog section.
func Write(dir string, cfg config.Config, pages []content.Page) error {
	var posts []content.Page
	for _, p := range pages {
		if p.Section == section && !p.IsSection {
			posts = append(posts, p)
		}
	}
	if len(posts) == 0 {
		return nil
	}
	sort.Slice(posts, func(i, j int) bool { return posts[i].Date.After(posts[j].Date) })
	if len(posts) > maxItems {
		posts = posts[:maxItems]
	}

	base := strings.TrimRight(cfg.BaseURL, "/")
	ch := channel{
		Title:       cfg.Title,
		Link:        base + "/",
		Description: cfg.Description,
	}
	if !posts[0].Date.IsZero() {
		ch.LastBuild = posts[0].Date.Format(time.RFC1123Z)
	}
	for _, p := range posts {
		link := base + p.URL
		desc := string(p.Content)
		if desc == "" {
			desc = p.Description
		}
		it := item{Title: p.Title, Link: link, GUID: link, Description: desc}
		if !p.Date.IsZero() {
			it.PubDate = p.Date.Format(time.RFC1123Z)
		}
		ch.Items = append(ch.Items, it)
	}

	body, err := xml.MarshalIndent(rss{Version: "2.0", Channel: ch}, "", "  ")
	if err != nil {
		return err
	}
	out := append([]byte(xml.Header), body...)
	out = append(out, '\n')
	return os.WriteFile(filepath.Join(dir, "feed.xml"), out, 0644)
}
