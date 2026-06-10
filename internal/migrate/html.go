package migrate

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	htmltomd "github.com/JohannesKaufmann/html-to-markdown"
	"golang.org/x/net/html"

	"github.com/thisisbalu/nemi/internal/config"
)

// HTML migrates a folder of raw .html pages at src into a new Nemi site at dst.
// Each page's main content is converted to Markdown; other files (images, etc.)
// are copied into static/. Old stylesheets/scripts are skipped — Nemi ships its
// own theme.
func HTML(src, dst string) (*Result, error) {
	if _, err := os.Stat(dst); err == nil {
		return nil, fmt.Errorf("destination %q already exists", dst)
	}
	if err := os.MkdirAll(dst, 0755); err != nil {
		return nil, err
	}

	r := &Result{}

	// Raw HTML carries no site config; seed a title from the folder name.
	if err := writeNemiToml(dst, config.Config{Title: titleFromDir(src)}); err != nil {
		return nil, err
	}
	if err := copyScaffoldTheme(dst); err != nil {
		return nil, err
	}

	conv := htmltomd.NewConverter("", true, nil)

	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		switch strings.ToLower(filepath.Ext(path)) {
		case ".html", ".htm":
			return migrateHTMLFile(path, rel, dst, conv, r)
		case ".css", ".js":
			return nil // Nemi has its own theme; drop the old assets
		default:
			if err := copyFile(path, filepath.Join(dst, "static", rel)); err != nil {
				return err
			}
			r.Static++
			return nil
		}
	})
	if err != nil {
		return nil, err
	}
	return r, nil
}

func migrateHTMLFile(path, rel, dst string, conv *htmltomd.Converter, r *Result) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		r.warn(fmt.Sprintf("%s: parse error: %v — skipped", rel, err))
		return nil
	}

	title := extractTitle(doc)
	if title == "" {
		title = titleFromDir(rel)
	}

	content := contentNode(doc)
	rewriteRefs(content, rel)

	markdown, err := conv.ConvertString(innerHTML(content))
	if err != nil {
		r.warn(fmt.Sprintf("%s: html→markdown failed: %v — skipped", rel, err))
		return nil
	}
	markdown = stripLeadingH1(markdown)

	outRel := strings.TrimSuffix(rel, filepath.Ext(rel)) + ".md"
	if err := writePage(filepath.Join(dst, "content", outRel), nemiFM{Title: title}, []byte(markdown)); err != nil {
		return err
	}
	r.Pages++
	return nil
}

// contentNode returns the most content-like region of the document: the first
// <main>, else the first <article>, else <body>, else the whole document.
func contentNode(doc *html.Node) *html.Node {
	for _, tag := range []string{"main", "article", "body"} {
		if n := findFirst(doc, tag); n != nil {
			return n
		}
	}
	return doc
}

func extractTitle(doc *html.Node) string {
	if t := findFirst(doc, "title"); t != nil {
		if s := strings.TrimSpace(textContent(t)); s != "" {
			return s
		}
	}
	if h := findFirst(doc, "h1"); h != nil {
		return strings.TrimSpace(textContent(h))
	}
	return ""
}

func findFirst(n *html.Node, tag string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if r := findFirst(c, tag); r != nil {
			return r
		}
	}
	return nil
}

func textContent(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.Join(strings.Fields(b.String()), " ")
}

func innerHTML(n *html.Node) string {
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		_ = html.Render(&b, c)
	}
	return b.String()
}

// rewriteRefs resolves relative src/href values against the page's directory.
// Asset references become root-absolute (so images copied into static/ resolve
// at any URL depth); links to other .html pages are mapped to their Nemi URL
// (about.html → /about/, blog/post.html → /blog/post/). pageRel is the page's
// path relative to the migration source.
func rewriteRefs(n *html.Node, pageRel string) {
	if n.Type == html.ElementNode {
		for i, a := range n.Attr {
			switch a.Key {
			case "src":
				if isRelativeRef(a.Val) {
					n.Attr[i].Val = resolveRef(pageRel, a.Val, false)
				}
			case "href":
				if isRelativeRef(a.Val) {
					n.Attr[i].Val = resolveRef(pageRel, a.Val, true)
				}
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		rewriteRefs(c, pageRel)
	}
}

// resolveRef turns a relative ref on the page at pageRel into a root-absolute
// URL. When isLink and the target is an .html/.htm page, it maps to that page's
// Nemi URL; otherwise it's treated as an asset path. Any #fragment is preserved.
func resolveRef(pageRel, ref string, isLink bool) string {
	frag := ""
	if i := strings.IndexByte(ref, '#'); i >= 0 {
		frag, ref = ref[i:], ref[:i]
	}
	if ref == "" {
		return frag
	}
	dir := path.Dir(filepath.ToSlash(pageRel))
	resolved := path.Clean(path.Join(dir, ref))
	if ext := strings.ToLower(path.Ext(resolved)); isLink && (ext == ".html" || ext == ".htm") {
		return htmlSourceToURL(strings.TrimSuffix(resolved, path.Ext(resolved))) + frag
	}
	return "/" + resolved + frag
}

// htmlSourceToURL maps an extension-less HTML source path to its Nemi URL:
// "index" → "/", "about" → "/about/", "blog/index" → "/blog/".
func htmlSourceToURL(p string) string {
	p = strings.TrimSuffix(p, "/index")
	if p == "index" || p == "." || p == "" {
		return "/"
	}
	return "/" + p + "/"
}

func isRelativeRef(u string) bool {
	if u == "" || strings.HasPrefix(u, "/") || strings.HasPrefix(u, "#") {
		return false
	}
	if strings.HasPrefix(u, "//") || strings.Contains(u, "://") ||
		strings.HasPrefix(u, "mailto:") || strings.HasPrefix(u, "tel:") {
		return false
	}
	return true
}

// stripLeadingH1 drops a leading "# Heading" line so it doesn't duplicate the
// title the layout already renders.
func stripLeadingH1(md string) string {
	lines := strings.Split(md, "\n")
	for i, ln := range lines {
		t := strings.TrimSpace(ln)
		if t == "" {
			continue
		}
		if strings.HasPrefix(t, "# ") {
			return strings.TrimLeft(strings.Join(append(lines[:i], lines[i+1:]...), "\n"), "\n")
		}
		break
	}
	return md
}

// titleFromDir derives a human-ish title from a path's base name:
// "my-old-site" → "My Old Site".
func titleFromDir(path string) string {
	base := filepath.Base(strings.TrimRight(path, string(filepath.Separator)))
	base = strings.TrimSuffix(base, filepath.Ext(base))
	base = strings.NewReplacer("-", " ", "_", " ").Replace(base)
	words := strings.Fields(base)
	for i, w := range words {
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	if len(words) == 0 {
		return "Migrated Site"
	}
	return strings.Join(words, " ")
}
