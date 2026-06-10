package content

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- splitFrontmatter ---

func TestSplitFrontmatter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantFM   string
		wantBody string
	}{
		{
			name:     "normal frontmatter",
			input:    "---\ntitle: Hello\n---\nBody text",
			wantFM:   "title: Hello",
			wantBody: "Body text",
		},
		{
			name:     "frontmatter with trailing newline after closing delimiter",
			input:    "---\ntitle: Hi\n---\n\nParagraph",
			wantFM:   "title: Hi",
			wantBody: "\nParagraph",
		},
		{
			name:     "no opening delimiter",
			input:    "Just body text",
			wantFM:   "",
			wantBody: "Just body text",
		},
		{
			name:     "unclosed frontmatter",
			input:    "---\ntitle: Hello\n",
			wantFM:   "",
			wantBody: "---\ntitle: Hello\n",
		},
		{
			name:     "empty frontmatter block",
			input:    "---\n\n---\nBody",
			wantFM:   "",
			wantBody: "Body",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, body := splitFrontmatter([]byte(tt.input))
			if string(fm) != tt.wantFM {
				t.Errorf("fm = %q, want %q", fm, tt.wantFM)
			}
			if string(body) != tt.wantBody {
				t.Errorf("body = %q, want %q", body, tt.wantBody)
			}
		})
	}
}

// --- resolveRefs / @/ link resolution ---

func TestResolveRefs(t *testing.T) {
	pages := []Page{
		{URL: "/blog/post/", Content: ""},
		{URL: "/about/", Content: `<p>see <a href="@/blog/post.md">the post</a> and <a href="@/blog/post.md#part-2">part 2</a></p>`},
		{URL: "/", Content: `<p><a href="https://x.com">ext</a> and <a href="/plain/">plain</a></p>`},
	}
	srcPaths := []string{"blog/post.md", "about.md", "_index.md"}

	if err := resolveRefs(pages, srcPaths); err != nil {
		t.Fatalf("resolveRefs: %v", err)
	}

	got := string(pages[1].Content)
	if !strings.Contains(got, `href="/blog/post/"`) {
		t.Errorf("ref not resolved to URL: %s", got)
	}
	if !strings.Contains(got, `href="/blog/post/#part-2"`) {
		t.Errorf("fragment not preserved: %s", got)
	}
	if strings.Contains(got, "@/") {
		t.Errorf("@/ prefix should be gone: %s", got)
	}
	// untouched: external and plain links
	if string(pages[2].Content) != `<p><a href="https://x.com">ext</a> and <a href="/plain/">plain</a></p>` {
		t.Errorf("non-ref links altered: %s", pages[2].Content)
	}
}

func TestResolveRefsBroken(t *testing.T) {
	pages := []Page{
		{URL: "/about/", Content: `<a href="@/blog/missing.md">gone</a>`},
	}
	err := resolveRefs(pages, []string{"about.md"})
	if err == nil {
		t.Fatal("expected error for unresolved reference")
	}
	if !strings.Contains(err.Error(), "@/blog/missing.md") {
		t.Errorf("error should name the broken ref, got: %v", err)
	}
}

// --- OGImagePath ---

func TestOGImagePath(t *testing.T) {
	cases := map[string]string{
		"":          "/og/index.png",
		"about":     "/og/about.png",
		"blog/post": "/og/blog/post.png",
	}
	for slug, want := range cases {
		if got := OGImagePath(slug); got != want {
			t.Errorf("OGImagePath(%q) = %q, want %q", slug, got, want)
		}
	}
}

// --- Mermaid + math ---

func TestMermaidBlock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "diagram.md")
	body := "---\ntitle: D\n---\n\n```mermaid\ngraph TD; A-->B;\n```\n\n```go\nfmt.Println(\"hi\")\n```\n"
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	p, err := parseFile(path, dir)
	if err != nil {
		t.Fatalf("parseFile: %v", err)
	}
	html := string(p.Content)
	if !strings.Contains(html, `<pre class="mermaid">graph TD; A--&gt;B;`) {
		t.Errorf("mermaid block not rendered raw: %s", html)
	}
	if !p.Mermaid {
		t.Error("Mermaid flag should be set")
	}
	// the go block should still be syntax-highlighted (chroma), not mermaid
	if !strings.Contains(html, "chroma") {
		t.Errorf("non-mermaid code block should still be highlighted: %s", html)
	}
}

func TestMathFlag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "math.md")
	if err := os.WriteFile(path, []byte("---\ntitle: M\nmath: true\n---\n\nbody"), 0644); err != nil {
		t.Fatal(err)
	}
	p, err := parseFile(path, dir)
	if err != nil {
		t.Fatalf("parseFile: %v", err)
	}
	if !p.Math {
		t.Error("Math flag should be set from frontmatter")
	}
}

