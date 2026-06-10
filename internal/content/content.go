package content

import (
	"bytes"
	"fmt"
	"html"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	"gopkg.in/yaml.v3"

	"github.com/thisisbalu/nemi/internal/config"
)

// md is the shared Markdown renderer. Syntax highlighting emits CSS classes
// (not inline styles) so themes can be restyled via static/style.css.
// AutoHeadingID gives every heading a stable id for deep links and the TOC.
// GFM adds tables, strikethrough, task lists and autolinks; Footnote and
// Typographer round out the common authoring features.
var md = goldmark.New(
	goldmark.WithExtensions(
		highlighting.NewHighlighting(
			highlighting.WithStyle("github-dark"),
			highlighting.WithFormatOptions(
				chromahtml.WithClasses(true),
			),
		),
		extension.GFM,
		extension.Footnote,
		extension.Typographer,
	),
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
		// Convert ```mermaid blocks into mermaid nodes before the highlighter
		// sees them, so their source is emitted raw for client-side rendering.
		parser.WithASTTransformers(util.Prioritized(mermaidTransformer{}, 100)),
	),
	goldmark.WithRendererOptions(
		renderer.WithNodeRenderers(util.Prioritized(mermaidRenderer{}, 100)),
	),
)

// --- Mermaid: render ```mermaid fenced blocks as <pre class="mermaid"> ---

var kindMermaid = ast.NewNodeKind("MermaidBlock")

// mermaidBlock holds the raw diagram source lifted out of a fenced code block.
type mermaidBlock struct {
	ast.BaseBlock
	code string
}

func (n *mermaidBlock) Kind() ast.NodeKind { return kindMermaid }
func (n *mermaidBlock) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

// mermaidTransformer rewrites every ```mermaid fenced code block into a
// mermaidBlock so the syntax highlighter never tokenizes it; mermaid.js needs
// the original source verbatim.
type mermaidTransformer struct{}

func (mermaidTransformer) Transform(doc *ast.Document, reader text.Reader, _ parser.Context) {
	source := reader.Source()
	var targets []*ast.FencedCodeBlock
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if fcb, ok := n.(*ast.FencedCodeBlock); ok {
			if lang := fcb.Language(source); lang != nil && strings.EqualFold(string(lang), "mermaid") {
				targets = append(targets, fcb)
			}
		}
		return ast.WalkContinue, nil
	})
	for _, fcb := range targets {
		var b strings.Builder
		for i := 0; i < fcb.Lines().Len(); i++ {
			seg := fcb.Lines().At(i)
			b.Write(seg.Value(source))
		}
		if parent := fcb.Parent(); parent != nil {
			parent.ReplaceChild(parent, fcb, &mermaidBlock{code: b.String()})
		}
	}
}

// mermaidRenderer emits the mermaidBlock as a <pre class="mermaid"> element that
// mermaid.js (loaded only on pages that need it) renders in the browser.
type mermaidRenderer struct{}

func (mermaidRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(kindMermaid, renderMermaid)
}

func renderMermaid(w util.BufWriter, _ []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		w.WriteString(`<pre class="mermaid">`)
		w.WriteString(escapeMermaid(n.(*mermaidBlock).code))
		w.WriteString("</pre>\n")
	}
	return ast.WalkSkipChildren, nil
}

// escapeMermaid escapes only the characters that are unsafe in HTML text — `&`
// and `<`. Crucially it leaves `>` raw: escaping arrows to `--&gt;` breaks
// mermaid.js, which reads some elements via innerHTML and then parses the
// literal entity. Inside a <pre>, a bare `>` is valid HTML.
func escapeMermaid(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	return s
}

type Page struct {
	Title       string
	Date        time.Time
	Draft       bool
	Tags        []string
	Description string
	Slug        string
	URL         string
	Permalink   string
	Content     template.HTML
	TOC         template.HTML
	Headings    []Heading
	Summary     string
	WordCount   int
	ReadingTime int
	Section     string
	Layout      string
	Pages       []Page
	Paginator   *Paginator
	IsSection   bool
	Tech        []string
	GitHub      string
	Live        string
	Featured    bool
	Type        string
	Math        bool   // load KaTeX on this page (frontmatter math: true)
	Mermaid     bool   // page contains a ```mermaid diagram (auto-detected)
	OGImage     string // absolute URL of this page's social card (set by renderer)
}

