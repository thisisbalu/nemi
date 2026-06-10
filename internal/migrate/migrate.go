package migrate

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/thisisbalu/nemi/internal/config"
	"github.com/thisisbalu/nemi/scaffold"
)

// copyScaffoldTheme copies the default theme — layouts/ and static/ — from the
// embedded scaffold into dst, so a migrated site builds and serves out of the
// box. Called by every migrator before source static is copied, so a source
// asset can still overlay a scaffold one of the same name.
func copyScaffoldTheme(dst string) error {
	for _, dir := range []string{"layouts", "static"} {
		err := fs.WalkDir(scaffold.FS, dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			data, err := scaffold.FS.ReadFile(p)
			if err != nil {
				return err
			}
			target := filepath.Join(dst, p)
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			return os.WriteFile(target, data, 0644)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// Result summarizes a completed migration.
type Result struct {
	Pages    int
	Static   int
	Warnings []string
}

func (r *Result) warn(msg string) {
	r.Warnings = append(r.Warnings, msg)
}

// nemiFM is the normalized frontmatter written to every migrated page.
type nemiFM struct {
	Title       string   `yaml:"title,omitempty"`
	Date        string   `yaml:"date,omitempty"`
	Draft       bool     `yaml:"draft,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
	Description string   `yaml:"description,omitempty"`
}

func writePage(path string, fm nemiFM, body []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	fmt.Fprintln(f, "---")
	enc := yaml.NewEncoder(f)
	enc.SetIndent(2)
	if err := enc.Encode(fm); err != nil {
		return err
	}
	fmt.Fprint(f, "---\n")
	_, err = f.Write(body)
	return err
}

func writeNemiToml(dst string, cfg config.Config) error {
	f, err := os.Create(filepath.Join(dst, "nemi.toml"))
	if err != nil {
		return err
	}
	defer f.Close()
	w := func(format string, args ...any) { fmt.Fprintf(f, format, args...) }
	w("title       = %q\n", cfg.Title)
	w("baseURL     = %q\n", cfg.BaseURL)
	w("description = %q\n\n", cfg.Description)
	w("[author]\n")
	if cfg.Author.Name != "" {
		w("name     = %q\n", cfg.Author.Name)
	}
	if cfg.Author.Email != "" {
		w("email    = %q\n", cfg.Author.Email)
	}
	if cfg.Author.GitHub != "" {
		w("github   = %q\n", cfg.Author.GitHub)
	}
	if cfg.Author.Twitter != "" {
		w("twitter  = %q\n", cfg.Author.Twitter)
	}
	if cfg.Author.LinkedIn != "" {
		w("linkedin = %q\n", cfg.Author.LinkedIn)
	}
	w("\n[pages]\n")
	w("about   = true\n")
	w("blog    = true\n")
	w("contact = true\n")
	return nil
}

func copyDir(src, dst string) (int, error) {
	count := 0
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		if err := copyFile(path, target); err != nil {
			return err
		}
		count++
		return nil
	})
	return count, err
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	r, err := os.Open(src)
	if err != nil {
		return err
	}
	defer r.Close()
	w, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer w.Close()
	_, err = io.Copy(w, r)
	return err
}

func mapStr(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	s, _ := m[key].(string)
	return s
}

func mapMap(m map[string]any, key string) map[string]any {
	if m == nil {
		return nil
	}
	mm, _ := m[key].(map[string]any)
	return mm
}

// mergeTags combines tags and categories into one de-duplicated list (order
// preserved). Nemi has no separate categories taxonomy — sections + tags cover
// it — so migrated Hugo/Jekyll categories become tags instead of being dropped.
func mergeTags(tags, categories []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, group := range [][]string{tags, categories} {
		for _, t := range group {
			if t == "" || seen[t] {
				continue
			}
			seen[t] = true
			out = append(out, t)
		}
	}
	return out
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
