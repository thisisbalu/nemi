package search

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/thisisbalu/nemi/internal/content"
)

func TestBuild(t *testing.T) {
	pages := []content.Page{
		{Title: "Hello", URL: "/blog/hello/", Tags: []string{"go"}, Summary: "hi", Content: "<p>Body <strong>text</strong> &amp; more.</p>"},
		{Title: "", URL: "/skip/", Content: "no title"},        // skipped: no title
		{Title: "Oops", Slug: "404", URL: "/404", Content: "x"}, // skipped: 404
	}
	got := Build(pages)
	if len(got) != 1 {
		t.Fatalf("entries = %d, want 1 (titleless + 404 skipped)", len(got))
	}
	e := got[0]
	if e.Title != "Hello" || e.URL != "/blog/hello/" {
		t.Errorf("unexpected entry: %+v", e)
	}
	if e.Body != "Body text & more." {
		t.Errorf("body = %q, want HTML stripped and entities decoded", e.Body)
	}
}

func TestWrite(t *testing.T) {
	dir := t.TempDir()
	if err := Write(dir, []content.Page{{Title: "A", URL: "/a/", Content: "<p>x</p>"}}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "search-index.json"))
	if err != nil {
		t.Fatal(err)
	}
	var entries []Entry
	if err := json.Unmarshal(raw, &entries); err != nil {
		t.Fatalf("index is not valid JSON: %v", err)
	}
	if len(entries) != 1 || entries[0].Title != "A" {
		t.Errorf("unexpected index: %+v", entries)
	}
}
