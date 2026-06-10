package build

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCopyStatic(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("chmod-based tests not applicable on Windows or as root")
	}

	t.Run("non-IsNotExist error propagates", func(t *testing.T) {
		src := t.TempDir()
		dst := t.TempDir()
		sub := filepath.Join(src, "restricted")
		os.MkdirAll(sub, 0755)
		os.WriteFile(filepath.Join(sub, "file.txt"), []byte("x"), 0644)
		os.Chmod(sub, 0000)
		t.Cleanup(func() { os.Chmod(sub, 0755) })

		if err := copyStatic(src, dst); err == nil {
			t.Error("expected error walking restricted directory")
		}
	})

	t.Run("ReadFile error propagates", func(t *testing.T) {
		src := t.TempDir()
		dst := t.TempDir()
		f := filepath.Join(src, "unreadable.txt")
		os.WriteFile(f, []byte("x"), 0644)
		os.Chmod(f, 0000)
		t.Cleanup(func() { os.Chmod(f, 0644) })

		if err := copyStatic(src, dst); err == nil {
			t.Error("expected error reading unreadable file")
		}
	})
}

func TestRun(t *testing.T) {
	t.Run("builds a complete site", func(t *testing.T) {
		dir := siteDir(t, map[string]string{
			"nemi.toml": `title = "Test Site"
baseURL = "http://localhost:3000"
description = "A test"
`,
			"layouts/base.html":   `{{define "base"}}<!doctype html><title>{{.Page.Title}}</title>{{block "content" .}}{{end}}{{end}}`,
			"layouts/single.html": `{{define "content"}}<main>{{.Page.Content}}</main>{{end}}`,
			"layouts/list.html":   `{{define "content"}}<ul>{{range .Page.Pages}}<li>{{.Title}}</li>{{end}}</ul>{{end}}`,
			"content/index.md":    "---\ntitle: Home\n---\nWelcome",
			"content/about.md":    "---\ntitle: About\n---\nAbout me",
			"content/blog/post.md": "---\ntitle: My Post\ndate: 2024-01-01\n---\nPost body",
			"static/style.css":    "body { margin: 0; }",
		})

		if _, err := Run(false, false); err != nil {
			t.Fatalf("Run: %v", err)
		}

		assertExists(t, filepath.Join(dir, "public", "index.html"))
		assertExists(t, filepath.Join(dir, "public", "about", "index.html"))
		assertExists(t, filepath.Join(dir, "public", "blog", "post", "index.html"))
		assertExists(t, filepath.Join(dir, "public", "blog", "index.html"))
		assertExists(t, filepath.Join(dir, "public", "style.css"))
		assertExists(t, filepath.Join(dir, "public", "sitemap.xml"))
		assertExists(t, filepath.Join(dir, "public", "feed.xml"))

		robots, err := os.ReadFile(filepath.Join(dir, "public", "robots.txt"))
		if err != nil {
			t.Fatalf("read robots.txt: %v", err)
		}
		if !strings.Contains(string(robots), "User-agent: *") {
			t.Errorf("robots.txt missing user-agent:\n%s", robots)
		}
		if !strings.Contains(string(robots), "Sitemap: http://localhost:3000/sitemap.xml") {
			t.Errorf("robots.txt missing sitemap line:\n%s", robots)
		}
	})

	t.Run("includes drafts when flag set", func(t *testing.T) {
		dir := siteDir(t, map[string]string{
			"nemi.toml": `title = "Test"
baseURL = "http://localhost:3000"
`,
			"layouts/base.html":   `{{define "base"}}{{block "content" .}}{{end}}{{end}}`,
			"layouts/single.html": `{{define "content"}}{{.Page.Title}}{{end}}`,
			"content/pub.md":      "---\ntitle: Published\n---\nBody",
			"content/draft.md":    "---\ntitle: Draft Post\ndraft: true\n---\nBody",
		})

		if _, err := Run(false, true); err != nil {
			t.Fatalf("Run: %v", err)
		}

		assertExists(t, filepath.Join(dir, "public", "pub", "index.html"))
		assertExists(t, filepath.Join(dir, "public", "draft", "index.html"))

		data, _ := os.ReadFile(filepath.Join(dir, "public", "draft", "index.html"))
		if !strings.Contains(string(data), "Draft Post") {
			t.Error("draft page should render when --drafts is set")
		}
	})

	t.Run("excludes drafts by default", func(t *testing.T) {
		siteDir(t, map[string]string{
			"nemi.toml": `title = "Test"
baseURL = "http://localhost:3000"
`,
			"layouts/base.html":   `{{define "base"}}{{block "content" .}}{{end}}{{end}}`,
			"layouts/single.html": `{{define "content"}}ok{{end}}`,
			"content/pub.md":      "---\ntitle: Published\n---\nBody",
			"content/draft.md":    "---\ntitle: Draft Post\ndraft: true\n---\nBody",
		})

		if _, err := Run(false, false); err != nil {
			t.Fatalf("Run: %v", err)
		}

		if _, err := os.Stat("public/draft/index.html"); err == nil {
			t.Error("draft page should not exist without --drafts")
		}
	})

	t.Run("minifies html and css for production but not serve", func(t *testing.T) {
		files := map[string]string{
			"nemi.toml": `title = "T"
baseURL = "http://x"
`,
			"layouts/base.html":   "{{define \"base\"}}<!doctype html>\n<html>\n  <body>\n    {{block \"content\" .}}{{end}}\n  </body>\n</html>{{end}}",
			"layouts/single.html": `{{define "content"}}<p>hi</p>{{end}}`,
			"content/index.md":    "---\ntitle: Home\n---\nBody",
			"static/style.css":    "body {\n    color: red;\n    margin: 0;\n}\n",
		}

		siteDir(t, files)
		if _, err := Run(false, false); err != nil { // production
			t.Fatalf("Run: %v", err)
		}
		html, _ := os.ReadFile("public/index.html")
		if strings.Contains(string(html), "  ") {
			t.Errorf("production html should be minified (no indentation):\n%s", html)
		}
		css, _ := os.ReadFile("public/style.css")
		if !strings.Contains(string(css), "color:red") {
			t.Errorf("production css should be minified:\n%s", css)
		}

		siteDir(t, files)
		if _, err := Run(true, false); err != nil { // serve mode
			t.Fatalf("Run serve: %v", err)
		}
		cssServe, _ := os.ReadFile("public/style.css")
		if !strings.Contains(string(cssServe), "color: red") {
			t.Errorf("serve css should NOT be minified:\n%s", cssServe)
		}
	})

	t.Run("missing nemi.toml returns error", func(t *testing.T) {
		siteDir(t, map[string]string{})
		_, err := Run(false, false)
		if err == nil {
			t.Error("expected error when nemi.toml is missing")
		}
	})

	t.Run("missing layout returns error", func(t *testing.T) {
		siteDir(t, map[string]string{
			"nemi.toml":        `title = "Test"`,
			"content/index.md": "---\ntitle: Home\n---\nBody",
		})
		_, err := Run(false, false)
		if err == nil {
			t.Error("expected error when layouts dir is missing")
		}
	})

	t.Run("invalid content frontmatter returns error", func(t *testing.T) {
		siteDir(t, map[string]string{
			"nemi.toml": `title = "Test"
baseURL = "http://localhost:3000"
`,
			"layouts/base.html":   `{{define "base"}}{{block "content" .}}{{end}}{{end}}`,
			"layouts/single.html": `{{define "content"}}ok{{end}}`,
			"content/bad.md":      "---\ntitle: [invalid yaml\n---\nBody",
		})
		_, err := Run(false, false)
		if err == nil {
			t.Error("expected error for invalid content frontmatter")
		}
	})

	t.Run("static dir absent is not an error", func(t *testing.T) {
		siteDir(t, map[string]string{
			"nemi.toml": `title = "Test"
baseURL = "http://localhost:3000"
`,
			"layouts/base.html":   `{{define "base"}}{{block "content" .}}{{end}}{{end}}`,
			"layouts/single.html": `{{define "content"}}ok{{end}}`,
			"content/index.md":    "---\ntitle: Home\n---\nBody",
		})
		if _, err := Run(false, false); err != nil {
			t.Fatalf("Run without static dir: %v", err)
		}
	})
}

// siteDir creates a temp directory, writes the given files into it, changes
// the working directory to it, and registers cleanup. Returns the dir path.
func siteDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
	return dir
}

func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected %s to exist: %v", path, err)
	}
}
