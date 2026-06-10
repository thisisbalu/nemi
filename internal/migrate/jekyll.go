package migrate

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"github.com/thisisbalu/nemi/internal/config"
)

// Jekyll migrates a Jekyll site at src into a new Nemi site at dst.
func Jekyll(src, dst string) (*Result, error) {
	if _, err := os.Stat(dst); err == nil {
		return nil, fmt.Errorf("destination %q already exists", dst)
	}
	if err := os.MkdirAll(dst, 0755); err != nil {
		return nil, err
	}

	r := &Result{}

	cfg, err := loadJekyllConfig(src)
	if err != nil {
		r.warn("could not read Jekyll config: " + err.Error())
	}
	if err := writeNemiToml(dst, cfg); err != nil {
		return nil, err
	}
	if err := copyScaffoldTheme(dst); err != nil {
		return nil, err
	}

	// _posts/ → content/blog/
	postsDir := filepath.Join(src, "_posts")
	if _, err := os.Stat(postsDir); err == nil {
		if err := migrateJekyllPosts(postsDir, filepath.Join(dst, "content", "blog"), r); err != nil {
			return nil, err
		}
	}

	// _pages/ → content/
	pagesDir := filepath.Join(src, "_pages")
	if _, err := os.Stat(pagesDir); err == nil {
		if err := migrateJekyllPages(pagesDir, filepath.Join(dst, "content"), r); err != nil {
			return nil, err
		}
	}

	// root .md files → content/
	if err := migrateJekyllRootPages(src, filepath.Join(dst, "content"), r); err != nil {
		return nil, err
	}

	// assets directories → static/<dir>
	for _, dir := range []string{"assets", "images", "files", "static"} {
		assetSrc := filepath.Join(src, dir)
		if _, err := os.Stat(assetSrc); err == nil {
			n, err := copyDir(assetSrc, filepath.Join(dst, "static", dir))
			if err != nil {
				return nil, err
			}
			r.Static += n
		}
	}

	return r, nil
}

func migrateJekyllPosts(src, dst string, r *Result) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
			return err
		}

		base := strings.TrimSuffix(filepath.Base(path), ".md")
		dateStr, slug := extractPostSlug(base)

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		fm, body, parseErr := parseJekyllFrontmatter(data)
		if parseErr != nil {
			r.warn(fmt.Sprintf("_posts/%s.md: frontmatter parse error: %v", base, parseErr))
		}

		date := fm.Date
		if date.IsZero() && dateStr != "" {
			date, _ = time.Parse("2006-01-02", dateStr)
		}

		draft := fm.Draft
		if fm.Published != nil && !*fm.Published {
			draft = true
		}

		tags := mergeTags(fm.Tags, fm.Categories)

		desc := firstNonEmpty(fm.Description, fm.Excerpt)

		if fm.Permalink != "" {
			r.warn(fmt.Sprintf("_posts/%s.md: permalink %q ignored", slug, fm.Permalink))
		}
		if hasLiquid(body) {
			r.warn(fmt.Sprintf("_posts/%s.md: contains Liquid tags — review manually", slug))
		}

		nemiFm := nemiFM{
			Title:       fm.Title,
			Date:        fmFromTime(date),
			Draft:       draft,
			Tags:        tags,
			Description: desc,
		}
		if err := writePage(filepath.Join(dst, slug+".md"), nemiFm, body); err != nil {
			return err
		}
		r.Pages++
		return nil
	})
}

func migrateJekyllPages(src, dst string, r *Result) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		fm, body, _ := parseJekyllFrontmatter(data)
		if hasLiquid(body) {
			r.warn(fmt.Sprintf("%s: contains Liquid tags — review manually", rel))
		}
		nemiFm := nemiFM{
			Title:       fm.Title,
			Date:        fmFromTime(fm.Date),
			Draft:       fm.Draft,
			Tags:        mergeTags(fm.Tags, fm.Categories),
			Description: firstNonEmpty(fm.Description, fm.Excerpt),
		}
		if err := writePage(filepath.Join(dst, rel), nemiFm, body); err != nil {
			return err
		}
		r.Pages++
		return nil
	})
}

