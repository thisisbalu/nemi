package sitemap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/thisisbalu/nemi/internal/content"
)

func TestWrite(t *testing.T) {
	dir := t.TempDir()
	pages := []content.Page{
		{Slug: "blog/post", URL: "/blog/post/", Date: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)},
		{Slug: "", URL: "/"},
		{Slug: "404", URL: "/404/"}, // excluded
	}
	if err := Write(dir, "https://example.com/", pages); err != nil {
		t.Fatalf("Write: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "sitemap.xml"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	out := string(data)

	if !strings.HasPrefix(out, "<?xml") {
		t.Error("missing XML header")
	}
	if !strings.Contains(out, "<loc>https://example.com/blog/post/</loc>") {
		t.Errorf("missing absolute post URL:\n%s", out)
	}
	if !strings.Contains(out, "<lastmod>2026-01-02</lastmod>") {
		t.Errorf("missing lastmod:\n%s", out)
	}
	if !strings.Contains(out, "<loc>https://example.com/</loc>") {
		t.Errorf("missing home URL:\n%s", out)
	}
	if strings.Contains(out, "/404/") {
		t.Error("404 page should be excluded from sitemap")
	}
	// home (no date) should not carry an empty lastmod element
	if strings.Contains(out, "<lastmod></lastmod>") {
		t.Error("empty lastmod element should be omitted")
	}
}
