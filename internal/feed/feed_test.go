package feed

import (
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/thisisbalu/nemi/internal/config"
	"github.com/thisisbalu/nemi/internal/content"
)

func TestWrite(t *testing.T) {
	t.Run("writes RSS for blog posts, newest first", func(t *testing.T) {
		dir := t.TempDir()
		cfg := config.Config{Title: "My Site", BaseURL: "https://example.com/", Description: "About me"}
		pages := []content.Page{
			{Title: "Older", Section: "blog", URL: "/blog/older/", Date: date(2026, 1, 1), Content: template.HTML("<p>old</p>")},
			{Title: "Newer", Section: "blog", URL: "/blog/newer/", Date: date(2026, 2, 1), Content: template.HTML("<p>new</p>")},
			{Title: "Blog Index", Section: "blog", IsSection: true, URL: "/blog/"},
			{Title: "About", Section: "", URL: "/about/"},
		}
		if err := Write(dir, cfg, pages); err != nil {
			t.Fatalf("Write: %v", err)
		}
		out := readFeed(t, dir)

		if !strings.Contains(out, "<title>My Site</title>") {
			t.Error("missing channel title")
		}
		if !strings.Contains(out, "<link>https://example.com/blog/newer/</link>") {
			t.Errorf("missing absolute item link:\n%s", out)
		}
		// newest must appear before oldest
		if strings.Index(out, "Newer") > strings.Index(out, "Older") {
			t.Error("items not sorted newest-first")
		}
		// HTML content is XML-escaped inside <description>
		if !strings.Contains(out, "&lt;p&gt;new&lt;/p&gt;") {
			t.Errorf("expected escaped HTML in description:\n%s", out)
		}
		// section index and non-blog page excluded
		if strings.Contains(out, "<link>https://example.com/blog/</link>") || strings.Contains(out, "/about/") {
			t.Error("non-post pages should be excluded")
		}
	})

	t.Run("no blog posts means no feed file", func(t *testing.T) {
		dir := t.TempDir()
		pages := []content.Page{{Title: "About", Section: "", URL: "/about/"}}
		if err := Write(dir, config.Config{}, pages); err != nil {
			t.Fatalf("Write: %v", err)
		}
		if _, err := os.Stat(filepath.Join(dir, "feed.xml")); err == nil {
			t.Error("feed.xml should not exist when there are no blog posts")
		}
	})
}

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func readFeed(t *testing.T, dir string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "feed.xml"))
	if err != nil {
		t.Fatalf("read feed: %v", err)
	}
	return string(data)
}
