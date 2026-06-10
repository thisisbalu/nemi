package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thisisbalu/nemi/internal/config"
)

// --- mapStr ---

func TestMapStr(t *testing.T) {
	m := map[string]any{"key": "val", "num": 42}
	if got := mapStr(m, "key"); got != "val" {
		t.Errorf("got %q, want val", got)
	}
	if got := mapStr(m, "num"); got != "" {
		t.Errorf("non-string should return empty, got %q", got)
	}
	if got := mapStr(m, "missing"); got != "" {
		t.Errorf("missing key should return empty, got %q", got)
	}
	if got := mapStr(nil, "key"); got != "" {
		t.Errorf("nil map should return empty, got %q", got)
	}
}

// --- mapMap ---

func TestMapMap(t *testing.T) {
	inner := map[string]any{"x": "y"}
	m := map[string]any{"params": inner, "str": "hello"}
	if got := mapMap(m, "params"); got == nil {
		t.Error("expected non-nil map")
	}
	if got := mapMap(m, "str"); got != nil {
		t.Error("non-map value should return nil")
	}
	if got := mapMap(m, "missing"); got != nil {
		t.Error("missing key should return nil")
	}
	if got := mapMap(nil, "params"); got != nil {
		t.Error("nil map should return nil")
	}
}

// --- firstNonEmpty ---

func TestFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("", "", "third"); got != "third" {
		t.Errorf("got %q, want third", got)
	}
	if got := firstNonEmpty("first", "second"); got != "first" {
		t.Errorf("got %q, want first", got)
	}
	if got := firstNonEmpty("", ""); got != "" {
		t.Errorf("all empty should return empty, got %q", got)
	}
	if got := firstNonEmpty(); got != "" {
		t.Errorf("no args should return empty, got %q", got)
	}
}

func TestMergeTags(t *testing.T) {
	join := func(s []string) string { return strings.Join(s, ",") }
	cases := []struct {
		tags, cats []string
		want       string
	}{
		{[]string{"go", "web"}, []string{"tutorial"}, "go,web,tutorial"}, // both kept
		{[]string{"go"}, []string{"go", "cli"}, "go,cli"},                // de-duplicated
		{nil, []string{"notes"}, "notes"},                                // categories only
		{[]string{"go"}, nil, "go"},                                      // tags only
		{[]string{"", "go", ""}, []string{""}, "go"},                     // empties dropped
		{nil, nil, ""},                                                   // nothing
	}
	for _, c := range cases {
		if got := join(mergeTags(c.tags, c.cats)); got != c.want {
			t.Errorf("mergeTags(%v, %v) = %q, want %q", c.tags, c.cats, got, c.want)
		}
	}
}

// --- HTML migration ---

func TestStripLeadingH1(t *testing.T) {
	if got := stripLeadingH1("# Title\n\nBody"); got != "Body" {
		t.Errorf("leading h1 not stripped: %q", got)
	}
	if got := stripLeadingH1("Intro\n# Later"); got != "Intro\n# Later" {
		t.Errorf("non-leading h1 should be kept: %q", got)
	}
}

