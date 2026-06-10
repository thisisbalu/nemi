package migrate

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// --- fmFromTime ---

func TestFmFromTime(t *testing.T) {
	if got := fmFromTime(time.Time{}); got != "" {
		t.Errorf("zero time should return empty, got %q", got)
	}
	d := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	if got := fmFromTime(d); got != "2024-03-15" {
		t.Errorf("got %q, want 2024-03-15", got)
	}
}

// --- hasHugoShortcodes ---

func TestHasHugoShortcodes(t *testing.T) {
	tests := []struct {
		body string
		want bool
	}{
		{"plain text", false},
		{"{{< figure src=\"x.png\" >}}", true},
		{"{{< youtube id >}}", true},
		{"{{% mdshortcode %}}", true},
		{"{{ .Title }}", false},
	}
	for _, tt := range tests {
		got := hasHugoShortcodes([]byte(tt.body))
		if got != tt.want {
			t.Errorf("hasHugoShortcodes(%q) = %v, want %v", tt.body, got, tt.want)
		}
	}
}

// --- parseHugoFrontmatter ---

func TestParseHugoFrontmatter(t *testing.T) {
	t.Run("YAML frontmatter", func(t *testing.T) {
		data := "---\ntitle: My Post\ndate: 2024-01-15\ndraft: true\ntags: [go, web]\ndescription: A post\n---\nBody text"
		fm, body, err := parseHugoFrontmatter([]byte(data))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fm.Title != "My Post" {
			t.Errorf("Title = %q", fm.Title)
		}
		if fm.Draft != true {
			t.Error("Draft should be true")
		}
		if len(fm.Tags) != 2 {
			t.Errorf("Tags = %v", fm.Tags)
		}
		if fm.Description != "A post" {
			t.Errorf("Description = %q", fm.Description)
		}
		if string(body) != "Body text" {
			t.Errorf("body = %q", body)
		}
	})

	t.Run("TOML frontmatter", func(t *testing.T) {
		data := "+++\ntitle = \"TOML Post\"\ndraft = true\ntags = [\"go\"]\n+++\nTOML body"
		fm, body, err := parseHugoFrontmatter([]byte(data))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fm.Title != "TOML Post" {
			t.Errorf("Title = %q", fm.Title)
		}
		if !fm.Draft {
			t.Error("Draft should be true")
		}
		if string(body) != "TOML body" {
			t.Errorf("body = %q", body)
		}
	})

	t.Run("JSON frontmatter", func(t *testing.T) {
		data := "{\n\"title\": \"JSON Post\",\n\"draft\": true\n}\nJSON body"
		fm, body, err := parseHugoFrontmatter([]byte(data))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fm.Title != "JSON Post" {
			t.Errorf("Title = %q", fm.Title)
		}
		if string(body) != "JSON body" {
			t.Errorf("body = %q", body)
		}
	})

	t.Run("no frontmatter", func(t *testing.T) {
		data := "Just content"
		fm, body, err := parseHugoFrontmatter([]byte(data))
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

	t.Run("unclosed YAML returns data as body", func(t *testing.T) {
		data := "---\ntitle: Unclosed"
		_, body, _ := parseHugoFrontmatter([]byte(data))
		if string(body) != data {
			t.Errorf("body = %q, want original data", body)
		}
	})

	t.Run("unclosed TOML returns data as body", func(t *testing.T) {
		data := "+++\ntitle = \"Unclosed\""
		_, body, _ := parseHugoFrontmatter([]byte(data))
		if string(body) != data {
			t.Errorf("body = %q, want original data", body)
		}
	})

	t.Run("JSON no closing brace returns data as body", func(t *testing.T) {
		data := "{\n\"title\": \"No close\""
		_, body, _ := parseHugoFrontmatter([]byte(data))
		if string(body) != data {
			t.Errorf("body = %q, want original data", body)
		}
	})

	t.Run("YAML with trailing newline after delimiter", func(t *testing.T) {
		data := "---\ntitle: Hi\n---\n\nParagraph"
		fm, body, err := parseHugoFrontmatter([]byte(data))
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

// --- parseHugoConfig ---

func TestParseHugoConfig(t *testing.T) {
	t.Run("TOML config", func(t *testing.T) {
		toml := `title = "Hugo Site"
baseURL = "https://example.com"

[params]
description = "A site"
author = "Bob"
email = "bob@example.com"
github = "bob"
twitter = "bobtw"
linkedin = "bobli"
`
		cfg, err := parseHugoConfig([]byte(toml), ".toml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Title != "Hugo Site" {
			t.Errorf("Title = %q", cfg.Title)
		}
		if cfg.BaseURL != "https://example.com" {
			t.Errorf("BaseURL = %q", cfg.BaseURL)
		}
		if cfg.Description != "A site" {
			t.Errorf("Description = %q", cfg.Description)
		}
		if cfg.Author.Name != "Bob" {
			t.Errorf("Author.Name = %q", cfg.Author.Name)
		}
		if cfg.Author.Email != "bob@example.com" {
			t.Errorf("Author.Email = %q", cfg.Author.Email)
		}
		if cfg.Author.GitHub != "bob" {
			t.Errorf("Author.GitHub = %q", cfg.Author.GitHub)
		}
		if cfg.Author.Twitter != "bobtw" {
			t.Errorf("Author.Twitter = %q", cfg.Author.Twitter)
		}
		if cfg.Author.LinkedIn != "bobli" {
			t.Errorf("Author.LinkedIn = %q", cfg.Author.LinkedIn)
		}
	})

	t.Run("YAML config", func(t *testing.T) {
		yaml := `title: Hugo YAML
baseURL: https://yaml.example.com
description: Root description
params:
  githubUsername: yamluser
  twitterUsername: yamltw
`
		cfg, err := parseHugoConfig([]byte(yaml), ".yaml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Title != "Hugo YAML" {
			t.Errorf("Title = %q", cfg.Title)
		}
		if cfg.Description != "Root description" {
			t.Errorf("Description = %q", cfg.Description)
		}
		if cfg.Author.GitHub != "yamluser" {
			t.Errorf("Author.GitHub = %q", cfg.Author.GitHub)
		}
		if cfg.Author.Twitter != "yamltw" {
			t.Errorf("Author.Twitter = %q", cfg.Author.Twitter)
		}
	})

	t.Run("JSON config", func(t *testing.T) {
		json := `{"title":"JSON Hugo","baseUrl":"https://json.example.com"}`
		cfg, err := parseHugoConfig([]byte(json), ".json")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Title != "JSON Hugo" {
			t.Errorf("Title = %q", cfg.Title)
		}
		if cfg.BaseURL != "https://json.example.com" {
			t.Errorf("BaseURL = %q", cfg.BaseURL)
		}
	})

	t.Run("invalid TOML returns error", func(t *testing.T) {
		_, err := parseHugoConfig([]byte("title = [invalid"), ".toml")
		if err == nil {
			t.Error("expected error for invalid TOML")
		}
	})

	t.Run("invalid YAML returns error", func(t *testing.T) {
		_, err := parseHugoConfig([]byte("title: [invalid yaml"), ".yaml")
		if err == nil {
			t.Error("expected error for invalid YAML")
		}
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		_, err := parseHugoConfig([]byte("{invalid}"), ".json")
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

// --- loadHugoConfig ---

func TestLoadHugoConfig(t *testing.T) {
	t.Run("hugo.toml found", func(t *testing.T) {
		dir := t.TempDir()
		writeF(t, filepath.Join(dir, "hugo.toml"), `title = "Hugo TOML"`)
		cfg, err := loadHugoConfig(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Title != "Hugo TOML" {
			t.Errorf("Title = %q", cfg.Title)
		}
	})

	t.Run("config.yaml found", func(t *testing.T) {
		dir := t.TempDir()
		writeF(t, filepath.Join(dir, "config.yaml"), "title: YAML Config")
		cfg, err := loadHugoConfig(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Title != "YAML Config" {
			t.Errorf("Title = %q", cfg.Title)
		}
	})

	t.Run("no config file returns error", func(t *testing.T) {
		_, err := loadHugoConfig(t.TempDir())
		if err == nil {
			t.Error("expected error when no config file found")
		}
	})
}

// --- Hugo (integration) ---

func TestHugo(t *testing.T) {
	t.Run("full migration", func(t *testing.T) {
		src := t.TempDir()
		writeF(t, filepath.Join(src, "hugo.toml"), `title = "My Hugo Site"
baseURL = "https://example.com"
[params]
description = "A blog"
`)
		writeF(t, filepath.Join(src, "content", "about.md"),
			"---\ntitle: About\n---\nAbout page")
		writeF(t, filepath.Join(src, "content", "blog", "post.md"),
			"---\ntitle: Post\ndate: 2024-01-15\ntags: [go]\n---\nPost body")
		writeF(t, filepath.Join(src, "static", "style.css"), "body{}")

		dst := filepath.Join(t.TempDir(), "out")
		result, err := Hugo(src, dst)
		if err != nil {
			t.Fatalf("Hugo: %v", err)
		}
		if result.Pages != 2 {
			t.Errorf("Pages = %d, want 2", result.Pages)
		}
		if result.Static != 1 {
			t.Errorf("Static = %d, want 1", result.Static)
		}

		// nemi.toml written
		toml, _ := os.ReadFile(filepath.Join(dst, "nemi.toml"))
		if !strings.Contains(string(toml), `"My Hugo Site"`) {
			t.Error("nemi.toml should contain site title")
		}

		// content migrated
		if _, err := os.Stat(filepath.Join(dst, "content", "about.md")); err != nil {
			t.Error("about.md should be migrated")
		}

		// static copied
		if _, err := os.Stat(filepath.Join(dst, "static", "style.css")); err != nil {
			t.Error("style.css should be copied")
		}
	})

	t.Run("warns about shortcodes", func(t *testing.T) {
		src := t.TempDir()
		writeF(t, filepath.Join(src, "hugo.toml"), `title = "X"`)
		writeF(t, filepath.Join(src, "content", "post.md"),
			"---\ntitle: Post\n---\n{{< figure src=\"x.png\" >}}")

		dst := filepath.Join(t.TempDir(), "out")
		result, err := Hugo(src, dst)
		if err != nil {
			t.Fatalf("Hugo: %v", err)
		}
		if len(result.Warnings) == 0 {
			t.Error("expected shortcode warning")
		}
	})

	t.Run("TOML frontmatter converted", func(t *testing.T) {
		src := t.TempDir()
		writeF(t, filepath.Join(src, "hugo.toml"), `title = "X"`)
		writeF(t, filepath.Join(src, "content", "toml-post.md"),
			"+++\ntitle = \"TOML Post\"\ndraft = false\n+++\nBody text")

		dst := filepath.Join(t.TempDir(), "out")
		result, err := Hugo(src, dst)
		if err != nil {
			t.Fatalf("Hugo: %v", err)
		}
		if result.Pages != 1 {
			t.Errorf("Pages = %d, want 1", result.Pages)
		}
		data, _ := os.ReadFile(filepath.Join(dst, "content", "toml-post.md"))
		if !strings.Contains(string(data), "title: TOML Post") {
			t.Errorf("expected YAML frontmatter in output, got:\n%s", data)
		}
	})

	t.Run("bad frontmatter copies as-is with warning", func(t *testing.T) {
		src := t.TempDir()
		writeF(t, filepath.Join(src, "hugo.toml"), `title = "X"`)
		writeF(t, filepath.Join(src, "content", "bad.md"),
			"---\ntitle: [bad yaml\n---\nBody")

		dst := filepath.Join(t.TempDir(), "out")
		result, err := Hugo(src, dst)
		if err != nil {
			t.Fatalf("Hugo: %v", err)
		}
		if len(result.Warnings) == 0 {
			t.Error("expected warning for bad frontmatter")
		}
		// file should still be copied
		if _, err := os.Stat(filepath.Join(dst, "content", "bad.md")); err != nil {
			t.Error("bad.md should be copied as-is")
		}
	})

	t.Run("destination already exists returns error", func(t *testing.T) {
		src := t.TempDir()
		dst := t.TempDir() // already exists
		_, err := Hugo(src, dst)
		if err == nil {
			t.Error("expected error when destination already exists")
		}
	})

	t.Run("no content or static dirs is fine", func(t *testing.T) {
		src := t.TempDir()
		writeF(t, filepath.Join(src, "hugo.toml"), `title = "X"`)
		dst := filepath.Join(t.TempDir(), "out")
		result, err := Hugo(src, dst)
		if err != nil {
			t.Fatalf("Hugo: %v", err)
		}
		if result.Pages != 0 || result.Static != 0 {
			t.Errorf("expected 0 pages and 0 static, got %d/%d", result.Pages, result.Static)
		}
	})

	t.Run("non-md files in content are skipped", func(t *testing.T) {
		src := t.TempDir()
		writeF(t, filepath.Join(src, "hugo.toml"), `title = "X"`)
		writeF(t, filepath.Join(src, "content", "post.md"), "---\ntitle: Post\n---\nBody")
		writeF(t, filepath.Join(src, "content", "image.png"), "fake image data")

		dst := filepath.Join(t.TempDir(), "out")
		result, err := Hugo(src, dst)
		if err != nil {
			t.Fatalf("Hugo: %v", err)
		}
		if result.Pages != 1 {
			t.Errorf("Pages = %d, want 1 (non-md should be skipped)", result.Pages)
		}
		if _, err := os.Stat(filepath.Join(dst, "content", "image.png")); err == nil {
			t.Error("image.png should not appear under content/")
		}
	})

	t.Run("config parse error produces warning not failure", func(t *testing.T) {
		src := t.TempDir()
		// No config file — should warn and continue
		dst := filepath.Join(t.TempDir(), "out")
		result, err := Hugo(src, dst)
		if err != nil {
			t.Fatalf("Hugo: %v", err)
		}
		if len(result.Warnings) == 0 {
			t.Error("expected warning when config not found")
		}
	})

	t.Run("MkdirAll error returns error", func(t *testing.T) {
		src := t.TempDir()
		writeF(t, filepath.Join(src, "hugo.toml"), `title = "X"`)
		blocker := filepath.Join(t.TempDir(), "blocker")
		os.WriteFile(blocker, []byte("x"), 0644)
		// blocker is a file so os.MkdirAll(blocker/dst, 0755) fails
		_, err := Hugo(src, filepath.Join(blocker, "dst"))
		if err == nil {
			t.Error("expected error when dst cannot be created")
		}
	})

	t.Run("unreadable content file returns error", func(t *testing.T) {
		if runtime.GOOS == "windows" || os.Getuid() == 0 {
			t.Skip("chmod-based test not applicable on Windows or as root")
		}
		src := t.TempDir()
		writeF(t, filepath.Join(src, "hugo.toml"), `title = "X"`)
		f := filepath.Join(src, "content", "post.md")
		writeF(t, f, "---\ntitle: Post\n---\nBody")
		os.Chmod(f, 0000)
		t.Cleanup(func() { os.Chmod(f, 0644) })

		dst := filepath.Join(t.TempDir(), "out")
		_, err := Hugo(src, dst)
		if err == nil {
			t.Error("expected error for unreadable content file")
		}
	})

	t.Run("WalkDir error in content is ignored", func(t *testing.T) {
		if runtime.GOOS == "windows" || os.Getuid() == 0 {
			t.Skip("chmod-based test not applicable on Windows or as root")
		}
		src := t.TempDir()
		writeF(t, filepath.Join(src, "hugo.toml"), `title = "X"`)
		writeF(t, filepath.Join(src, "content", "post.md"), "---\ntitle: Post\n---\nBody")
		restricted := filepath.Join(src, "content", "restricted")
		os.MkdirAll(restricted, 0755)
		writeF(t, filepath.Join(restricted, "secret.md"), "secret")
		os.Chmod(restricted, 0000)
		t.Cleanup(func() { os.Chmod(restricted, 0755) })

		dst := filepath.Join(t.TempDir(), "out")
		_, err := Hugo(src, dst)
		if err != nil {
			t.Fatalf("Hugo: WalkDir errors in content should be ignored, got: %v", err)
		}
	})
}
