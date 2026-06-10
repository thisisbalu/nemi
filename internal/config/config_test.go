package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	t.Run("all fields", func(t *testing.T) {
		dir := t.TempDir()
		write(t, filepath.Join(dir, "nemi.toml"), `
title       = "My Site"
baseURL     = "https://example.com"
description = "A site"

[author]
name     = "Alice"
tagline  = "Builder"
bio      = "I build things"
email    = "alice@example.com"
github   = "alice"
twitter  = "alicetw"
linkedin = "aliceli"

[pages]
about    = true
resume   = true
projects = true
blog     = true
updates  = true
now      = true
uses     = true
speaking = true
contact  = true
`)
		cfg, err := Load(filepath.Join(dir, "nemi.toml"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Title != "My Site" {
			t.Errorf("Title = %q", cfg.Title)
		}
		if cfg.BaseURL != "https://example.com" {
			t.Errorf("BaseURL = %q", cfg.BaseURL)
		}
		if cfg.Description != "A site" {
			t.Errorf("Description = %q", cfg.Description)
		}
		if cfg.Author.Name != "Alice" {
			t.Errorf("Author.Name = %q", cfg.Author.Name)
		}
		if cfg.Author.Tagline != "Builder" {
			t.Errorf("Author.Tagline = %q", cfg.Author.Tagline)
		}
		if cfg.Author.Bio != "I build things" {
			t.Errorf("Author.Bio = %q", cfg.Author.Bio)
		}
		if cfg.Author.Email != "alice@example.com" {
			t.Errorf("Author.Email = %q", cfg.Author.Email)
		}
		if cfg.Author.GitHub != "alice" {
			t.Errorf("Author.GitHub = %q", cfg.Author.GitHub)
		}
		if cfg.Author.Twitter != "alicetw" {
			t.Errorf("Author.Twitter = %q", cfg.Author.Twitter)
		}
		if cfg.Author.LinkedIn != "aliceli" {
			t.Errorf("Author.LinkedIn = %q", cfg.Author.LinkedIn)
		}
		if !cfg.Pages.About || !cfg.Pages.Resume || !cfg.Pages.Projects ||
			!cfg.Pages.Blog || !cfg.Pages.Updates || !cfg.Pages.Now ||
			!cfg.Pages.Uses || !cfg.Pages.Speaking || !cfg.Pages.Contact {
			t.Error("one or more Pages flags not true")
		}
	})

	t.Run("menu sorted by weight, external detected", func(t *testing.T) {
		dir := t.TempDir()
		write(t, filepath.Join(dir, "nemi.toml"), `
title = "Site"

[[menu]]
name = "Blog"
url = "/blog/"
weight = 2

[[menu]]
name = "About"
url = "/about/"
weight = 1

[[menu]]
name = "GitHub"
url = "https://github.com/me"
weight = 3
`)
		cfg, err := Load(filepath.Join(dir, "nemi.toml"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cfg.Menu) != 3 {
			t.Fatalf("Menu len = %d, want 3", len(cfg.Menu))
		}
		if cfg.Menu[0].Name != "About" || cfg.Menu[1].Name != "Blog" || cfg.Menu[2].Name != "GitHub" {
			t.Errorf("menu order = %v", []string{cfg.Menu[0].Name, cfg.Menu[1].Name, cfg.Menu[2].Name})
		}
		if cfg.Menu[0].IsExternal() {
			t.Error("/about/ should not be external")
		}
		if !cfg.Menu[2].IsExternal() {
			t.Error("https://github.com/me should be external")
		}
	})

	t.Run("empty file", func(t *testing.T) {
		dir := t.TempDir()
		write(t, filepath.Join(dir, "nemi.toml"), "")
		cfg, err := Load(filepath.Join(dir, "nemi.toml"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Title != "" {
			t.Errorf("expected empty title, got %q", cfg.Title)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := Load(filepath.Join(t.TempDir(), "nemi.toml"))
		if err == nil {
			t.Error("expected error for missing file")
		}
	})

	t.Run("invalid TOML", func(t *testing.T) {
		dir := t.TempDir()
		write(t, filepath.Join(dir, "nemi.toml"), "title = [invalid")
		_, err := Load(filepath.Join(dir, "nemi.toml"))
		if err == nil {
			t.Error("expected error for invalid TOML")
		}
	})
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
