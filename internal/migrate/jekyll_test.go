package migrate

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// --- hasLiquid ---

func TestHasLiquid(t *testing.T) {
	tests := []struct {
		body string
		want bool
	}{
		{"plain text", false},
		{"{% if page.title %}Title{% endif %}", true},
		{"{{ page.url }}", true},
		{"no liquid here", false},
	}
	for _, tt := range tests {
		got := hasLiquid([]byte(tt.body))
		if got != tt.want {
			t.Errorf("hasLiquid(%q) = %v, want %v", tt.body, got, tt.want)
		}
	}
}

// --- extractPostSlug ---

func TestExtractPostSlug(t *testing.T) {
	tests := []struct {
		base         string
		wantDate     string
		wantSlug     string
	}{
		{"2024-01-15-my-post-title", "2024-01-15", "my-post-title"},
		{"2024-12-31-another", "2024-12-31", "another"},
		{"no-date-here", "", "no-date-here"},
		{"2024-01-notadate", "", "2024-01-notadate"},
		{"short", "", "short"},
	}
	for _, tt := range tests {
		dateStr, slug := extractPostSlug(tt.base)
		if dateStr != tt.wantDate {
			t.Errorf("extractPostSlug(%q) dateStr = %q, want %q", tt.base, dateStr, tt.wantDate)
		}
		if slug != tt.wantSlug {
			t.Errorf("extractPostSlug(%q) slug = %q, want %q", tt.base, slug, tt.wantSlug)
		}
	}
}

// --- parseJekyllFrontmatter ---

func TestParseJekyllFrontmatter(t *testing.T) {
	t.Run("full frontmatter", func(t *testing.T) {
		data := "---\ntitle: My Post\ndate: 2024-03-10\ntags: [go]\ncategories: [web]\ndescription: Desc\nexcerpt: Ex\npermalink: /custom/\n---\nBody"
		fm, body, err := parseJekyllFrontmatter([]byte(data))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fm.Title != "My Post" {
			t.Errorf("Title = %q", fm.Title)
		}
		if len(fm.Tags) != 1 || fm.Tags[0] != "go" {
			t.Errorf("Tags = %v", fm.Tags)
		}
		if len(fm.Categories) != 1 {
			t.Errorf("Categories = %v", fm.Categories)
		}
		if fm.Description != "Desc" {
			t.Errorf("Description = %q", fm.Description)
		}
		if fm.Excerpt != "Ex" {
			t.Errorf("Excerpt = %q", fm.Excerpt)
		}
		if fm.Permalink != "/custom/" {
			t.Errorf("Permalink = %q", fm.Permalink)
		}
		if string(body) != "Body" {
			t.Errorf("body = %q", body)
		}
	})

	t.Run("published false sets draft", func(t *testing.T) {
		data := "---\ntitle: X\npublished: false\n---\nBody"
		fm, _, err := parseJekyllFrontmatter([]byte(data))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fm.Published == nil || *fm.Published != false {
			t.Error("Published should be false")
		}
	})

	t.Run("no frontmatter", func(t *testing.T) {
		data := "Just content"
		fm, body, err := parseJekyllFrontmatter([]byte(data))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fm.Title != "" {
			t.Errorf("expected empty title, got %q", fm.Title)
		}
		if string(body) != "Just content" {
			t.Errorf("body = %q", body)
		}
	})

	t.Run("unclosed frontmatter returns data as body", func(t *testing.T) {
		data := "---\ntitle: Unclosed"
		_, body, _ := parseJekyllFrontmatter([]byte(data))
		if string(body) != data {
			t.Errorf("body = %q, want original data", body)
		}
	})

	t.Run("trailing newline after closing delimiter consumed", func(t *testing.T) {
		data := "---\ntitle: Hi\n---\n\nParagraph"
		fm, body, err := parseJekyllFrontmatter([]byte(data))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fm.Title != "Hi" {
			t.Errorf("Title = %q", fm.Title)
		}
		if string(body) != "\nParagraph" {
			t.Errorf("body = %q", body)
		}
	})
}

// --- loadJekyllConfig ---

