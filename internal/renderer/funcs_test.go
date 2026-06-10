package renderer

import (
	"testing"
	"time"

	"github.com/thisisbalu/nemi/internal/content"
)

func TestURLize(t *testing.T) {
	cases := map[string]string{
		"Hello World":     "hello-world",
		"  Spaced  Out  ": "spaced-out",
		"Go & Rust!":      "go-rust",
		"Already-slug":    "already-slug",
		"":                "",
		"C++":             "c",
	}
	for in, want := range cases {
		if got := content.Urlize(in); got != want {
			t.Errorf("Urlize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTruncate(t *testing.T) {
	cases := []struct {
		n    int
		in   string
		want string
	}{
		{5, "hi", "hi"},                    // shorter than n
		{5, "hello", "hello"},              // exactly n
		{8, "hello world", "hello…"},       // backs up to word boundary
		{11, "hello world", "hello world"}, // fits
		{4, "abcdefgh", "abcd…"},           // no space → hard cut
	}
	for _, c := range cases {
		if got := truncate(c.n, c.in); got != c.want {
			t.Errorf("truncate(%d, %q) = %q, want %q", c.n, c.in, got, c.want)
		}
	}
}

func TestTitle(t *testing.T) {
	if got := title("hello brave world"); got != "Hello Brave World" {
		t.Errorf("title() = %q", got)
	}
}

func TestDateFormat(t *testing.T) {
	d := time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC)
	if got := dateFormat("2006-01-02", d); got != "2026-06-09" {
		t.Errorf("dateFormat() = %q", got)
	}
}

func TestFirst(t *testing.T) {
	xs := []int{1, 2, 3, 4}
	got := first(2, xs).([]int)
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Errorf("first(2) = %v", got)
	}
	// n greater than length returns whole slice
	if g := first(10, xs).([]int); len(g) != 4 {
		t.Errorf("first(10) len = %d", len(g))
	}
	// non-slice returned unchanged
	if g := first(2, "scalar"); g != "scalar" {
		t.Errorf("first on non-slice = %v", g)
	}
}

func TestUniqueTags(t *testing.T) {
	pages := []content.Page{
		{Tags: []string{"go", "web"}},
		{Tags: []string{"web", "rust"}},
		{Tags: nil},
		{Tags: []string{"go"}},
	}
	got := uniqueTags(pages)
	want := []string{"go", "rust", "web"} // de-duplicated and sorted
	if len(got) != len(want) {
		t.Fatalf("uniqueTags = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("uniqueTags[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestWhere(t *testing.T) {
	pages := []content.Page{
		{Title: "A", Section: "blog", Featured: true},
		{Title: "B", Section: "blog", Featured: false},
		{Title: "C", Section: "projects", Featured: true},
	}
	if got := where(pages, "Section", "blog"); len(got) != 2 {
		t.Errorf("where Section=blog returned %d, want 2", len(got))
	}
	if got := where(pages, "Featured", true); len(got) != 2 {
		t.Errorf("where Featured=true returned %d, want 2", len(got))
	}
	if got := where(pages, "Nonexistent", "x"); len(got) != 0 {
		t.Errorf("where on unknown field returned %d, want 0", len(got))
	}
}