// --- pathSlug ---

func TestPathSlug(t *testing.T) {
	tests := []struct {
		path string
		dir  string
		want string
	}{
		{"content/index.md", "content", ""},
		{"content/_index.md", "content", ""},
		{"content/about.md", "content", "about"},
		{"content/blog/post.md", "content", "blog/post"},
		{"content/blog/index.md", "content", "blog"},
		{"content/blog/_index.md", "content", "blog"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := pathSlug(tt.path, tt.dir)
			if got != tt.want {
				t.Errorf("pathSlug(%q, %q) = %q, want %q", tt.path, tt.dir, got, tt.want)
			}
		})
	}
}

// --- slugURL ---

func TestSlugURL(t *testing.T) {
	tests := []struct{ slug, want string }{
		{"", "/"},
		{"about", "/about/"},
		{"blog/post", "/blog/post/"},
	}
	for _, tt := range tests {
		got := slugURL(tt.slug)
		if got != tt.want {
			t.Errorf("slugURL(%q) = %q, want %q", tt.slug, got, tt.want)
		}
	}
}

// --- pathSection ---

func TestPathSection(t *testing.T) {
	tests := []struct {
		path, dir, want string
	}{
		{"content/about.md", "content", ""},
		{"content/blog/post.md", "content", "blog"},
		{"content/blog/_index.md", "content", "blog"},
	}
	for _, tt := range tests {
		got := pathSection(tt.path, tt.dir)
		if got != tt.want {
			t.Errorf("pathSection(%q, %q) = %q, want %q", tt.path, tt.dir, got, tt.want)
		}
	}
}

// --- isIndexFile ---

func TestIsIndexFile(t *testing.T) {
	tests := []struct {
		path, dir string
		want      bool
	}{
		// root index.md is the home page, NOT a section index
		{"content/index.md", "content", false},
		{"content/_index.md", "content", false},
		// subdirectory index files ARE section indexes
		{"content/blog/index.md", "content", true},
		{"content/blog/_index.md", "content", true},
		{"content/blog/post.md", "content", false},
		{"content/about.md", "content", false},
	}
	for _, tt := range tests {
		got := isIndexFile(tt.path, tt.dir)
		if got != tt.want {
			t.Errorf("isIndexFile(%q, %q) = %v, want %v", tt.path, tt.dir, got, tt.want)
		}
	}
}

// --- resolveLayout ---

func TestResolveLayout(t *testing.T) {
	tests := []struct {
		slug, section string
		isSection     bool
		want          string
	}{
		{"", "", false, "home"},
		{"about", "", false, "about"},
		{"resume", "", false, "resume"},
		{"now", "", false, "now"},
		{"uses", "", false, "uses"},
		{"speaking", "", false, "speaking"},
		{"contact", "", false, "contact"},
		{"open-source", "", false, "open-source"},
		{"blog", "blog", true, "blog-list"},
		{"projects", "projects", true, "projects-list"},
		{"blog/my-post", "blog", false, "blog-single"},
		{"my-page", "", false, "single"},
	}
	for _, tt := range tests {
		got := resolveLayout(tt.slug, tt.section, tt.isSection)
		if got != tt.want {
			t.Errorf("resolveLayout(%q, %q, %v) = %q, want %q",
				tt.slug, tt.section, tt.isSection, got, tt.want)
		}
	}
}

// --- parseFile ---

