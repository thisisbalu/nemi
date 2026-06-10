package renderer

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/thisisbalu/nemi/internal/config"
	"github.com/thisisbalu/nemi/internal/content"
)

// --- outputPath ---

func TestOutputPath(t *testing.T) {
	r := &Renderer{outputDir: "public"}
	tests := []struct{ slug, want string }{
		{"", "public/index.html"},
		{"404", "public/404.html"},
		{"about", "public/about/index.html"},
		{"blog/my-post", "public/blog/my-post/index.html"},
	}
	for _, tt := range tests {
		got := r.outputPath(tt.slug)
		if filepath.ToSlash(got) != tt.want {
			t.Errorf("outputPath(%q) = %q, want %q", tt.slug, got, tt.want)
		}
	}
}

// --- capitalize ---

func TestCapitalize(t *testing.T) {
	tests := []struct{ in, want string }{
		{"", ""},
		{"blog", "Blog"},
		{"Blog", "Blog"},
		{"my-section", "My-section"},
	}
	for _, tt := range tests {
		got := capitalize(tt.in)
		if got != tt.want {
			t.Errorf("capitalize(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// --- sortByDate ---

func TestSortByDate(t *testing.T) {
	pages := []content.Page{
		{Title: "Old", Date: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Title: "New", Date: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)},
		{Title: "Mid", Date: time.Date(2023, 3, 15, 0, 0, 0, 0, time.UTC)},
	}
	sorted := sortByDate(pages)
	if sorted[0].Title != "New" || sorted[1].Title != "Mid" || sorted[2].Title != "Old" {
		t.Errorf("wrong sort order: %v %v %v", sorted[0].Title, sorted[1].Title, sorted[2].Title)
	}
	// original slice must not be modified
	if pages[0].Title != "Old" {
		t.Error("sortByDate modified the original slice")
	}
}

// --- templateContent ---

func TestTemplateContent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "base.html"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	r := &Renderer{layoutsDir: dir}

	t.Run("found", func(t *testing.T) {
		got, err := r.templateContent("base.html")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "hello" {
			t.Errorf("got %q, want %q", got, "hello")
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := r.templateContent("missing.html")
		if err == nil {
			t.Error("expected error for missing file")
		}
	})
}

// --- loadTemplate ---

func TestLoadTemplate(t *testing.T) {
	base := `{{define "base"}}BASE{{block "content" .}}{{end}}END{{end}}`
	single := `{{define "content"}}SINGLE{{end}}`
	list := `{{define "content"}}LIST{{end}}`
	blogList := `{{define "content"}}BLOG-LIST{{end}}`

	t.Run("specific layout found", func(t *testing.T) {
		dir := layouts(t, map[string]string{
			"base.html":      base,
			"blog-list.html": blogList,
		})
		r := &Renderer{layoutsDir: dir}
		tmpl, err := r.loadTemplate("blog-list")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tmpl == nil {
			t.Fatal("expected non-nil template")
		}
	})

	t.Run("list layout falls back to list.html", func(t *testing.T) {
		dir := layouts(t, map[string]string{
			"base.html": base,
			"list.html": list,
		})
		r := &Renderer{layoutsDir: dir}
		tmpl, err := r.loadTemplate("blog-list")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tmpl == nil {
			t.Fatal("expected non-nil template")
		}
	})

	t.Run("non-list layout falls back to single.html", func(t *testing.T) {
		dir := layouts(t, map[string]string{
			"base.html":   base,
			"single.html": single,
		})
		r := &Renderer{layoutsDir: dir}
		tmpl, err := r.loadTemplate("about")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tmpl == nil {
			t.Fatal("expected non-nil template")
		}
	})

	t.Run("base.html missing returns error", func(t *testing.T) {
		dir := layouts(t, map[string]string{
			"single.html": single,
		})
		r := &Renderer{layoutsDir: dir}
		_, err := r.loadTemplate("single")
		if err == nil {
			t.Error("expected error when base.html is missing")
		}
	})

	t.Run("fallback list.html missing returns error", func(t *testing.T) {
		dir := layouts(t, map[string]string{
			"base.html": base,
		})
		r := &Renderer{layoutsDir: dir}
		_, err := r.loadTemplate("blog-list")
		if err == nil {
			t.Error("expected error when list.html fallback is also missing")
		}
	})

	t.Run("fallback single.html missing returns error", func(t *testing.T) {
		dir := layouts(t, map[string]string{
			"base.html": base,
		})
		r := &Renderer{layoutsDir: dir}
		_, err := r.loadTemplate("about")
		if err == nil {
			t.Error("expected error when single.html fallback is also missing")
		}
	})

	t.Run("invalid base.html returns error", func(t *testing.T) {
		dir := layouts(t, map[string]string{
			"base.html":   `{{define "base"}}{{unclosed`,
			"single.html": single,
		})
		r := &Renderer{layoutsDir: dir}
		_, err := r.loadTemplate("single")
		if err == nil {
			t.Error("expected error for invalid base template")
		}
	})

	t.Run("invalid layout returns error", func(t *testing.T) {
		dir := layouts(t, map[string]string{
			"base.html":   base,
			"single.html": `{{define "content"}}{{unclosed`,
		})
		r := &Renderer{layoutsDir: dir}
		_, err := r.loadTemplate("single")
		if err == nil {
			t.Error("expected error for invalid layout template")
		}
	})
}

// --- Render ---

func TestRender(t *testing.T) {
	base := `{{define "base"}}<!doctype html><title>{{.Page.Title}}</title>{{block "content" .}}{{end}}{{end}}`
	single := `{{define "content"}}<main>{{.Page.Content}}</main>{{end}}`
	list := `{{define "content"}}<ul>{{range .Page.Pages}}<li>{{.Title}}</li>{{end}}</ul>{{end}}`

	t.Run("renders single pages", func(t *testing.T) {
		ldir := layouts(t, map[string]string{
			"base.html":   base,
			"single.html": single,
		})
		odir := t.TempDir()
		r := New(ldir, odir, false)

		site := content.Site{
			Config: config.Config{Title: "Test"},
			Pages: []content.Page{
				{Title: "About", Slug: "about", URL: "/about/", Layout: "about"},
			},
		}
		if err := r.Render(site); err != nil {
			t.Fatalf("Render: %v", err)
		}
		assertFile(t, filepath.Join(odir, "about", "index.html"), "About")
	})

	t.Run("partials are available to layouts", func(t *testing.T) {
		ldir := layouts(t, map[string]string{
			"base.html":   `{{define "base"}}<head>{{template "head" .}}</head>{{block "content" .}}{{end}}{{end}}`,
			"single.html": `{{define "content"}}<main>{{template "icons/logo" .}}{{.Page.Content}}</main>{{end}}`,
		})
		if err := os.MkdirAll(filepath.Join(ldir, "partials", "icons"), 0755); err != nil {
			t.Fatal(err)
		}
		mustWrite(t, filepath.Join(ldir, "partials", "head.html"), `<title>{{.Site.Config.Title}}</title>`)
		mustWrite(t, filepath.Join(ldir, "partials", "icons", "logo.html"), `<svg id="logo"></svg>`)

		odir := t.TempDir()
		r := New(ldir, odir, false)
		site := content.Site{
			Config: config.Config{Title: "Partial Site"},
			Pages:  []content.Page{{Title: "Home", Slug: "", URL: "/", Layout: "single"}},
		}
		if err := r.Render(site); err != nil {
			t.Fatalf("Render: %v", err)
		}
		out := filepath.Join(odir, "index.html")
		assertFile(t, out, "<title>Partial Site</title>") // top-level partial, with data
		assertFile(t, out, `<svg id="logo"></svg>`)       // nested partial by path name
	})

	t.Run("missing partials directory is not an error", func(t *testing.T) {
		ldir := layouts(t, map[string]string{
			"base.html":   base,
			"single.html": single,
		})
		odir := t.TempDir()
		r := New(ldir, odir, false)
		site := content.Site{Pages: []content.Page{{Title: "X", Slug: "x", URL: "/x/", Layout: "single"}}}
		if err := r.Render(site); err != nil {
			t.Fatalf("Render with no partials dir should succeed: %v", err)
		}
	})

	t.Run("generates tag term pages and a tags index", func(t *testing.T) {
		ldir := layouts(t, map[string]string{
			"base.html":   base,
			"single.html": single,
			"list.html":   list,
			"tags.html":   `{{define "content"}}<ul>{{range .Page.Pages}}<li>{{.Title}}:{{len .Pages}}</li>{{end}}</ul>{{end}}`,
		})
		odir := t.TempDir()
		r := New(ldir, odir, false)
		site := content.Site{
			Pages: []content.Page{
				{Title: "P1", Slug: "blog/p1", URL: "/blog/p1/", Section: "blog", Layout: "single", Tags: []string{"go", "web dev"}},
				{Title: "P2", Slug: "blog/p2", URL: "/blog/p2/", Section: "blog", Layout: "single", Tags: []string{"go"}},
			},
		}
		if err := r.Render(site); err != nil {
			t.Fatalf("Render: %v", err)
		}
		// term page for "go" lists both posts
		goPage := filepath.Join(odir, "tags", "go", "index.html")
		assertFile(t, goPage, "P1")
		assertFile(t, goPage, "P2")
		// "web dev" slugified to "web-dev"
		if _, err := os.Stat(filepath.Join(odir, "tags", "web-dev", "index.html")); err != nil {
			t.Error("expected /tags/web-dev/ term page")
		}
		// index lists terms with counts (go has 2)
		assertFile(t, filepath.Join(odir, "tags", "index.html"), "go:2")
	})

	t.Run("no tags means no tags directory", func(t *testing.T) {
		ldir := layouts(t, map[string]string{"base.html": base, "single.html": single})
		odir := t.TempDir()
		r := New(ldir, odir, false)
		site := content.Site{Pages: []content.Page{{Title: "X", Slug: "x", URL: "/x/", Layout: "single"}}}
		if err := r.Render(site); err != nil {
			t.Fatalf("Render: %v", err)
		}
		if _, err := os.Stat(filepath.Join(odir, "tags")); err == nil {
			t.Error("tags/ should not exist when no page has tags")
		}
	})

	t.Run("paginates a section list across numbered pages", func(t *testing.T) {
		paged := `{{define "content"}}<ul>{{range .Page.Pages}}<li>{{.Title}}</li>{{end}}</ul>` +
			`{{with .Page.Paginator}}<p>page {{.PageNumber}}/{{.TotalPages}} prev={{.PrevURL}} next={{.NextURL}}</p>{{end}}{{end}}`
		ldir := layouts(t, map[string]string{
			"base.html":   base,
			"single.html": single,
			"list.html":   paged,
		})
		odir := t.TempDir()
		r := New(ldir, odir, false)

		var posts []content.Page
		for i := 1; i <= 5; i++ {
			n := strconv.Itoa(i)
			posts = append(posts, content.Page{Title: "Post" + n, Slug: "blog/p" + n, URL: "/blog/p" + n + "/", Section: "blog", Layout: "single"})
		}
		site := content.Site{
			Config: config.Config{Paginate: 2}, // 5 posts, 2 per page → 3 pages
			Pages:  append([]content.Page{{Title: "Blog", Slug: "blog", URL: "/blog/", Section: "blog", IsSection: true, Layout: "blog-list"}}, posts...),
		}
		if err := r.Render(site); err != nil {
			t.Fatalf("Render: %v", err)
		}

		// page 1 at base URL, with 2 items and a next link to page 2
		p1 := filepath.Join(odir, "blog", "index.html")
		assertFile(t, p1, "page 1/3")
		assertFile(t, p1, "next=/blog/page/2/")
		assertFile(t, p1, "prev=") // empty prev on page 1
		// page 2 at /blog/page/2/
		assertFile(t, filepath.Join(odir, "blog", "page", "2", "index.html"), "page 2/3")
		// page 3 is the last, no next
		p3 := filepath.Join(odir, "blog", "page", "3", "index.html")
		assertFile(t, p3, "prev=/blog/page/2/")
		assertFile(t, p3, "next=</p>") // empty next on last page
	})

	t.Run("no pagination when items fit on one page", func(t *testing.T) {
		ldir := layouts(t, map[string]string{
			"base.html": base, "single.html": single,
			"list.html": `{{define "content"}}{{with .Page.Paginator}}HAS-PAGINATOR{{end}}<ul>{{range .Page.Pages}}<li>{{.Title}}</li>{{end}}</ul>{{end}}`,
		})
		odir := t.TempDir()
		r := New(ldir, odir, false)
		site := content.Site{
			Config: config.Config{Paginate: 10},
			Pages: []content.Page{
				{Title: "Blog", Slug: "blog", URL: "/blog/", Section: "blog", IsSection: true, Layout: "blog-list"},
				{Title: "Only", Slug: "blog/only", URL: "/blog/only/", Section: "blog", Layout: "single"},
			},
		}
		if err := r.Render(site); err != nil {
			t.Fatalf("Render: %v", err)
		}
		data, _ := os.ReadFile(filepath.Join(odir, "blog", "index.html"))
		if strings.Contains(string(data), "HAS-PAGINATOR") {
			t.Error("single-page list should not have a paginator")
		}
		if _, err := os.Stat(filepath.Join(odir, "blog", "page")); err == nil {
			t.Error("blog/page/ should not exist for a single-page list")
		}
	})

	t.Run("sets absolute Permalink from baseURL", func(t *testing.T) {
		ldir := layouts(t, map[string]string{
			"base.html":   `{{define "base"}}<link rel="canonical" href="{{.Page.Permalink}}">{{block "content" .}}{{end}}{{end}}`,
			"single.html": single,
		})
		odir := t.TempDir()
		r := New(ldir, odir, false)
		site := content.Site{
			Config: config.Config{BaseURL: "https://example.com/"},
			Pages:  []content.Page{{Title: "About", Slug: "about", URL: "/about/", Layout: "single"}},
		}
		if err := r.Render(site); err != nil {
			t.Fatalf("Render: %v", err)
		}
		assertFile(t, filepath.Join(odir, "about", "index.html"), `href="https://example.com/about/"`)
	})

	t.Run("renders 404 at root", func(t *testing.T) {
		ldir := layouts(t, map[string]string{
			"base.html":   base,
			"single.html": single,
		})
		odir := t.TempDir()
		r := New(ldir, odir, false)

		site := content.Site{
			Pages: []content.Page{
				{Title: "Not Found", Slug: "404", URL: "/404/", Layout: "single"},
			},
		}
		if err := r.Render(site); err != nil {
			t.Fatalf("Render: %v", err)
		}
		if _, err := os.Stat(filepath.Join(odir, "404.html")); err != nil {
			t.Error("expected public/404.html to exist")
		}
		if _, err := os.Stat(filepath.Join(odir, "404", "index.html")); err == nil {
			t.Error("public/404/index.html should NOT exist")
		}
	})

	t.Run("section list page populated with child pages", func(t *testing.T) {
		ldir := layouts(t, map[string]string{
			"base.html":   base,
			"list.html":   list,
			"single.html": single,
		})
		odir := t.TempDir()
		r := New(ldir, odir, false)

		site := content.Site{
			Pages: []content.Page{
				{Title: "Blog", Slug: "blog", URL: "/blog/", Section: "blog", IsSection: true, Layout: "blog-list"},
				{Title: "Post 1", Slug: "blog/post-1", URL: "/blog/post-1/", Section: "blog", Layout: "blog-single"},
			},
		}
		if err := r.Render(site); err != nil {
			t.Fatalf("Render: %v", err)
		}
		data, _ := os.ReadFile(filepath.Join(odir, "blog", "index.html"))
		if !strings.Contains(string(data), "Post 1") {
			t.Error("blog index should contain Post 1 in list")
		}
	})

	t.Run("auto-generates list page for section without _index", func(t *testing.T) {
		ldir := layouts(t, map[string]string{
			"base.html":   base,
			"list.html":   list,
			"single.html": single,
		})
		odir := t.TempDir()
		r := New(ldir, odir, false)

		site := content.Site{
			Pages: []content.Page{
				{Title: "Post A", Slug: "blog/post-a", URL: "/blog/post-a/", Section: "blog", Layout: "blog-single"},
			},
		}
		if err := r.Render(site); err != nil {
			t.Fatalf("Render: %v", err)
		}
		// auto-generated list page at /blog/
		if _, err := os.Stat(filepath.Join(odir, "blog", "index.html")); err != nil {
			t.Error("expected auto-generated blog/index.html")
		}
	})

	t.Run("serveMode injects SSE script", func(t *testing.T) {
		ldir := layouts(t, map[string]string{
			"base.html":   `{{define "base"}}{{if .ServeMode}}SSE{{end}}{{block "content" .}}{{end}}{{end}}`,
			"single.html": `{{define "content"}}ok{{end}}`,
		})
		odir := t.TempDir()
		r := New(ldir, odir, true)

		site := content.Site{
			Pages: []content.Page{
				{Title: "Home", Slug: "", URL: "/", Layout: "home"},
			},
		}
		if err := r.Render(site); err != nil {
			t.Fatalf("Render: %v", err)
		}
		data, _ := os.ReadFile(filepath.Join(odir, "index.html"))
		if !strings.Contains(string(data), "SSE") {
			t.Error("ServeMode should be true in template data")
		}
	})

	t.Run("render error on missing base layout", func(t *testing.T) {
		odir := t.TempDir()
		r := New(t.TempDir(), odir, false)

		site := content.Site{
			Pages: []content.Page{
				{Title: "Home", Slug: "", Layout: "home"},
			},
		}
		if err := r.Render(site); err == nil {
			t.Error("expected error when base.html is missing")
		}
	})

	t.Run("not template function works", func(t *testing.T) {
		ldir := layouts(t, map[string]string{
			"base.html":   `{{define "base"}}{{if not .ServeMode}}static{{end}}{{block "content" .}}{{end}}{{end}}`,
			"single.html": `{{define "content"}}ok{{end}}`,
		})
		odir := t.TempDir()
		r := New(ldir, odir, false)
		site := content.Site{
			Pages: []content.Page{
				{Title: "Home", Slug: "", Layout: "home"},
			},
		}
		if err := r.Render(site); err != nil {
			t.Fatalf("Render: %v", err)
		}
		data, _ := os.ReadFile(filepath.Join(odir, "index.html"))
		if !strings.Contains(string(data), "static") {
			t.Error("not function should return true when ServeMode=false")
		}
	})

	t.Run("template execution error propagates", func(t *testing.T) {
		ldir := layouts(t, map[string]string{
			"base.html":   `{{define "base"}}{{template "nonexistent_tmpl_xyz" .}}{{end}}`,
			"single.html": `{{define "content"}}ok{{end}}`,
		})
		odir := t.TempDir()
		r := New(ldir, odir, false)
		site := content.Site{
			Pages: []content.Page{
				{Title: "Home", Slug: "", Layout: "home"},
			},
		}
		if err := r.Render(site); err == nil {
			t.Error("expected error for undefined template reference")
		}
	})

	t.Run("Render MkdirAll error propagates", func(t *testing.T) {
		ldir := layouts(t, map[string]string{
			"base.html":   `{{define "base"}}{{block "content" .}}{{end}}{{end}}`,
			"single.html": `{{define "content"}}ok{{end}}`,
		})
		blocker := filepath.Join(t.TempDir(), "blocker")
		if err := os.WriteFile(blocker, []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
		r := New(ldir, filepath.Join(blocker, "output"), false)
		if err := r.Render(content.Site{}); err == nil {
			t.Error("expected error when outputDir cannot be created")
		}
	})

	t.Run("renderPage MkdirAll error propagates", func(t *testing.T) {
		ldir := layouts(t, map[string]string{
			"base.html":   `{{define "base"}}{{block "content" .}}{{end}}{{end}}`,
			"single.html": `{{define "content"}}ok{{end}}`,
		})
		odir := t.TempDir()
		// Place a file at "odir/about" to block os.MkdirAll("odir/about", 0755)
		if err := os.WriteFile(filepath.Join(odir, "about"), []byte("blocker"), 0644); err != nil {
			t.Fatal(err)
		}
		r := New(ldir, odir, false)
		site := content.Site{
			Pages: []content.Page{
				{Title: "About", Slug: "about", Layout: "single"},
			},
		}
		if err := r.Render(site); err == nil {
			t.Error("expected error when output subdir is blocked by a file")
		}
	})

	t.Run("auto-list render error propagates", func(t *testing.T) {
		ldir := layouts(t, map[string]string{
			"base.html":   `{{define "base"}}{{block "content" .}}{{end}}{{end}}`,
			"single.html": `{{define "content"}}post{{end}}`,
			"list.html":   `{{define "content"}}list{{end}}`,
		})
		odir := t.TempDir()
		// Pre-create blog/index.html as a directory so WriteFile fails for the auto-list,
		// but blog/post-a still renders successfully to blog/post-a/index.html.
		os.MkdirAll(filepath.Join(odir, "blog", "index.html"), 0755)
		r := New(ldir, odir, false)
		site := content.Site{
			Pages: []content.Page{
				{Title: "Post A", Slug: "blog/post-a", Section: "blog", Layout: "blog-single"},
			},
		}
		if err := r.Render(site); err == nil {
			t.Error("expected error when auto-list page cannot be written")
		}
	})
}

// helpers

func layouts(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatalf("write layout %s: %v", name, err)
		}
	}
	return dir
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func assertFile(t *testing.T, path, contains string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected file %s to exist: %v", path, err)
	}
	if !strings.Contains(string(data), contains) {
		t.Errorf("file %s does not contain %q\ngot: %s", path, contains, data)
	}
}