func TestTitleFromDir(t *testing.T) {
	cases := map[string]string{
		"my-old-site":  "My Old Site",
		"/tmp/foo_bar": "Foo Bar",
		"about.html":   "About",
	}
	for in, want := range cases {
		if got := titleFromDir(in); got != want {
			t.Errorf("titleFromDir(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsRelativeRef(t *testing.T) {
	for in, want := range map[string]bool{
		"images/x.png":   true,
		"./a.png":        true,
		"/abs.png":       false,
		"http://x.com":   false,
		"//cdn/x":        false,
		"#frag":          false,
		"mailto:a@b.com": false,
		"":               false,
	} {
		if got := isRelativeRef(in); got != want {
			t.Errorf("isRelativeRef(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestResolveRef(t *testing.T) {
	cases := []struct {
		pageRel, ref string
		isLink       bool
		want         string
	}{
		{"index.html", "about.html", true, "/about/"},
		{"index.html", "blog/post.html", true, "/blog/post/"},
		{"blog/post.html", "other.html", true, "/blog/other/"},
		{"blog/post.html", "../about.html", true, "/about/"},
		{"about.html", "index.html", true, "/"},
		{"about.html", "page.html#sec", true, "/page/#sec"},
		{"index.html", "img/a.png", false, "/img/a.png"},
		{"blog/post.html", "pic.png", false, "/blog/pic.png"},
	}
	for _, c := range cases {
		if got := resolveRef(c.pageRel, c.ref, c.isLink); got != c.want {
			t.Errorf("resolveRef(%q, %q, %v) = %q, want %q", c.pageRel, c.ref, c.isLink, got, c.want)
		}
	}
}

func TestHTML(t *testing.T) {
	src := t.TempDir()
	writeF(t, filepath.Join(src, "index.html"),
		`<html><head><title>Home</title></head><body><nav>skipnav</nav>`+
			`<main><h1>Home</h1><p>Hi <b>there</b>. See <a href="about.html">about</a>.</p>`+
			`<img src="img/a.png" alt="A"></main>`+
			`<footer>skipfoot</footer></body></html>`)
	writeF(t, filepath.Join(src, "img", "a.png"), "PNG")

	dst := filepath.Join(t.TempDir(), "out")
	r, err := HTML(src, dst)
	if err != nil {
		t.Fatalf("HTML: %v", err)
	}
	if r.Pages != 1 || r.Static != 1 {
		t.Errorf("counts: pages=%d static=%d, want 1/1", r.Pages, r.Static)
	}

	md, err := os.ReadFile(filepath.Join(dst, "content", "index.md"))
	if err != nil {
		t.Fatalf("read index.md: %v", err)
	}
	s := string(md)
	if !strings.Contains(s, "title: Home") {
		t.Errorf("missing title frontmatter: %s", s)
	}
	if strings.Contains(s, "# Home") {
		t.Errorf("leading h1 should be stripped (dups title): %s", s)
	}
	if !strings.Contains(s, "Hi **there**.") {
		t.Errorf("body not converted to markdown: %s", s)
	}
	if !strings.Contains(s, "/img/a.png") {
		t.Errorf("image ref not absolutized: %s", s)
	}
	if !strings.Contains(s, "(/about/)") {
		t.Errorf("relative .html link not rewritten to Nemi URL: %s", s)
	}
	if strings.Contains(s, "skipnav") || strings.Contains(s, "skipfoot") {
		t.Errorf("nav/footer chrome should be dropped: %s", s)
	}
	if _, err := os.Stat(filepath.Join(dst, "static", "img", "a.png")); err != nil {
		t.Error("image not copied to static/")
	}
	if _, err := os.Stat(filepath.Join(dst, "layouts", "base.html")); err != nil {
		t.Error("scaffold theme (layouts/base.html) not copied")
	}
}

// --- writePage ---

func TestWritePage(t *testing.T) {
	t.Run("creates file with frontmatter and body", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "post.md")
		fm := nemiFM{Title: "Hello", Date: "2024-01-01", Tags: []string{"go"}}
		body := []byte("# Hello\n\nWorld")

		if err := writePage(path, fm, body); err != nil {
			t.Fatalf("writePage: %v", err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		s := string(data)
		if !strings.Contains(s, "title: Hello") {
			t.Errorf("expected title in frontmatter, got: %s", s)
		}
		if !strings.Contains(s, "2024-01-01") {
			t.Errorf("expected date in frontmatter, got: %s", s)
		}
		if !strings.Contains(s, "# Hello") {
			t.Errorf("expected body, got: %s", s)
		}
	})

	t.Run("creates parent directories", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "nested", "deep", "post.md")
		if err := writePage(path, nemiFM{Title: "Deep"}, []byte("body")); err != nil {
			t.Fatalf("writePage: %v", err)
		}
		if _, err := os.Stat(path); err != nil {
			t.Error("file should exist in nested dir")
		}
	})

	t.Run("MkdirAll error returns error", func(t *testing.T) {
		blocker := filepath.Join(t.TempDir(), "blocker")
		os.WriteFile(blocker, []byte("x"), 0644)
		// parent path component is a file, so MkdirAll fails
		if err := writePage(filepath.Join(blocker, "post.md"), nemiFM{Title: "T"}, []byte("body")); err == nil {
			t.Error("expected error when parent dir cannot be created")
		}
	})

	t.Run("Create error returns error", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "post.md")
		os.MkdirAll(target, 0755) // target is a directory, not a writable file
		if err := writePage(target, nemiFM{Title: "T"}, []byte("body")); err == nil {
			t.Error("expected error when target path is a directory")
		}
	})

	t.Run("omits empty fields", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "minimal.md")
		if err := writePage(path, nemiFM{Title: "Min"}, []byte("")); err != nil {
			t.Fatalf("writePage: %v", err)
		}
		data, _ := os.ReadFile(path)
		s := string(data)
		if strings.Contains(s, "date:") {
			t.Error("empty date should be omitted")
		}
		if strings.Contains(s, "tags:") {
			t.Error("empty tags should be omitted")
		}
	})
}

// --- writeNemiToml ---