// OGImagePath returns the root-relative path of a page's generated social card,
// e.g. slug "blog/post" → "/og/blog/post.png". The empty (home) slug maps to
// "/og/index.png". Single source of truth shared by the renderer (which builds
// the absolute og:image URL) and the build's OG-image generator.
func OGImagePath(slug string) string {
	if slug == "" {
		slug = "index"
	}
	return "/og/" + slug + ".png"
}

// Heading is one entry in a page's table of contents.
type Heading struct {
	Level int
	ID    string
	Text  string
}

// Paginator describes one page of a paginated list. It is attached to list
// pages as .Page.Paginator; .Page.Pages already holds only this page's items.
type Paginator struct {
	PageNumber int    // 1-based index of the current page
	TotalPages int    // total number of pages
	TotalItems int    // total items across all pages
	PrevURL    string // URL of the previous page, "" on the first
	NextURL    string // URL of the next page, "" on the last
}

type Site struct {
	Config config.Config
	Pages  []Page
	Data   map[string]any
}

type frontmatter struct {
	Title       string    `yaml:"title"`
	Date        time.Time `yaml:"date"`
	Draft       bool      `yaml:"draft"`
	Tags        []string  `yaml:"tags"`
	Description string    `yaml:"description"`
	Slug        string    `yaml:"slug"`
	Layout      string    `yaml:"layout"`
	Tech        []string  `yaml:"tech"`
	GitHub      string    `yaml:"github"`
	Live        string    `yaml:"live"`
	Featured    bool      `yaml:"featured"`
	Type        string    `yaml:"type"`
	Math        bool      `yaml:"math"`
}

func Load(dir string, drafts bool) ([]Page, error) {
	var pages []Page
	var srcPaths []string // content-relative source path, parallel to pages
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
			return err
		}
		page, err := parseFile(path, dir)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if !page.Draft || drafts {
			pages = append(pages, page)
			rel, _ := filepath.Rel(dir, path)
			srcPaths = append(srcPaths, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := resolveRefs(pages, srcPaths); err != nil {
		return nil, err
	}
	return pages, nil
}

// refLinkRe matches an @/ reference in a rendered link destination, capturing
// the target source path and an optional #fragment, e.g.
// href="@/blog/post.md#section".
var refLinkRe = regexp.MustCompile(`href="@/([^"#]*)(#[^"]*)?"`)

// resolveRefs rewrites @/path/to/file.md link destinations to the resolved
// output URL of that content file. srcPaths[i] is the content-relative source
// path of pages[i] (the key authors reference). A reference whose target does
// not exist fails the build, so dangling cross-links are caught at compile time
// rather than shipping as 404s.
func resolveRefs(pages []Page, srcPaths []string) error {
	refMap := make(map[string]string, len(pages))
	for i := range pages {
		refMap[srcPaths[i]] = pages[i].URL
	}
	var broken []string
	for i := range pages {
		body := string(pages[i].Content)
		if !strings.Contains(body, `href="@/`) {
			continue
		}
		resolved := refLinkRe.ReplaceAllStringFunc(body, func(m string) string {
			sub := refLinkRe.FindStringSubmatch(m)
			target, frag := sub[1], sub[2]
			url, ok := refMap[target]
			if !ok {
				broken = append(broken, fmt.Sprintf("%s: unresolved reference @/%s", srcPaths[i], target))
				return m
			}
			return fmt.Sprintf(`href="%s%s"`, url, frag)
		})
		pages[i].Content = template.HTML(resolved)
	}
	if len(broken) > 0 {
		return fmt.Errorf("broken @/ references:\n  %s", strings.Join(broken, "\n  "))
	}
	return nil
}

func parseFile(path, dir string) (Page, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Page{}, err
	}

	fm, body := splitFrontmatter(data)

	var front frontmatter
	if len(fm) > 0 {
		if err := yaml.Unmarshal(fm, &front); err != nil {
			return Page{}, err
		}
	}

	doc := md.Parser().Parse(text.NewReader(body))
	var buf bytes.Buffer
	if err := md.Renderer().Render(&buf, body, doc); err != nil {
		return Page{}, err
	}
	headings := extractHeadings(doc, body)
	toc := renderTOC(headings)

	words := len(strings.Fields(plainText(buf.String())))
	readingTime := (words + wordsPerMinute - 1) / wordsPerMinute
	summary := summarize(doc, body)

	slug := pathSlug(path, dir)
	section := pathSection(path, dir)
	isSection := isIndexFile(path, dir)

	layout := front.Layout
	if layout == "" {
		layout = resolveLayout(slug, section, isSection)
	}

	// A slug: override changes the URL/output path only — layout and section
	// are resolved from the file's location above, so the override is safe for
	// preserving migrated URLs without altering which template renders.
	if front.Slug != "" {
		slug = normalizeSlug(front.Slug)
	}

	return Page{
		Title:       front.Title,
		Date:        front.Date,
		Draft:       front.Draft,
		Tags:        front.Tags,
		Description: front.Description,
		Slug:        slug,
		URL:         slugURL(slug),
		Content:     template.HTML(buf.String()),
		TOC:         toc,
		Headings:    headings,
		Summary:     summary,
		WordCount:   words,
		ReadingTime: readingTime,
		Section:     section,
		IsSection:   isSection,
		Layout:      layout,
		Tech:        front.Tech,
		GitHub:      front.GitHub,
		Live:        front.Live,
		Featured:    front.Featured,
		Type:        front.Type,
		Math:        front.Math,
		Mermaid:     strings.Contains(buf.String(), `<pre class="mermaid">`),
	}, nil
}

