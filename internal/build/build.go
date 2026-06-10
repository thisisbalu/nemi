package build

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/thisisbalu/nemi/internal/config"
	"github.com/thisisbalu/nemi/internal/content"
	"github.com/thisisbalu/nemi/internal/data"
	"github.com/thisisbalu/nemi/internal/feed"
	"github.com/thisisbalu/nemi/internal/minify"
	"github.com/thisisbalu/nemi/internal/ogimage"
	"github.com/thisisbalu/nemi/internal/renderer"
	"github.com/thisisbalu/nemi/internal/sitemap"
)

// Stats summarizes a completed build.
type Stats struct {
	Pages int
}

func Run(serveMode, drafts bool) (Stats, error) {
	var stats Stats

	cfg, err := config.Load("nemi.toml")
	if err != nil {
		return stats, err
	}

	pages, err := content.Load("content", drafts)
	if err != nil {
		return stats, err
	}

	siteData, err := data.Load("data")
	if err != nil {
		return stats, err
	}

	site := content.Site{Config: cfg, Pages: pages, Data: siteData}

	if err := os.RemoveAll("public"); err != nil {
		return stats, err
	}

	r := renderer.New("layouts", "public", serveMode)
	if err := r.Render(site); err != nil {
		return stats, err
	}
	stats.Pages = len(r.Rendered())

	if err := sitemap.Write("public", cfg.BaseURL, r.Rendered()); err != nil {
		return stats, err
	}

	if err := feed.Write("public", cfg, pages); err != nil {
		return stats, err
	}

	if err := writeRobots("public", cfg.BaseURL); err != nil {
		return stats, err
	}

	if err := copyStatic("static", "public"); err != nil {
		return stats, err
	}

	// Production-only post-passes; serve keeps output readable and rebuilds fast.
	if !serveMode {
		if !cfg.OG.Disable {
			generateOGImages("public", cfg, r.Rendered())
		}
		if !cfg.Images.Disable {
			if err := optimizeImages("public", cfg); err != nil {
				return stats, err
			}
		}
		return stats, minifyTree("public")
	}
	return stats, nil
}

// generateOGImages writes a 1200×630 social card per rendered page to
// public/og/<slug>.png. Best-effort: a page whose card can't be rendered is
// skipped rather than failing the build. Runs before optimizeImages, but the
// /og/ PNGs are referenced from <meta> (not <img>), so they're untouched by it.
func generateOGImages(publicDir string, cfg config.Config, pages []content.Page) {
	for _, p := range pages {
		if p.Slug == "404" {
			continue
		}
		title := p.Title
		if title == "" {
			title = cfg.Title
		}
		out := filepath.Join(publicDir, filepath.FromSlash(strings.TrimPrefix(content.OGImagePath(p.Slug), "/")))
		_ = ogimage.Generate(out, title, cfg.Title)
	}
}

// minifyTree minifies every .html/.css/.js file under dir in place. Minification
// is best-effort: a file that fails to minify is left as-is rather than failing
// the build.
func minifyTree(dir string) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		var fn func([]byte) ([]byte, error)
		switch strings.ToLower(filepath.Ext(path)) {
		case ".html":
			fn = minify.HTML
		case ".css":
			fn = minify.CSS
		case ".js":
			fn = minify.JS
		default:
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out, err := fn(data)
		if err != nil {
			return nil // leave the original file untouched
		}
		return os.WriteFile(path, out, 0644)
	})
}

// writeRobots emits a permissive robots.txt that advertises the sitemap. The
// Sitemap line is omitted when no baseURL is configured (it must be absolute).
// copyStatic runs afterward, so a user-provided static/robots.txt wins.
func writeRobots(dir, baseURL string) error {
	var b strings.Builder
	b.WriteString("User-agent: *\nAllow: /\n")
	if baseURL != "" {
		b.WriteString("\nSitemap: " + strings.TrimRight(baseURL, "/") + "/sitemap.xml\n")
	}
	return os.WriteFile(filepath.Join(dir, "robots.txt"), []byte(b.String()), 0644)
}

func copyStatic(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		rel, _ := filepath.Rel(src, path)
		dest := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(dest, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dest, data, 0644)
	})
}