func TestWriteNemiToml(t *testing.T) {
	t.Run("writes all author fields", func(t *testing.T) {
		dir := t.TempDir()
		cfg := config.Config{
			Title:       "My Site",
			BaseURL:     "https://example.com",
			Description: "A site",
			Author: config.Author{
				Name:     "Alice",
				Email:    "alice@example.com",
				GitHub:   "alice",
				Twitter:  "alicetw",
				LinkedIn: "aliceli",
			},
		}
		if err := writeNemiToml(dir, cfg); err != nil {
			t.Fatalf("writeNemiToml: %v", err)
		}
		data, _ := os.ReadFile(filepath.Join(dir, "nemi.toml"))
		s := string(data)
		for _, want := range []string{
			`title       = "My Site"`,
			`baseURL     = "https://example.com"`,
			`description = "A site"`,
			`name     = "Alice"`,
			`email    = "alice@example.com"`,
			`github   = "alice"`,
			`twitter  = "alicetw"`,
			`linkedin = "aliceli"`,
			"[pages]",
			"about   = true",
			"blog    = true",
		} {
			if !strings.Contains(s, want) {
				t.Errorf("nemi.toml missing %q\ngot:\n%s", want, s)
			}
		}
	})

	t.Run("skips empty author fields", func(t *testing.T) {
		dir := t.TempDir()
		if err := writeNemiToml(dir, config.Config{Title: "X"}); err != nil {
			t.Fatalf("writeNemiToml: %v", err)
		}
		data, _ := os.ReadFile(filepath.Join(dir, "nemi.toml"))
		s := string(data)
		if strings.Contains(s, "email") || strings.Contains(s, "github") {
			t.Error("empty author fields should not be written")
		}
	})

	t.Run("missing dest dir returns error", func(t *testing.T) {
		err := writeNemiToml(filepath.Join(t.TempDir(), "nonexistent"), config.Config{})
		if err == nil {
			t.Error("expected error for nonexistent dest dir")
		}
	})
}

// --- copyFile ---

func TestCopyFile(t *testing.T) {
	t.Run("copies content", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "src.txt")
		dst := filepath.Join(dir, "sub", "dst.txt")
		if err := os.WriteFile(src, []byte("hello"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := copyFile(src, dst); err != nil {
			t.Fatalf("copyFile: %v", err)
		}
		data, _ := os.ReadFile(dst)
		if string(data) != "hello" {
			t.Errorf("got %q, want hello", data)
		}
	})

	t.Run("missing source returns error", func(t *testing.T) {
		dir := t.TempDir()
		err := copyFile(filepath.Join(dir, "missing.txt"), filepath.Join(dir, "dst.txt"))
		if err == nil {
			t.Error("expected error for missing source")
		}
	})

	t.Run("dst is directory returns error", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "src.txt")
		os.WriteFile(src, []byte("data"), 0644)
		dst := filepath.Join(dir, "dst_dir")
		os.MkdirAll(dst, 0755) // dst is a directory, Create will fail
		if err := copyFile(src, dst); err == nil {
			t.Error("expected error when dst is a directory")
		}
	})

	t.Run("MkdirAll error returns error", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "src.txt")
		os.WriteFile(src, []byte("data"), 0644)
		blocker := filepath.Join(dir, "blocker")
		os.WriteFile(blocker, []byte("x"), 0644)
		// filepath.Dir(blocker/dst.txt) = blocker (a file, not a directory)
		if err := copyFile(src, filepath.Join(blocker, "dst.txt")); err == nil {
			t.Error("expected error when parent dir cannot be created")
		}
	})
}

// --- copyDir ---

func TestCopyDir(t *testing.T) {
	t.Run("copies all files recursively", func(t *testing.T) {
		src := t.TempDir()
		dst := t.TempDir()
		writeF(t, filepath.Join(src, "a.txt"), "a")
		writeF(t, filepath.Join(src, "sub", "b.txt"), "b")

		n, err := copyDir(src, dst)
		if err != nil {
			t.Fatalf("copyDir: %v", err)
		}
		if n != 2 {
			t.Errorf("count = %d, want 2", n)
		}
		data, _ := os.ReadFile(filepath.Join(dst, "sub", "b.txt"))
		if string(data) != "b" {
			t.Errorf("got %q, want b", data)
		}
	})

	t.Run("nonexistent src returns zero count no error", func(t *testing.T) {
		n, err := copyDir(filepath.Join(t.TempDir(), "missing"), t.TempDir())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n != 0 {
			t.Errorf("count = %d, want 0", n)
		}
	})

	t.Run("propagates copyFile error", func(t *testing.T) {
		src := t.TempDir()
		dst := t.TempDir()
		writeF(t, filepath.Join(src, "file.txt"), "data")
		// Create a directory at dst/file.txt so os.Create fails inside copyFile
		os.MkdirAll(filepath.Join(dst, "file.txt"), 0755)
		_, err := copyDir(src, dst)
		if err == nil {
			t.Error("expected error when copyFile fails during walk")
		}
	})
}

func writeF(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