func TestLoadJekyllConfig(t *testing.T) {
	t.Run("_config.yml with string author", func(t *testing.T) {
		dir := t.TempDir()
		writeF(t, filepath.Join(dir, "_config.yml"), `title: Jekyll Site
url: https://example.com
description: A blog
author: Alice
github_username: alice
twitter_username: alicetw
`)
		cfg, err := loadJekyllConfig(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Title != "Jekyll Site" {
			t.Errorf("Title = %q", cfg.Title)
		}
		if cfg.BaseURL != "https://example.com" {
			t.Errorf("BaseURL = %q", cfg.BaseURL)
		}
		if cfg.Description != "A blog" {
			t.Errorf("Description = %q", cfg.Description)
		}
		if cfg.Author.Name != "Alice" {
			t.Errorf("Author.Name = %q", cfg.Author.Name)
		}
		if cfg.Author.GitHub != "alice" {
			t.Errorf("Author.GitHub = %q", cfg.Author.GitHub)
		}
		if cfg.Author.Twitter != "alicetw" {
			t.Errorf("Author.Twitter = %q", cfg.Author.Twitter)
		}
	})

	t.Run("_config.yml with map author", func(t *testing.T) {
		dir := t.TempDir()
		writeF(t, filepath.Join(dir, "_config.yml"), `title: Site
url: https://example.com
author:
  name: Bob
  email: bob@example.com
  github: bob
  twitter: bobtw
  linkedin: bobli
`)
		cfg, err := loadJekyllConfig(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Author.Name != "Bob" {
			t.Errorf("Author.Name = %q", cfg.Author.Name)
		}
		if cfg.Author.Email != "bob@example.com" {
			t.Errorf("Author.Email = %q", cfg.Author.Email)
		}
		if cfg.Author.LinkedIn != "bobli" {
			t.Errorf("Author.LinkedIn = %q", cfg.Author.LinkedIn)
		}
	})

	t.Run("url + baseurl combined", func(t *testing.T) {
		dir := t.TempDir()
		writeF(t, filepath.Join(dir, "_config.yml"), `title: Site
url: https://example.com
baseurl: /blog
`)
		cfg, err := loadJekyllConfig(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.BaseURL != "https://example.com/blog" {
			t.Errorf("BaseURL = %q", cfg.BaseURL)
		}
	})

	t.Run("baseurl is / ignored", func(t *testing.T) {
		dir := t.TempDir()
		writeF(t, filepath.Join(dir, "_config.yml"), `title: Site
url: https://example.com
baseurl: /
`)
		cfg, err := loadJekyllConfig(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.BaseURL != "https://example.com" {
			t.Errorf("BaseURL = %q, want https://example.com", cfg.BaseURL)
		}
	})

	t.Run("_config.yaml also accepted", func(t *testing.T) {
		dir := t.TempDir()
		writeF(t, filepath.Join(dir, "_config.yaml"), "title: YAML Site")
		cfg, err := loadJekyllConfig(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Title != "YAML Site" {
			t.Errorf("Title = %q", cfg.Title)
		}
	})

	t.Run("invalid YAML returns error", func(t *testing.T) {
		dir := t.TempDir()
		writeF(t, filepath.Join(dir, "_config.yml"), `title: [invalid yaml`)
		_, err := loadJekyllConfig(dir)
		if err == nil {
			t.Error("expected error for invalid YAML in _config.yml")
		}
	})

	t.Run("no config returns error", func(t *testing.T) {
		_, err := loadJekyllConfig(t.TempDir())
		if err == nil {
			t.Error("expected error when no config found")
		}
	})
}

// --- Jekyll (integration) ---

func TestJekyll(t *testing.T) {
	t.Run("full migration", func(t *testing.T) {
		src := t.TempDir()
		writeF(t, filepath.Join(src, "_config.yml"), `title: Jekyll Blog
url: https://example.com
`)
		// posts
		writeF(t, filepath.Join(src, "_posts", "2024-01-15-hello-world.md"),
			"---\ntitle: Hello World\ntags: [go]\n---\nPost body")
		writeF(t, filepath.Join(src, "_posts", "2024-03-20-second-post.md"),
			"---\ntitle: Second Post\ncategories: [web]\ndescription: Desc\n---\nBody")
		// pages dir
		writeF(t, filepath.Join(src, "_pages", "about.md"),
			"---\ntitle: About\n---\nAbout page")
		// root pages
		writeF(t, filepath.Join(src, "contact.md"),
			"---\ntitle: Contact\n---\nContact page")
		// assets
		writeF(t, filepath.Join(src, "assets", "style.css"), "body{}")

		dst := filepath.Join(t.TempDir(), "out")
		result, err := Jekyll(src, dst)
		if err != nil {
			t.Fatalf("Jekyll: %v", err)
		}

		// 2 posts + about + contact = 4 pages
		if result.Pages != 4 {
			t.Errorf("Pages = %d, want 4", result.Pages)
		}
		if result.Static != 1 {
			t.Errorf("Static = %d, want 1", result.Static)
		}

		// posts go to content/blog/
		assertExistsJ(t, filepath.Join(dst, "content", "blog", "hello-world.md"))
		assertExistsJ(t, filepath.Join(dst, "content", "blog", "second-post.md"))

		// slug from filename, date from filename
		data, _ := os.ReadFile(filepath.Join(dst, "content", "blog", "hello-world.md"))
		if !strings.Contains(string(data), "2024-01-15") {
			t.Errorf("expected date from filename, got:\n%s", data)
		}

		// categories mapped to tags
		data2, _ := os.ReadFile(filepath.Join(dst, "content", "blog", "second-post.md"))
		if !strings.Contains(string(data2), "web") {
			t.Errorf("categories should be mapped to tags, got:\n%s", data2)
		}

		// _pages migrated
		assertExistsJ(t, filepath.Join(dst, "content", "about.md"))

		// root pages migrated
		assertExistsJ(t, filepath.Join(dst, "content", "contact.md"))

		// assets copied
		assertExistsJ(t, filepath.Join(dst, "static", "assets", "style.css"))
	})

	t.Run("warns about liquid tags", func(t *testing.T) {
		src := t.TempDir()
		writeF(t, filepath.Join(src, "_config.yml"), "title: X")
		writeF(t, filepath.Join(src, "_posts", "2024-01-01-liquid.md"),
			"---\ntitle: Liquid\n---\n{% if true %}yes{% endif %}")

		dst := filepath.Join(t.TempDir(), "out")
		result, err := Jekyll(src, dst)
		if err != nil {
			t.Fatalf("Jekyll: %v", err)
		}
		if len(result.Warnings) == 0 {
			t.Error("expected warning for liquid tags")
		}
	})

	t.Run("published false becomes draft", func(t *testing.T) {
		src := t.TempDir()
		writeF(t, filepath.Join(src, "_config.yml"), "title: X")
		writeF(t, filepath.Join(src, "_posts", "2024-01-01-unpublished.md"),
			"---\ntitle: Unpublished\npublished: false\n---\nBody")

		dst := filepath.Join(t.TempDir(), "out")
		_, err := Jekyll(src, dst)
		if err != nil {
			t.Fatalf("Jekyll: %v", err)
		}
		data, _ := os.ReadFile(filepath.Join(dst, "content", "blog", "unpublished.md"))
		if !strings.Contains(string(data), "draft: true") {
			t.Errorf("published: false should produce draft: true, got:\n%s", data)
		}
	})

	t.Run("permalink in post produces warning", func(t *testing.T) {
		src := t.TempDir()
		writeF(t, filepath.Join(src, "_config.yml"), "title: X")
		writeF(t, filepath.Join(src, "_posts", "2024-01-01-perm.md"),
			"---\ntitle: Perm\npermalink: /custom/url/\n---\nBody")

		dst := filepath.Join(t.TempDir(), "out")
		result, err := Jekyll(src, dst)
		if err != nil {
			t.Fatalf("Jekyll: %v", err)
		}
		found := false
		for _, w := range result.Warnings {
			if strings.Contains(w, "permalink") {
				found = true
			}
		}
		if !found {
			t.Error("expected permalink warning")
		}
	})

	t.Run("skips README and Gemfile at root", func(t *testing.T) {
		src := t.TempDir()
		writeF(t, filepath.Join(src, "_config.yml"), "title: X")
		writeF(t, filepath.Join(src, "README.md"), "# Readme")
		writeF(t, filepath.Join(src, "Gemfile"), "source 'https://rubygems.org'")
		writeF(t, filepath.Join(src, "about.md"), "---\ntitle: About\n---\nBody")

		dst := filepath.Join(t.TempDir(), "out")
		result, err := Jekyll(src, dst)
		if err != nil {
			t.Fatalf("Jekyll: %v", err)
		}
		if result.Pages != 1 {
			t.Errorf("Pages = %d, want 1 (README and Gemfile skipped)", result.Pages)
		}
		if _, err := os.Stat(filepath.Join(dst, "content", "README.md")); err == nil {
			t.Error("README.md should not be migrated")
		}
	})

	t.Run("liquid tags in root pages produce warning", func(t *testing.T) {
		src := t.TempDir()
		writeF(t, filepath.Join(src, "_config.yml"), "title: X")
		writeF(t, filepath.Join(src, "index.md"),
			"---\ntitle: Home\n---\n{{ site.title }}")

		dst := filepath.Join(t.TempDir(), "out")
		result, err := Jekyll(src, dst)
		if err != nil {
			t.Fatalf("Jekyll: %v", err)
		}
		if len(result.Warnings) == 0 {
			t.Error("expected warning for liquid tags in root page")
		}
	})

	t.Run("liquid tags in _pages produce warning", func(t *testing.T) {
		src := t.TempDir()
		writeF(t, filepath.Join(src, "_config.yml"), "title: X")
		writeF(t, filepath.Join(src, "_pages", "about.md"),
			"---\ntitle: About\n---\n{% include nav.html %}")

		dst := filepath.Join(t.TempDir(), "out")
		result, err := Jekyll(src, dst)
		if err != nil {
			t.Fatalf("Jekyll: %v", err)
		}
		if len(result.Warnings) == 0 {
			t.Error("expected warning for liquid tags in _pages")
		}
	})

	t.Run("excerpt mapped to description", func(t *testing.T) {
		src := t.TempDir()
		writeF(t, filepath.Join(src, "_config.yml"), "title: X")
		writeF(t, filepath.Join(src, "_posts", "2024-01-01-excerpt.md"),
			"---\ntitle: Post\nexcerpt: Short summary\n---\nBody")

		dst := filepath.Join(t.TempDir(), "out")
		_, err := Jekyll(src, dst)
		if err != nil {
			t.Fatalf("Jekyll: %v", err)
		}
		data, _ := os.ReadFile(filepath.Join(dst, "content", "blog", "excerpt.md"))
		if !strings.Contains(string(data), "description: Short summary") {
			t.Errorf("excerpt should map to description, got:\n%s", data)
		}
	})

	t.Run("post without date in filename uses frontmatter date", func(t *testing.T) {
		src := t.TempDir()
		writeF(t, filepath.Join(src, "_config.yml"), "title: X")
		writeF(t, filepath.Join(src, "_posts", "nodateprefix.md"),
			"---\ntitle: No Date\ndate: 2023-06-01\n---\nBody")

		dst := filepath.Join(t.TempDir(), "out")
		_, err := Jekyll(src, dst)
		if err != nil {
			t.Fatalf("Jekyll: %v", err)
		}
		data, _ := os.ReadFile(filepath.Join(dst, "content", "blog", "nodateprefix.md"))
		if !strings.Contains(string(data), "2023-06-01") {
			t.Errorf("should use frontmatter date, got:\n%s", data)
		}
	})

	t.Run("invalid frontmatter in post produces warning and continues", func(t *testing.T) {
		src := t.TempDir()
		writeF(t, filepath.Join(src, "_config.yml"), "title: X")
		writeF(t, filepath.Join(src, "_posts", "2024-01-01-bad-fm.md"),
			"---\ntitle: [invalid yaml\n---\nBody")

		dst := filepath.Join(t.TempDir(), "out")
		result, err := Jekyll(src, dst)
		if err != nil {
			t.Fatalf("Jekyll: %v", err)
		}
		found := false
		for _, w := range result.Warnings {
			if strings.Contains(w, "frontmatter parse error") {
				found = true
			}
		}
		if !found {
			t.Errorf("expected frontmatter parse warning, got warnings: %v", result.Warnings)
		}
	})

	t.Run("MkdirAll error returns error", func(t *testing.T) {
		src := t.TempDir()
		writeF(t, filepath.Join(src, "_config.yml"), "title: X")
		blocker := filepath.Join(t.TempDir(), "blocker")
		os.WriteFile(blocker, []byte("x"), 0644)
		_, err := Jekyll(src, filepath.Join(blocker, "dst"))
		if err == nil {
			t.Error("expected error when dst cannot be created")
		}
	})

	t.Run("unreadable root page returns error", func(t *testing.T) {
		if runtime.GOOS == "windows" || os.Getuid() == 0 {
			t.Skip("chmod-based test not applicable on Windows or as root")
		}
		src := t.TempDir()
		writeF(t, filepath.Join(src, "_config.yml"), "title: X")
		f := filepath.Join(src, "about.md")
		writeF(t, f, "---\ntitle: About\n---\nBody")
		os.Chmod(f, 0000)
		t.Cleanup(func() { os.Chmod(f, 0644) })

		dst := filepath.Join(t.TempDir(), "out")
		_, err := Jekyll(src, dst)
		if err == nil {
			t.Error("expected error for unreadable root page")
		}
	})

	t.Run("unreadable post returns error", func(t *testing.T) {
		if runtime.GOOS == "windows" || os.Getuid() == 0 {
			t.Skip("chmod-based test not applicable on Windows or as root")
		}
		src := t.TempDir()
		writeF(t, filepath.Join(src, "_config.yml"), "title: X")
		f := filepath.Join(src, "_posts", "2024-01-01-post.md")
		writeF(t, f, "---\ntitle: Post\n---\nBody")
		os.Chmod(f, 0000)
		t.Cleanup(func() { os.Chmod(f, 0644) })

		dst := filepath.Join(t.TempDir(), "out")
		_, err := Jekyll(src, dst)
		if err == nil {
			t.Error("expected error for unreadable post")
		}
	})

	t.Run("unreadable page in _pages returns error", func(t *testing.T) {
		if runtime.GOOS == "windows" || os.Getuid() == 0 {
			t.Skip("chmod-based test not applicable on Windows or as root")
		}
		src := t.TempDir()
		writeF(t, filepath.Join(src, "_config.yml"), "title: X")
		f := filepath.Join(src, "_pages", "about.md")
		writeF(t, f, "---\ntitle: About\n---\nBody")
		os.Chmod(f, 0000)
		t.Cleanup(func() { os.Chmod(f, 0644) })

		dst := filepath.Join(t.TempDir(), "out")
		_, err := Jekyll(src, dst)
		if err == nil {
			t.Error("expected error for unreadable page")
		}
	})

	t.Run("destination already exists returns error", func(t *testing.T) {
		src := t.TempDir()
		dst := t.TempDir() // already exists
		_, err := Jekyll(src, dst)
		if err == nil {
			t.Error("expected error when destination already exists")
		}
	})

	t.Run("config parse error produces warning", func(t *testing.T) {
		src := t.TempDir()
		// No _config.yml — should warn and continue
		dst := filepath.Join(t.TempDir(), "out")
		result, err := Jekyll(src, dst)
		if err != nil {
			t.Fatalf("Jekyll: %v", err)
		}
		if len(result.Warnings) == 0 {
			t.Error("expected warning when config not found")
		}
	})

	t.Run("multiple asset dirs copied", func(t *testing.T) {
		src := t.TempDir()
		writeF(t, filepath.Join(src, "_config.yml"), "title: X")
		writeF(t, filepath.Join(src, "images", "photo.jpg"), "img")
		writeF(t, filepath.Join(src, "files", "doc.pdf"), "pdf")

		dst := filepath.Join(t.TempDir(), "out")
		result, err := Jekyll(src, dst)
		if err != nil {
			t.Fatalf("Jekyll: %v", err)
		}
		if result.Static != 2 {
			t.Errorf("Static = %d, want 2", result.Static)
		}
		assertExistsJ(t, filepath.Join(dst, "static", "images", "photo.jpg"))
		assertExistsJ(t, filepath.Join(dst, "static", "files", "doc.pdf"))
	})
}

func assertExistsJ(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected %s to exist: %v", path, err)
	}
}