var skipRootFiles = map[string]bool{
	"README.md": true, "README": true,
	"LICENSE": true, "LICENSE.md": true, "CHANGELOG.md": true,
	"Gemfile": true, "Gemfile.lock": true,
}

func migrateJekyllRootPages(src, dst string, r *Result) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".md") {
			continue
		}
		if strings.HasPrefix(name, "_") || strings.HasPrefix(name, ".") || skipRootFiles[name] {
			continue
		}
		data, err := os.ReadFile(filepath.Join(src, name))
		if err != nil {
			return err
		}
		fm, body, _ := parseJekyllFrontmatter(data)
		if hasLiquid(body) {
			r.warn(fmt.Sprintf("%s: contains Liquid tags — review manually", name))
		}
		nemiFm := nemiFM{
			Title:       fm.Title,
			Date:        fmFromTime(fm.Date),
			Draft:       fm.Draft,
			Tags:        mergeTags(fm.Tags, fm.Categories),
			Description: firstNonEmpty(fm.Description, fm.Excerpt),
		}
		if err := writePage(filepath.Join(dst, name), nemiFm, body); err != nil {
			return err
		}
		r.Pages++
	}
	return nil
}

type jekyllFM struct {
	Title       string    `yaml:"title"`
	Date        time.Time `yaml:"date"`
	Draft       bool      `yaml:"draft"`
	Tags        []string  `yaml:"tags"`
	Categories  []string  `yaml:"categories"`
	Description string    `yaml:"description"`
	Excerpt     string    `yaml:"excerpt"`
	Published   *bool     `yaml:"published"`
	Permalink   string    `yaml:"permalink"`
}

func parseJekyllFrontmatter(data []byte) (jekyllFM, []byte, error) {
	var fm jekyllFM
	if !bytes.HasPrefix(data, []byte("---\n")) {
		return fm, data, nil
	}
	rest := data[4:]
	idx := bytes.Index(rest, []byte("\n---"))
	if idx == -1 {
		return fm, data, nil
	}
	end := idx + 4
	if end < len(rest) && rest[end] == '\n' {
		end++
	}
	err := yaml.Unmarshal(rest[:idx], &fm)
	return fm, rest[end:], err
}

func loadJekyllConfig(src string) (config.Config, error) {
	for _, name := range []string{"_config.yml", "_config.yaml"} {
		data, err := os.ReadFile(filepath.Join(src, name))
		if err != nil {
			continue
		}
		var m map[string]any
		if err := yaml.Unmarshal(data, &m); err != nil {
			return config.Config{}, err
		}

		cfg := config.Config{Title: mapStr(m, "title"), Description: mapStr(m, "description")}

		// Jekyll: full URL = url + baseurl
		base := mapStr(m, "url")
		if sub := mapStr(m, "baseurl"); sub != "" && sub != "/" {
			base += sub
		}
		cfg.BaseURL = base

		switch v := m["author"].(type) {
		case string:
			cfg.Author.Name = v
		case map[string]any:
			cfg.Author.Name = mapStr(v, "name")
			cfg.Author.Email = mapStr(v, "email")
			cfg.Author.GitHub = mapStr(v, "github")
			cfg.Author.Twitter = mapStr(v, "twitter")
			cfg.Author.LinkedIn = mapStr(v, "linkedin")
		}
		if cfg.Author.GitHub == "" {
			cfg.Author.GitHub = mapStr(m, "github_username")
		}
		if cfg.Author.Twitter == "" {
			cfg.Author.Twitter = mapStr(m, "twitter_username")
		}

		return cfg, nil
	}
	return config.Config{}, fmt.Errorf("no Jekyll config file found")
}

// extractPostSlug parses Jekyll's "YYYY-MM-DD-slug" filename convention.
func extractPostSlug(base string) (dateStr, slug string) {
	if len(base) >= 12 && base[4] == '-' && base[7] == '-' && base[10] == '-' {
		return base[:10], base[11:]
	}
	return "", base
}

func hasLiquid(body []byte) bool {
	s := string(body)
	return strings.Contains(s, "{%") || strings.Contains(s, "{{")
}
