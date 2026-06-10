package renderer

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/thisisbalu/nemi/internal/content"
)

type TemplateData struct {
	Site      content.Site
	Page      content.Page
	ServeMode bool
}

type Renderer struct {
	layoutsDir string
	outputDir  string
	serveMode  bool
	partials   []partial
	pageSize   int
	rendered   []content.Page
}

// defaultPageSize is used when nemi.toml does not set a positive paginate value.
const defaultPageSize = 10

// partial is a reusable template fragment from layouts/partials/. Its name is
// the path under partials/ without the .html suffix, so layouts/partials/head.html
// is invoked as {{template "head" .}}.
type partial struct {
	name    string
	content string
}

func New(layoutsDir, outputDir string, serveMode bool) *Renderer {
	return &Renderer{
		layoutsDir: layoutsDir,
		outputDir:  outputDir,
		serveMode:  serveMode,
	}
}

func (r *Renderer) Render(site content.Site) error {
	if err := os.MkdirAll(r.outputDir, 0755); err != nil {
		return err
	}

	if err := r.loadPartials(); err != nil {
		return err
	}

	r.pageSize = site.Config.Paginate
	if r.pageSize <= 0 {
		r.pageSize = defaultPageSize
	}
	r.rendered = nil

	// Group non-index pages by section
	sectionPages := map[string][]content.Page{}
	hasIndex := map[string]bool{}

	for _, p := range site.Pages {
		if p.IsSection {
			hasIndex[p.Section] = true
		} else {
			sectionPages[p.Section] = append(sectionPages[p.Section], p)
		}
	}

	// Render explicit pages; populate .Pages for section list pages
	for _, page := range site.Pages {
		if page.IsSection {
			page.Pages = sortByDate(sectionPages[page.Section])
		}
		if err := r.renderListPage(site, page); err != nil {
			return err
		}
	}

	// Auto-generate list pages for sections with no _index.md
	for section, pages := range sectionPages {
		if section == "" || hasIndex[section] {
			continue
		}
		list := content.Page{
			Title:     capitalize(section),
			URL:       "/" + section + "/",
			Slug:      section,
			Section:   section,
			IsSection: true,
			Pages:     sortByDate(pages),
			Layout:    section + "-list",
		}
		if err := r.renderListPage(site, list); err != nil {
			return err
		}
	}

	if err := r.renderTags(site); err != nil {
		return err
	}

	return nil
}

// renderTags generates a term page at /tags/<tag>/ for every tag used in the
// content, plus a /tags/ index listing those terms. Term pages use the
// "tag-list" layout (falling back to list.html) and expose the tagged posts as
// .Page.Pages; the index uses the "tags" layout and exposes the term pages as
// .Page.Pages (each carrying its own posts, so templates can show a count).
// Slugs that already exist in the content are left untouched.
func (r *Renderer) renderTags(site content.Site) error {
	existing := map[string]bool{}
	for _, p := range site.Pages {
		existing[p.Slug] = true
	}

	byTag := map[string][]content.Page{}
	for _, p := range site.Pages {
		if p.IsSection {
			continue
		}
		for _, tag := range p.Tags {
			byTag[tag] = append(byTag[tag], p)
		}
	}
	if len(byTag) == 0 {
		return nil
	}

	names := make([]string, 0, len(byTag))
	for name := range byTag {
		names = append(names, name)
	}
	sort.Strings(names)

	terms := make([]content.Page, 0, len(names))
	for _, name := range names {
		slug := "tags/" + content.Urlize(name)
		term := content.Page{
			Title:     name,
			Slug:      slug,
			URL:       "/" + slug + "/",
			Section:   "tags",
			IsSection: true,
			Layout:    "tag-list",
			Pages:     sortByDate(byTag[name]),
		}
		terms = append(terms, term)
		if existing[slug] {
			continue
		}
		if err := r.renderListPage(site, term); err != nil {
			return err
		}
	}

	if existing["tags"] {
		return nil
	}
	index := content.Page{
		Title:     "Tags",
		Slug:      "tags",
		URL:       "/tags/",
		Section:   "tags",
		IsSection: true,
		Layout:    "tags",
		Pages:     terms,
	}
	return r.renderListPage(site, index)
}