func TestParseFile(t *testing.T) {
	t.Run("full frontmatter", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "blog", "hello.md")
		writeFile(t, path, "---\ntitle: Hello\ndate: 2024-03-15\ntags: [go, web]\ndescription: A post\n---\n# Hello\n\nWorld")

		page, err := parseFile(path, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if page.Title != "Hello" {
			t.Errorf("Title = %q", page.Title)
		}
		if len(page.Tags) != 2 || page.Tags[0] != "go" || page.Tags[1] != "web" {
			t.Errorf("Tags = %v", page.Tags)
		}
		if page.Description != "A post" {
			t.Errorf("Description = %q", page.Description)
		}
		if page.Slug != "blog/hello" {
			t.Errorf("Slug = %q", page.Slug)
		}
		if page.URL != "/blog/hello/" {
			t.Errorf("URL = %q", page.URL)
		}
		if page.Section != "blog" {
			t.Errorf("Section = %q", page.Section)
		}
		if page.IsSection {
			t.Error("IsSection should be false for a regular post")
		}
		if string(page.Content) == "" {
			t.Error("Content is empty")
		}
		expectedDate := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
		if !page.Date.Equal(expectedDate) {
			t.Errorf("Date = %v, want %v", page.Date, expectedDate)
		}
	})

	t.Run("fenced code block is syntax highlighted", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "blog", "code.md")
		writeFile(t, path, "---\ntitle: Code\n---\n```go\nfunc main() {}\n```\n")

		page, err := parseFile(path, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		html := string(page.Content)
		// Chroma emits class-based token spans wrapped in <pre class="chroma ...">.
		if !strings.Contains(html, `class="chroma`) {
			t.Errorf("expected chroma wrapper in output, got: %s", html)
		}
		if !strings.Contains(html, `<span class="kd">func</span>`) {
			t.Errorf("expected highlighted keyword span, got: %s", html)
		}
	})

	t.Run("headings get ids and a table of contents", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "blog", "toc.md")
		writeFile(t, path, "---\ntitle: TOC\n---\n## First Section\n\ntext\n\n### Nested\n\ntext\n\n## Second Section\n\ntext\n")

		page, err := parseFile(path, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(string(page.Content), `<h2 id="first-section">`) {
			t.Errorf("expected auto heading id in content, got: %s", page.Content)
		}
		if len(page.Headings) != 3 {
			t.Fatalf("Headings = %d, want 3", len(page.Headings))
		}
		if page.Headings[1].Level != 3 || page.Headings[1].ID != "nested" || page.Headings[1].Text != "Nested" {
			t.Errorf("Headings[1] = %+v", page.Headings[1])
		}
		toc := string(page.TOC)
		if !strings.Contains(toc, `<a href="#first-section">First Section</a>`) {
			t.Errorf("TOC missing first section link: %s", toc)
		}
		if !strings.Contains(toc, "<ul><li><a") || !strings.Contains(toc, "</ul></li>") {
			t.Errorf("TOC nesting looks wrong: %s", toc)
		}
	})

	t.Run("single heading produces no TOC", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "blog", "one.md")
		writeFile(t, path, "---\ntitle: One\n---\n## Only One\n\ntext\n")

		page, err := parseFile(path, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if page.TOC != "" {
			t.Errorf("expected empty TOC for one heading, got: %s", page.TOC)
		}
	})

	t.Run("GFM, footnote and typographer extensions", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "blog", "gfm.md")
		writeFile(t, path,
			"---\ntitle: GFM\n---\n"+
				"| A | B |\n|---|---|\n| 1 | 2 |\n\n"+
				"- [ ] todo\n- [x] done\n\n"+
				"~~gone~~\n\n"+
				"Note.[^1]\n\n[^1]: The note.\n\n"+
				"\"quoted\"\n")

		page, err := parseFile(path, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := string(page.Content)
		checks := map[string]string{
			"table":            "<table>",
			"task checkbox":    `type="checkbox"`,
			"strikethrough":    "<del>gone</del>",
			"footnote ref":     `class="footnote-ref"`,
			"footnote section": `class="footnotes"`,
			"smart quotes":     "&ldquo;quoted&rdquo;",
		}
		for name, want := range checks {
			if !strings.Contains(out, want) {
				t.Errorf("%s: expected %q in output, got: %s", name, want, out)
			}
		}
	})

	t.Run("slug override changes URL but not section or layout", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "blog", "original-name.md")
		writeFile(t, path, "---\ntitle: Renamed\nslug: \"/custom/path/\"\n---\nBody")

		page, err := parseFile(path, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if page.Slug != "custom/path" {
			t.Errorf("Slug = %q, want %q (trimmed)", page.Slug, "custom/path")
		}
		if page.URL != "/custom/path/" {
			t.Errorf("URL = %q, want /custom/path/", page.URL)
		}
		if page.Section != "blog" {
			t.Errorf("Section = %q, want blog (from path, not slug)", page.Section)
		}
		if page.Layout != "blog-single" {
			t.Errorf("Layout = %q, want blog-single (from path, not slug)", page.Layout)
		}
	})

	t.Run("computes summary, word count and reading time", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "blog", "long.md")
		// First paragraph is 80 words (> 60 → truncated); a second paragraph and
		// heading push the total higher for word count / reading time.
		firstPara := strings.TrimSpace(strings.Repeat("alpha ", 80))
		rest := strings.Repeat("word ", 400)
		writeFile(t, path, "---\ntitle: Long\n---\n## Heading\n\n"+firstPara+"\n\n"+rest)

		page, err := parseFile(path, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if page.WordCount < 480 {
			t.Errorf("WordCount = %d, want >= 480", page.WordCount)
		}
		if page.ReadingTime != (page.WordCount+199)/200 {
			t.Errorf("ReadingTime = %d, inconsistent with WordCount %d", page.ReadingTime, page.WordCount)
		}
		if !strings.HasPrefix(page.Summary, "alpha") {
			t.Errorf("Summary should come from the first paragraph, got: %q", page.Summary)
		}
		if !strings.HasSuffix(page.Summary, "…") {
			t.Errorf("long summary should be truncated with an ellipsis, got: %q", page.Summary)
		}
		if n := len(strings.Fields(page.Summary)); n != 60 {
			t.Errorf("summary should be 60 words, got %d", n)
		}
	})

	t.Run("short post has no ellipsis and a 1 minute read", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "blog", "short.md")
		writeFile(t, path, "---\ntitle: Short\n---\nJust a few words here.")

		page, err := parseFile(path, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if page.ReadingTime != 1 {
			t.Errorf("ReadingTime = %d, want 1", page.ReadingTime)
		}
		if page.Summary != "Just a few words here." {
			t.Errorf("Summary = %q", page.Summary)
		}
	})

	t.Run("layout override", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "custom.md")
		writeFile(t, path, "---\ntitle: Custom\nlayout: my-layout\n---\nContent")
		page, err := parseFile(path, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if page.Layout != "my-layout" {
			t.Errorf("Layout = %q, want my-layout", page.Layout)
		}
	})

	t.Run("project fields", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "project.md")
		writeFile(t, path, "---\ntitle: Proj\ntech: [go, react]\ngithub: user/repo\nlive: https://example.com\nfeatured: true\ntype: project\n---\nBody")
		page, err := parseFile(path, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(page.Tech) != 2 {
			t.Errorf("Tech = %v", page.Tech)
		}
		if page.GitHub != "user/repo" {
			t.Errorf("GitHub = %q", page.GitHub)
		}
		if page.Live != "https://example.com" {
			t.Errorf("Live = %q", page.Live)
		}
		if !page.Featured {
			t.Error("Featured should be true")
		}
		if page.Type != "project" {
			t.Errorf("Type = %q", page.Type)
		}
	})

	t.Run("no frontmatter", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "bare.md")
		writeFile(t, path, "# Just markdown\n\nNo frontmatter.")
		page, err := parseFile(path, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if page.Title != "" {
			t.Errorf("Title should be empty, got %q", page.Title)
		}
		if string(page.Content) == "" {
			t.Error("Content should not be empty")
		}
	})

	t.Run("draft flag", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "draft.md")
		writeFile(t, path, "---\ntitle: Draft\ndraft: true\n---\nContent")
		page, err := parseFile(path, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !page.Draft {
			t.Error("Draft should be true")
		}
	})

	t.Run("section index file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "blog", "_index.md")
		writeFile(t, path, "---\ntitle: Blog\n---\nBlog section")
		page, err := parseFile(path, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !page.IsSection {
			t.Error("IsSection should be true for _index.md")
		}
		if page.Layout != "blog-list" {
			t.Errorf("Layout = %q, want blog-list", page.Layout)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := parseFile("/nonexistent/file.md", "/nonexistent")
		if err == nil {
			t.Error("expected error for missing file")
		}
	})

	t.Run("invalid frontmatter", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "bad.md")
		writeFile(t, path, "---\ntitle: [invalid yaml\n---\nContent")
		_, err := parseFile(path, dir)
		if err == nil {
			t.Error("expected error for invalid YAML frontmatter")
		}
	})
}

