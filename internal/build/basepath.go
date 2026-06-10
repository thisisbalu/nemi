package build

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// href/src attributes whose value is a root-relative path.
	attrURLRe = regexp.MustCompile(`(href|src)="(/[^"]*)"`)
	// a srcset attribute (comma-separated "url descriptor" entries).
	srcsetRe = regexp.MustCompile(`srcset="([^"]*)"`)
	// CSS url(...) with an optional quote, pointing at a root-relative path.
	cssURLRe = regexp.MustCompile(`url\(\s*(['"]?)(/[^)'"]*)['"]?\s*\)`)
)

// applyBasePath rewrites root-relative URLs in the built HTML/CSS to sit under
// basePath, so a site can be served from a subpath (e.g. /nemi/ on GitHub
// project pages). It is a no-op when basePath is empty, and it never touches
// protocol-relative ("//host") or absolute ("https://") URLs. Runs after the
// image pass (so generated srcset URLs are covered) and before minify.
func applyBasePath(publicDir, basePath string) error {
	if basePath == "" {
		return nil
	}
	return filepath.WalkDir(publicDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".html" && ext != ".css" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out := rewriteBasePath(string(raw), basePath, ext == ".html")
		return os.WriteFile(path, []byte(out), 0644)
	})
}

func rewriteBasePath(s, base string, html bool) string {
	if html {
		s = attrURLRe.ReplaceAllStringFunc(s, func(m string) string {
			sub := attrURLRe.FindStringSubmatch(m)
			return sub[1] + `="` + prefix(base, sub[2]) + `"`
		})
		s = srcsetRe.ReplaceAllStringFunc(s, func(m string) string {
			sub := srcsetRe.FindStringSubmatch(m)
			return `srcset="` + prefixSrcset(base, sub[1]) + `"`
		})
	}
	s = cssURLRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := cssURLRe.FindStringSubmatch(m)
		return "url(" + sub[1] + prefix(base, sub[2]) + sub[1] + ")"
	})
	return s
}

// prefix prepends base to a root-relative path, leaving protocol-relative URLs
// ("//cdn…") untouched.
func prefix(base, u string) string {
	if strings.HasPrefix(u, "//") {
		return u
	}
	return base + u
}

// prefixSrcset prefixes the URL in each comma-separated srcset candidate.
func prefixSrcset(base, val string) string {
	parts := strings.Split(val, ",")
	for i, p := range parts {
		fields := strings.Fields(strings.TrimSpace(p))
		if len(fields) == 0 {
			continue
		}
		if strings.HasPrefix(fields[0], "/") {
			fields[0] = prefix(base, fields[0])
		}
		parts[i] = strings.Join(fields, " ")
	}
	return strings.Join(parts, ", ")
}