// renderListPage renders a page, splitting it across numbered pages when it is
// a section/list page with more items than the page size. Page 1 keeps the base
// URL; later pages live at <url>page/<n>/. Non-list pages render once.
func (r *Renderer) renderListPage(site content.Site, page content.Page) error {
	items := page.Pages
	if !page.IsSection || r.pageSize <= 0 || len(items) <= r.pageSize {
		return r.renderPage(site, page)
	}

	total := (len(items) + r.pageSize - 1) / r.pageSize
	for i := 0; i < total; i++ {
		start := i * r.pageSize
		end := start + r.pageSize
		if end > len(items) {
			end = len(items)
		}
		n := i + 1
		p := page
		p.Pages = items[start:end]
		p.Paginator = &content.Paginator{
			PageNumber: n,
			TotalPages: total,
			TotalItems: len(items),
			PrevURL:    paginatedURL(page.URL, n-1, total),
			NextURL:    paginatedURL(page.URL, n+1, total),
		}
		if n > 1 {
			p.Slug = page.Slug + "/page/" + strconv.Itoa(n)
			p.URL = page.URL + "page/" + strconv.Itoa(n) + "/"
		}
		if err := r.renderPage(site, p); err != nil {
			return err
		}
	}
	return nil
}

// absURL joins a site base URL with a root-relative page path. An empty base
// leaves the path relative.
func absURL(base, path string) string {
	return strings.TrimRight(base, "/") + path
}

// paginatedURL returns the URL of page n given the list's base URL, or "" when
// n is out of range. Page 1 maps back to the base URL.
func paginatedURL(baseURL string, n, total int) string {
	switch {
	case n < 1 || n > total:
		return ""
	case n == 1:
		return baseURL
	default:
		return baseURL + "page/" + strconv.Itoa(n) + "/"
	}
}

func (r *Renderer) renderPage(site content.Site, page content.Page) error {
	tmpl, err := r.loadTemplate(page.Layout)
	if err != nil {
		return fmt.Errorf("layout %q: %w", page.Layout, err)
	}

	page.Permalink = absURL(site.Config.BaseURL, page.URL)
	if page.Slug != "404" {
		page.OGImage = absURL(site.Config.BaseURL, content.OGImagePath(page.Slug))
	}

	var buf bytes.Buffer
	data := TemplateData{Site: site, Page: page, ServeMode: r.serveMode}
	if err := tmpl.ExecuteTemplate(&buf, "base", data); err != nil {
		return fmt.Errorf("render %q: %w", page.URL, err)
	}

	outPath := r.outputPath(page.Slug)
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(outPath, buf.Bytes(), 0644); err != nil {
		return err
	}
	r.rendered = append(r.rendered, page)
	return nil
}

// Rendered returns every page written during the last Render, including
// auto-generated section, tag, and paginated pages. Used to build the sitemap.
func (r *Renderer) Rendered() []content.Page {
	return r.rendered
}

func (r *Renderer) loadTemplate(layout string) (*template.Template, error) {
	baseContent, err := r.templateContent("base.html")
	if err != nil {
		return nil, err
	}
	layoutContent, err := r.templateContent(layout + ".html")
	if err != nil {
		fallback := "single.html"
		if strings.HasSuffix(layout, "-list") {
			fallback = "list.html"
		}
		layoutContent, err = r.templateContent(fallback)
		if err != nil {
			return nil, err
		}
	}

	tmpl := template.New("base").Funcs(funcMap)
	if _, err := tmpl.Parse(baseContent); err != nil {
		return nil, fmt.Errorf("parse base.html: %w", err)
	}
	if _, err := tmpl.New(layout + ".html").Parse(layoutContent); err != nil {
		return nil, fmt.Errorf("parse %s.html: %w", layout, err)
	}
	for _, p := range r.partials {
		if _, err := tmpl.New(p.name).Parse(p.content); err != nil {
			return nil, fmt.Errorf("parse partial %q: %w", p.name, err)
		}
	}
	return tmpl, nil
}

// loadPartials reads layouts/partials/ once per build. A missing partials
// directory is not an error. Partial names are paths relative to partials/
// with the .html suffix removed (e.g. icons/logo.html → "icons/logo").
func (r *Renderer) loadPartials() error {
	r.partials = nil
	dir := filepath.Join(r.layoutsDir, "partials")
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".html") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		name := strings.TrimSuffix(filepath.ToSlash(rel), ".html")
		r.partials = append(r.partials, partial{name: name, content: string(data)})
		return nil
	})
}

func (r *Renderer) templateContent(name string) (string, error) {
	data, err := os.ReadFile(filepath.Join(r.layoutsDir, name))
	if err != nil {
		return "", fmt.Errorf("layout %q not found in %s/", name, r.layoutsDir)
	}
	return string(data), nil
}

func (r *Renderer) outputPath(slug string) string {
	if slug == "" {
		return filepath.Join(r.outputDir, "index.html")
	}
	if slug == "404" {
		return filepath.Join(r.outputDir, "404.html")
	}
	return filepath.Join(r.outputDir, filepath.FromSlash(slug), "index.html")
}

func sortByDate(pages []content.Page) []content.Page {
	out := make([]content.Page, len(pages))
	copy(out, pages)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Date.After(out[j].Date)
	})
	return out
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