// --- Load ---

func TestLoad(t *testing.T) {
	t.Run("multiple pages", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "index.md"), "---\ntitle: Home\n---\nWelcome")
		writeFile(t, filepath.Join(dir, "about.md"), "---\ntitle: About\n---\nAbout")
		writeFile(t, filepath.Join(dir, "blog", "post.md"), "---\ntitle: Post\n---\nPost")

		pages, err := Load(dir, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(pages) != 3 {
			t.Errorf("got %d pages, want 3", len(pages))
		}
	})

	t.Run("excludes drafts by default", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "pub.md"), "---\ntitle: Published\n---\nBody")
		writeFile(t, filepath.Join(dir, "draft.md"), "---\ntitle: Draft\ndraft: true\n---\nBody")

		pages, err := Load(dir, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(pages) != 1 || pages[0].Title != "Published" {
			t.Errorf("expected 1 published page, got %d", len(pages))
		}
	})

	t.Run("includes drafts when flag set", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "pub.md"), "---\ntitle: Published\n---\nBody")
		writeFile(t, filepath.Join(dir, "draft.md"), "---\ntitle: Draft\ndraft: true\n---\nBody")

		pages, err := Load(dir, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(pages) != 2 {
			t.Errorf("expected 2 pages with drafts, got %d", len(pages))
		}
	})

	t.Run("nonexistent dir returns error", func(t *testing.T) {
		_, err := Load(filepath.Join(t.TempDir(), "nonexistent"), false)
		if err == nil {
			t.Error("expected error for nonexistent dir")
		}
	})

	t.Run("invalid frontmatter propagates error", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "bad.md"), "---\ntitle: [bad\n---\nBody")
		_, err := Load(dir, false)
		if err == nil {
			t.Error("expected error for invalid frontmatter")
		}
	})
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
