package build

import (
	"fmt"
	"html"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/thisisbalu/nemi/internal/config"
	"github.com/thisisbalu/nemi/internal/images"
)

var (
	imgTagRe = regexp.MustCompile(`<img\b[^>]*>`)
	attrRe   = regexp.MustCompile(`([a-zA-Z_:][-a-zA-Z0-9_:.]*)\s*=\s*"([^"]*)"`)
)

// optimizeImages rewrites every <img> over a local raster source in the built
// HTML into a responsive image: it generates resized variants and emits
// srcset/sizes plus intrinsic width/height (no layout shift) and lazy loading.
// Best-effort — a source that can't be decoded is left exactly as written.
// Runs after copyStatic (sources are already in publicDir) and before minify.
func optimizeImages(publicDir string, cfg config.Config) error {
	cache := map[string]*images.Processed{} // src URL → variants (nil = couldn't process)

	return filepath.WalkDir(publicDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".html") {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out := imgTagRe.ReplaceAllStringFunc(string(raw), func(tag string) string {
			return rewriteImg(tag, publicDir, cfg, cache)
		})
		return os.WriteFile(path, []byte(out), 0644)
	})
}

// rewriteImg turns one <img> tag into a responsive one, or returns it unchanged
// when the source is external, an unsupported format, or already has a srcset.
func rewriteImg(tag, publicDir string, cfg config.Config, cache map[string]*images.Processed) string {
	attrs := map[string]string{}
	for _, m := range attrRe.FindAllStringSubmatch(tag, -1) {
		attrs[strings.ToLower(m[1])] = m[2]
	}
	src := attrs["src"]
	if src == "" || !strings.HasPrefix(src, "/") || !images.Supported(src) {
		return tag
	}
	if _, has := attrs["srcset"]; has {
		return tag
	}

	p, seen := cache[src]
	if !seen {
		res, err := images.Process(publicDir, src, cfg.Images.Widths, cfg.Images.Quality)
		if err != nil {
			cache[src] = nil // remember the failure so we don't retry on every page
			return tag
		}
		p = &res
		cache[src] = p
	}
	if p == nil {
		return tag
	}

	srcset := make([]string, len(p.Variants))
	for i, v := range p.Variants {
		srcset[i] = fmt.Sprintf("%s %dw", v.URL, v.Width)
	}

	var b strings.Builder
	fmt.Fprintf(&b, `<img src="%s" srcset="%s" sizes="%s" width="%d" height="%d"`,
		html.EscapeString(src),
		html.EscapeString(strings.Join(srcset, ", ")),
		html.EscapeString(cfg.Images.Sizes),
		p.Width, p.Height,
	)
	if alt, ok := attrs["alt"]; ok {
		fmt.Fprintf(&b, ` alt="%s"`, html.EscapeString(alt))
	}
	if title, ok := attrs["title"]; ok {
		fmt.Fprintf(&b, ` title="%s"`, html.EscapeString(title))
	}
	b.WriteString(` loading="lazy" decoding="async">`)
	return b.String()
}
