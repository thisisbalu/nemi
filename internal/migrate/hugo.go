package migrate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
	"github.com/thisisbalu/nemi/internal/config"
)

// Hugo migrates a Hugo site at src into a new Nemi site at dst.
func Hugo(src, dst string) (*Result, error) {
	if _, err := os.Stat(dst); err == nil {
		return nil, fmt.Errorf("destination %q already exists", dst)
	}
	if err := os.MkdirAll(dst, 0755); err != nil {
		return nil, err
	}

	r := &Result{}

	cfg, err := loadHugoConfig(src)
	if err != nil {
		r.warn("could not read Hugo config: " + err.Error())
	}
	if err := writeNemiToml(dst, cfg); err != nil {
		return nil, err
	}

	contentSrc := filepath.Join(src, "content")
	if _, err := os.Stat(contentSrc); err == nil {
		if err := migrateHugoContent(contentSrc, filepath.Join(dst, "content"), r); err != nil {
			return nil, err
		}
	}

	staticSrc := filepath.Join(src, "static")
	if _, err := os.Stat(staticSrc); err == nil {
		n, err := copyDir(staticSrc, filepath.Join(dst, "static"))
		if err != nil {
			return nil, err
		}
		r.Static += n
	}

	return r, nil
}

func migrateHugoContent(src, dst string, r *Result) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(src, path)
		dstPath := filepath.Join(dst, rel)

		if d.IsDir() {
			return os.MkdirAll(dstPath, 0755)
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		fm, body, err := parseHugoFrontmatter(data)
		if err != nil {
			r.warn(fmt.Sprintf("%s: frontmatter parse error: %v — copied as-is", rel, err))
			return copyFile(path, dstPath)
		}
		if hasHugoShortcodes(body) {
			r.warn(fmt.Sprintf("%s: contains Hugo shortcodes — review manually", rel))
		}

		nemiFm := nemiFM{
			Title:       fm.Title,
			Date:        fmFromTime(fm.Date),
			Draft:       fm.Draft,
			Tags:        mergeTags(fm.Tags, fm.Categories),
			Description: fm.Description,
		}
		if err := writePage(dstPath, nemiFm, body); err != nil {
			return err
		}
		r.Pages++
		return nil
	})
}

func fmFromTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

func hasHugoShortcodes(body []byte) bool {
	s := string(body)
	return strings.Contains(s, "{{<") || strings.Contains(s, "{{%")
}

type hugoFM struct {
	Title       string    `toml:"title"       yaml:"title"       json:"title"`
	Date        time.Time `toml:"date"        yaml:"date"        json:"date"`
	Draft       bool      `toml:"draft"       yaml:"draft"       json:"draft"`
	Tags        []string  `toml:"tags"        yaml:"tags"        json:"tags"`
	Categories  []string  `toml:"categories"  yaml:"categories"  json:"categories"`
	Description string    `toml:"description" yaml:"description" json:"description"`
}

func parseHugoFrontmatter(data []byte) (hugoFM, []byte, error) {
	var fm hugoFM

	// TOML: +++
	if bytes.HasPrefix(data, []byte("+++\n")) {
		rest := data[4:]
		idx := bytes.Index(rest, []byte("\n+++"))
		if idx == -1 {
			return fm, data, nil
		}
		end := idx + 4
		if end < len(rest) && rest[end] == '\n' {
			end++
		}
		_, err := toml.Decode(string(rest[:idx]), &fm)
		return fm, rest[end:], err
	}

	// YAML: ---
	if bytes.HasPrefix(data, []byte("---\n")) {
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

	// JSON: opening brace
	if bytes.HasPrefix(bytes.TrimSpace(data), []byte("{")) {
		idx := bytes.Index(data, []byte("\n}"))
		if idx != -1 {
			end := idx + 2
			if end < len(data) && data[end] == '\n' {
				end++
			}
			err := json.Unmarshal(data[:idx+2], &fm)
			return fm, data[end:], err
		}
	}

	return fm, data, nil
}

func loadHugoConfig(src string) (config.Config, error) {
	candidates := []string{
		"hugo.toml", "config.toml",
		"hugo.yaml", "config.yaml", "hugo.yml", "config.yml",
		"hugo.json", "config.json",
	}
	for _, name := range candidates {
		data, err := os.ReadFile(filepath.Join(src, name))
		if err != nil {
			continue
		}
		return parseHugoConfig(data, strings.ToLower(filepath.Ext(name)))
	}
	return config.Config{}, fmt.Errorf("no Hugo config file found")
}

func parseHugoConfig(data []byte, ext string) (config.Config, error) {
	var m map[string]any
	var err error
	switch ext {
	case ".toml":
		_, err = toml.Decode(string(data), &m)
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, &m)
	case ".json":
		err = json.Unmarshal(data, &m)
	}
	if err != nil {
		return config.Config{}, err
	}

	params := mapMap(m, "params")
	cfg := config.Config{
		Title:   mapStr(m, "title"),
		BaseURL: firstNonEmpty(mapStr(m, "baseURL"), mapStr(m, "baseUrl")),
		Description: firstNonEmpty(
			mapStr(params, "description"),
			mapStr(m, "description"),
		),
	}
	cfg.Author.Name = mapStr(params, "author")
	cfg.Author.Email = mapStr(params, "email")
	cfg.Author.GitHub = firstNonEmpty(mapStr(params, "github"), mapStr(params, "githubUsername"))
	cfg.Author.Twitter = firstNonEmpty(mapStr(params, "twitter"), mapStr(params, "twitterUsername"))
	cfg.Author.LinkedIn = mapStr(params, "linkedin")
	return cfg, nil
}
