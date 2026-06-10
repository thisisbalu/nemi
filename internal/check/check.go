// Package check validates a site's content before it ships: broken internal
// links, dead asset references, missing titles, and duplicate URLs.
package check

import (
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/thisisbalu/nemi/internal/content"
)

// Issue is a single problem found in the content. Errors fail the check;
// warnings are advisory.
type Issue struct {
	Page    string
	Message string
	IsError bool
}

// Report collects the issues found during a check.
type Report struct {
	Issues []Issue
}

func (r *Report) add(page, msg string, isErr bool) {
	r.Issues = append(r.Issues, Issue{Page: page, Message: msg, IsError: isErr})
}

// Errors counts the failing issues.
func (r Report) Errors() int {
	n := 0
	for _, i := range r.Issues {
		if i.IsError {
			n++
		}
	}
	return n
}

// Warnings counts the advisory issues.
func (r Report) Warnings() int { return len(r.Issues) - r.Errors() }

// linkRe matches href/src attribute values in the rendered (Goldmark) HTML,
// which always quotes its attributes.
var linkRe = regexp.MustCompile(`(?:href|src)="([^"]*)"`)

// Run loads the content and validates it against the set of URLs the build
// would produce plus the files in staticDir.
func Run(contentDir, staticDir string, drafts bool) (Report, error) {
	var rep Report
	pages, err := content.Load(contentDir, drafts)
	if err != nil {
		return rep, err
	}

	valid := validPaths(pages, staticDir)
	seen := map[string]string{}

	for _, p := range pages {
		id := pageID(p)
		if p.Title == "" {
			rep.add(id, "missing title in frontmatter", false)
		}
		if prev, ok := seen[p.URL]; ok {
			rep.add(id, "duplicate URL "+p.URL+" (also from "+prev+")", true)
		} else {
			seen[p.URL] = id
		}
		for _, m := range linkRe.FindAllStringSubmatch(string(p.Content), -1) {
			target := m[1]
			if !isInternal(target) {
				continue
			}
			if _, ok := valid[normalizePath(target)]; !ok {
				rep.add(id, "broken link or asset: "+target, true)
			}
		}
	}
	return rep, nil
}

// validPaths is the set of internal paths a link may point to: every page URL,
// each section index, tag pages, and every static file. Keys are normalized
// (no trailing slash; home is "").
func validPaths(pages []content.Page, staticDir string) map[string]struct{} {
	valid := map[string]struct{}{"": {}}
	hasTags := false
	for _, p := range pages {
		valid[normalizePath(p.URL)] = struct{}{}
		if p.Section != "" {
			valid[normalizePath("/"+p.Section+"/")] = struct{}{}
		}
		for _, tag := range p.Tags {
			hasTags = true
			valid[normalizePath("/tags/"+content.Urlize(tag)+"/")] = struct{}{}
		}
	}
	if hasTags {
		valid[normalizePath("/tags/")] = struct{}{}
	}
	_ = filepath.WalkDir(staticDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(staticDir, path)
		valid["/"+filepath.ToSlash(rel)] = struct{}{}
		return nil
	})
	return valid
}

// isInternal reports whether a link target is a site-relative path (not an
// external URL, anchor, mailto, or protocol-relative link).
func isInternal(s string) bool {
	return strings.HasPrefix(s, "/") && !strings.HasPrefix(s, "//")
}

// normalizePath strips any fragment/query and trailing slash so that "/about/",
// "/about", and "/about/#top" all compare equal. Root becomes "".
func normalizePath(s string) string {
	if i := strings.IndexAny(s, "#?"); i >= 0 {
		s = s[:i]
	}
	if s != "/" {
		s = strings.TrimRight(s, "/")
	}
	if s == "/" {
		return ""
	}
	return s
}

func pageID(p content.Page) string {
	if p.Slug == "" {
		return "/ (home)"
	}
	return p.Slug
}
