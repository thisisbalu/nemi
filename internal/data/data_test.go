package data

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestLoad(t *testing.T) {
	t.Run("decodes yaml, toml, json and nests by directory", func(t *testing.T) {
		dir := t.TempDir()
		write(t, filepath.Join(dir, "social.yaml"), "- name: GitHub\n  url: https://gh\n")
		write(t, filepath.Join(dir, "site.toml"), "tagline = \"Hello\"\n")
		write(t, filepath.Join(dir, "team", "lead.json"), `{"name":"Jane"}`)
		write(t, filepath.Join(dir, "notes.txt"), "ignored")

		got, err := Load(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		social, ok := got["social"].([]any)
		if !ok || len(social) != 1 {
			t.Fatalf("social = %#v", got["social"])
		}
		entry := social[0].(map[string]any)
		if entry["name"] != "GitHub" || entry["url"] != "https://gh" {
			t.Errorf("social[0] = %#v", entry)
		}

		site := got["site"].(map[string]any)
		if site["tagline"] != "Hello" {
			t.Errorf("site.tagline = %#v", site["tagline"])
		}

		team := got["team"].(map[string]any)
		lead := team["lead"].(map[string]any)
		if lead["name"] != "Jane" {
			t.Errorf("team.lead.name = %#v", lead["name"])
		}

		if _, present := got["notes"]; present {
			t.Error("unsupported .txt file should be ignored")
		}
	})

	t.Run("missing directory yields empty map", func(t *testing.T) {
		got, err := Load(filepath.Join(t.TempDir(), "nope"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil || len(got) != 0 {
			t.Errorf("expected empty non-nil map, got %#v", got)
		}
	})

	t.Run("malformed file returns error", func(t *testing.T) {
		dir := t.TempDir()
		write(t, filepath.Join(dir, "bad.json"), "{not json")
		if _, err := Load(dir); err == nil {
			t.Error("expected error for malformed JSON")
		}
	})
}