// extractHeadings walks the document for h2/h3 headings (h1 is the page title,
// rendered by the layout) and returns them in document order. Headings without
// an id are skipped since they cannot be linked.
func extractHeadings(doc ast.Node, source []byte) []Heading {
	var headings []Heading
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		h, ok := n.(*ast.Heading)
		if !ok || h.Level < 2 || h.Level > 3 {
			return ast.WalkContinue, nil
		}
		id, ok := h.AttributeString("id")
		if !ok {
			return ast.WalkContinue, nil
		}
		headings = append(headings, Heading{
			Level: h.Level,
			ID:    string(id.([]byte)),
			Text:  headingText(h, source),
		})
		return ast.WalkContinue, nil
	})
	return headings
}

// headingText collects the plain-text content of a heading node.
func headingText(n ast.Node, source []byte) string {
	var b strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch t := c.(type) {
		case *ast.Text:
			b.Write(t.Segment.Value(source))
		case *ast.String:
			b.Write(t.Value)
		default:
			b.WriteString(headingText(c, source))
		}
	}
	return b.String()
}

// wordsPerMinute is the assumed reading speed for ReadingTime.
const wordsPerMinute = 200

// summaryWords caps the auto-generated summary length.
const summaryWords = 60

var tagRe = regexp.MustCompile(`<[^>]*>`)

// plainText strips HTML tags and collapses whitespace, yielding readable text
// for word counting.
func plainText(htmlStr string) string {
	s := tagRe.ReplaceAllString(htmlStr, " ")
	s = html.UnescapeString(s)
	return strings.Join(strings.Fields(s), " ")
}

// summarize returns a plain-text excerpt from the document's first paragraph,
// truncated to summaryWords words.
func summarize(doc ast.Node, source []byte) string {
	var para ast.Node
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering && para == nil {
			if _, ok := n.(*ast.Paragraph); ok {
				para = n
				return ast.WalkStop, nil
			}
		}
		return ast.WalkContinue, nil
	})
	if para == nil {
		return ""
	}
	fields := strings.Fields(nodeText(para, source))
	if len(fields) <= summaryWords {
		return strings.Join(fields, " ")
	}
	return strings.Join(fields[:summaryWords], " ") + "…"
}

