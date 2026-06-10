package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// errorsFor returns the messages of error-level issues for a given page id.
func messages(rep Report, errorsOnly bool) []string {
	var out []string
	for _, i := range rep.Issues {
		if errorsOnly && !i.IsError {
			continue
		}
		out = append(out, i.Page+": "+i.Message)
	}
	return out
}

func TestRun(t *testing.T) {
	t.Run("flags broken links, allows valid ones", func(t *testing.T) {
		dir := t.TempDir()
		cdir := filepath.Join(dir, "content")
		sdir := filepath.Join(dir, "static")
		write(t, filepath.Join(sdir, "logo.png"), "x")
		write(t, filepath.Join(cdir, "index.md"), "---\ntitle: Home\n---\n[about](/about/)")
		write(t, filepath.Join(cdir, "blog", "post.md"), "---\ntitle: Post\n---\nbody")
		write(t, filepath.Join(cdir, "about.md"),
			"---\ntitle: About\n---\n"+
				"[ok](/blog/post/) [section](/blog/) [img](/logo.png) "+
				"[external](https://example.com) [anchor](#top) [dead](/missing/)")

		rep, err := Run(cdir, sdir, false)
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		errs := messages(rep, true)
		if len(errs) != 1 {
			t.Fatalf("expected exactly 1 error, got %v", errs)
		}
		if !strings.Contains(errs[0], "/missing/") {
			t.Errorf("expected broken /missing/ link, got %q", errs[0])
		}
	})

	t.Run("warns on missing title", func(t *testing.T) {
		dir := t.TempDir()
		cdir := filepath.Join(dir, "content")
		write(t, filepath.Join(cdir, "untitled.md"), "---\ndescription: no title here\n---\nbody")

		rep, err := Run(cdir, filepath.Join(dir, "static"), false)
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		if rep.Errors() != 0 {
			t.Errorf("missing title should be a warning, not an error: %v", rep.Issues)
		}
		if rep.Warnings() != 1 {
			t.Errorf("expected 1 warning, got %d: %v", rep.Warnings(), rep.Issues)
		}
	})

	t.Run("flags duplicate URLs", func(t *testing.T) {
		dir := t.TempDir()
		cdir := filepath.Join(dir, "content")
		write(t, filepath.Join(cdir, "a.md"), "---\ntitle: A\nslug: dup\n---\nbody")
		write(t, filepath.Join(cdir, "b.md"), "---\ntitle: B\nslug: dup\n---\nbody")

		rep, err := Run(cdir, filepath.Join(dir, "static"), false)
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		found := false
		for _, m := range messages(rep, true) {
			if strings.Contains(m, "duplicate URL") {
				found = true
			}
		}
		if !found {
			t.Errorf("expected a duplicate URL error, got %v", rep.Issues)
		}
	})

	t.Run("clean site has no issues", func(t *testing.T) {
		dir := t.TempDir()
		cdir := filepath.Join(dir, "content")
		write(t, filepath.Join(cdir, "index.md"), "---\ntitle: Home\n---\nWelcome")
		rep, err := Run(cdir, filepath.Join(dir, "static"), false)
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		if len(rep.Issues) != 0 {
			t.Errorf("expected no issues, got %v", rep.Issues)
		}
	})
}