// nodeText collects the plain text under a node, inserting spaces at line breaks
// so adjacent text segments don't run together.
func nodeText(n ast.Node, source []byte) string {
	var b strings.Builder
	_ = ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch t := c.(type) {
		case *ast.Text:
			b.Write(t.Segment.Value(source))
			if t.SoftLineBreak() || t.HardLineBreak() {
				b.WriteByte(' ')
			}
		case *ast.String:
			b.Write(t.Value)
		}
		return ast.WalkContinue, nil
	})
	return b.String()
}

// renderTOC builds a nested <nav> list from the page's headings. Returns empty
// when there are fewer than two headings (a TOC of one isn't worth showing).
func renderTOC(headings []Heading) template.HTML {
	if len(headings) < 2 {
		return ""
	}
	base := headings[0].Level
	for _, h := range headings {
		if h.Level < base {
			base = h.Level
		}
	}

	var b strings.Builder
	b.WriteString(`<nav class="toc"><ul>`)
	prev := base
	for i, h := range headings {
		if i > 0 {
			if h.Level > prev {
				for l := prev; l < h.Level; l++ {
					b.WriteString("<ul>")
				}
			} else {
				b.WriteString("</li>")
				for l := prev; l > h.Level; l-- {
					b.WriteString("</ul></li>")
				}
			}
		}
		fmt.Fprintf(&b, `<li><a href="#%s">%s</a>`, h.ID, html.EscapeString(h.Text))
		prev = h.Level
	}
	b.WriteString("</li>")
	for l := prev; l > base; l-- {
		b.WriteString("</ul></li>")
	}
	b.WriteString("</ul></nav>")
	return template.HTML(b.String())
}

func resolveLayout(slug, section string, isSection bool) string {
	if slug == "" {
		return "home"
	}
	for _, p := range []string{"about", "resume", "now", "uses", "speaking", "contact", "open-source", "publications"} {
		if slug == p {
			return p
		}
	}
	if isSection {
		return section + "-list"
	}
	if section != "" {
		return section + "-single"
	}
	return "single"
}

func isIndexFile(path, dir string) bool {
	rel, _ := filepath.Rel(dir, path)
	rel = filepath.ToSlash(rel)
	base := strings.TrimSuffix(rel, ".md")
	return strings.HasSuffix(base, "/_index") || strings.HasSuffix(base, "/index")
}

func splitFrontmatter(data []byte) (fm, body []byte) {
	if !bytes.HasPrefix(data, []byte("---\n")) {
		return nil, data
	}
	rest := data[4:]
	idx := bytes.Index(rest, []byte("\n---"))
	if idx == -1 {
		return nil, data
	}
	end := idx + 4
	if end < len(rest) && rest[end] == '\n' {
		end++
	}
	return rest[:idx], rest[end:]
}

func pathSlug(path, dir string) string {
	rel, _ := filepath.Rel(dir, path)
	rel = strings.TrimSuffix(rel, ".md")
	rel = filepath.ToSlash(rel)
	rel = strings.TrimSuffix(rel, "/_index")
	if rel == "index" || rel == "_index" {
		return ""
	}
	rel = strings.TrimSuffix(rel, "/index")
	return rel
}

// Urlize turns arbitrary text into a URL slug: lower-cased, with runs of
// non-alphanumeric characters collapsed to single hyphens. Used for tag URLs
// and available to templates as the urlize function.
func Urlize(s string) string {
	var b strings.Builder
	prevHyphen := false
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevHyphen = false
			continue
		}
		if !prevHyphen && b.Len() > 0 {
			b.WriteByte('-')
			prevHyphen = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// normalizeSlug cleans a user-supplied slug: forward slashes, no surrounding
// whitespace, no leading/trailing slashes. "/blog/custom/" → "blog/custom".
func normalizeSlug(s string) string {
	return strings.Trim(strings.TrimSpace(filepath.ToSlash(s)), "/")
}

func slugURL(slug string) string {
	if slug == "" {
		return "/"
	}
	return "/" + slug + "/"
}

func pathSection(path, dir string) string {
	rel, _ := filepath.Rel(dir, path)
	parts := strings.SplitN(filepath.ToSlash(rel), "/", 2)
	if len(parts) > 1 {
		return parts[0]
	}
	return ""
}
