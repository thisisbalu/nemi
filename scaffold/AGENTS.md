# Nemi Site — AI Reference

This is a [Nemi](https://github.com/lumis) static site. Nemi compiles Markdown content and Go HTML templates into plain HTML.

## Project structure

```
nemi.toml              # Site config (title, baseURL, description) — TOML format
content/                # Markdown files with YAML frontmatter
  index.md              # Home page → /
  about.md              # → /about/
  blog/                 # Blog section
    hello-world.md      # → /blog/hello-world/
layouts/                # Go html/template files (minimal/typographic theme)
  base.html             # Outer shell: header (name+nav+theme toggle), footer, {{block "content" .}}
  home.html             # Home (/): hero + recent writing + project cards + elsewhere
  single.html           # Individual page / post
  list.html             # Section index (blog, etc.)
  tags.html             # /tags/ index
  projects-list.html    # /projects/ — project cards
  projects-single.html  # one project — tech + GitHub/Live + content
  partials/             # head.html, theme-toggle.html, project-card.html
static/                 # Copied as-is to output (CSS, images, JS)
data/                   # Structured data (.yaml/.toml/.json) → .Site.Data
public/                 # Generated output — never edit directly (gitignored)
```

`nemi build` also writes:
- `public/sitemap.xml` — all pages, absolute URLs from `baseURL`
- `public/feed.xml` — RSS 2.0 of the latest `blog` posts (linked from `<head>`)
- `public/robots.txt` — permissive, points at the sitemap

Each page also gets a `<link rel="canonical">` (its `.Page.Permalink`). Set `baseURL` in `nemi.toml` to your real domain before deploying so these URLs are correct. To override `robots.txt`, drop your own in `static/` (static files win).

`nemi build` minifies HTML/CSS/JS in `public/`; `nemi serve` does not (output stays readable for debugging).

## Content frontmatter

Every `.md` file begins with YAML frontmatter:

```markdown
---
title: "Post Title"
date: 2026-01-01
draft: false
tags: ["tag1", "tag2"]
description: "Optional meta description"
---

Markdown body here.
```

- `draft: true` excludes the page from builds
- `tags: [...]` auto-generate tag pages at `/tags/<tag>/`, plus a `/tags/` index
- `slug: my/custom/url` overrides the output path/URL (handy for preserving old links); section and layout still come from the file's location
- `content/index.md` → `/`
- `content/blog/my-post.md` → `/blog/my-post/`
- Any subdirectory under `content/` auto-generates a list page (e.g. `content/blog/` → `/blog/`)

## Navigation

The header nav is driven by `[[menu]]` tables in `nemi.toml`, ordered by `weight`:

```toml
[[menu]]
name   = "Blog"
url    = "/blog/"
weight = 2

[[menu]]
name     = "GitHub"
url      = "https://github.com/me"
weight   = 3
external = true   # opens in a new tab (auto-detected for http(s) URLs)
```

Available in templates as `.Site.Config.Menu` (each item has `.Name`, `.URL`, `.Weight`, `.IsExternal`).

## Data files

Files under `data/` (`.yaml`, `.yml`, `.toml`, `.json`) are decoded and exposed as `.Site.Data`, keyed by filename. Subdirectories nest: `data/team/lead.json` → `.Site.Data.team.lead`.

```yaml
# data/social.yaml
- name: GitHub
  url: https://github.com/me
```

```html
{{range .Site.Data.social}}<a href="{{.url}}">{{.name}}</a>{{end}}
```

Use data files for structured content that isn't a page — social links, project lists, team members, etc.

## Tags

Add `tags: ["go", "web"]` to any page's frontmatter and Nemi generates:

- `/tags/<tag>/` — a list of posts with that tag (layout `tag-list`, falls back to `list.html`)
- `/tags/` — an index of all tags with counts (layout `tags.html`)

Tag slugs are URL-safe (`"web dev"` → `/tags/web-dev/`). Link to a tag with `<a href="/tags/{{urlize .}}/">`.

## Pagination

List pages (sections and tag pages) are paginated when they exceed `paginate` items (set in `nemi.toml`, default 10). Page 1 stays at the section URL; later pages are at `<url>page/2/`, `<url>page/3/`, etc. `.Page.Pages` holds only the current page's items, so existing `{{range .Page.Pages}}` loops keep working. Add navigation with the paginator:

```html
{{with .Page.Paginator}}{{if gt .TotalPages 1}}
<nav class="pagination">
  {{if .PrevURL}}<a href="{{.PrevURL}}">← Newer</a>{{end}}
  <span>Page {{.PageNumber}} of {{.TotalPages}}</span>
  {{if .NextURL}}<a href="{{.NextURL}}">Older →</a>{{end}}
</nav>
{{end}}{{end}}
```

## Template data

| Variable | Type | Description |
|---|---|---|
| `.Site.Config.Title` | string | Site title from nemi.toml |
| `.Site.Config.BaseURL` | string | Base URL from nemi.toml |
| `.Site.Config.Description` | string | Description from nemi.toml |
| `.Page.Title` | string | Page title |
| `.Page.Content` | template.HTML | Rendered HTML — output with `{{.Page.Content}}` |
| `.Page.TOC` | template.HTML | Table of contents (`<nav class="toc">`) built from h2/h3; empty if <2 headings |
| `.Page.Headings` | []Heading | Structured headings: `.Level` (int), `.ID` (string), `.Text` (string) |
| `.Page.Date` | time.Time | Use `.Page.Date.Format "Jan 2, 2006"` |
| `.Page.URL` | string | Page URL e.g. `/blog/my-post/` |
| `.Page.Permalink` | string | Absolute URL (`baseURL` + URL) — used for canonical links |
| `.Page.Tags` | []string | Tags from frontmatter |
| `.Page.Description` | string | Description from frontmatter |
| `.Page.Summary` | string | Auto excerpt from the first paragraph (≤60 words) |
| `.Page.WordCount` | int | Word count of the rendered content |
| `.Page.ReadingTime` | int | Estimated minutes to read (200 wpm) |
| `.Page.Pages` | []Page | Child pages — on a paginated list, only the current page's items |
| `.Page.Paginator` | *Paginator | On paginated lists: `.PageNumber`, `.TotalPages`, `.TotalItems`, `.PrevURL`, `.NextURL` (nil if unpaginated) |
| `.Page.IsSection` | bool | True on section/list pages |

## Markdown features

- **Syntax highlighting** — fenced code blocks with a language (```` ```go ````) are highlighted at build time using CSS classes (Chroma "github-dark"). Token colors live in `static/style.css`; restyle or swap themes there.
- **Heading anchors** — every `##`/`###` heading gets a stable `id`, so `/post/#section` deep links work.
- **Table of contents** — output `{{.Page.TOC}}` for a ready-made nested list, or loop `{{range .Page.Headings}}` to build your own.
- **GitHub-Flavored Markdown** — tables, `~~strikethrough~~`, task lists (`- [ ]` / `- [x]`), and bare-URL autolinking all work.
- **Footnotes** — `Text[^1]` with `[^1]: note` renders a linked footnotes section.
- **Smart typography** — straight quotes, `--`, and `...` become curly quotes, en-dashes, and ellipses automatically.

## Template inheritance

`base.html` defines the outer layout and declares a replaceable block:

```html
{{block "content" .}}{{end}}
```

`single.html` and `list.html` fill that block:

```html
{{define "content"}}
  <h1>{{.Page.Title}}</h1>
  {{.Page.Content}}
{{end}}
```

Parse order: `base.html` + `single.html` are parsed together. Execute `base.html`.

### Partials

Reusable fragments live in `layouts/partials/`. Each file is available as a named template — its name is the path under `partials/` without `.html`. They receive the same data (`.Site`, `.Page`):

```html
<!-- layouts/partials/head.html -->
<title>{{.Page.Title}} · {{.Site.Config.Title}}</title>

<!-- used in base.html -->
<head>{{template "head" .}}</head>
```

Subdirectories work too: `partials/icons/logo.html` → `{{template "icons/logo" .}}`.

## Template functions

| Function | Example | Result |
|---|---|---|
| `not` | `{{if not .Page.Draft}}` | logical negation |
| `now` | `© {{now.Year}}` | current time (a `time.Time`) |
| `dateFormat` | `{{dateFormat "Jan 2, 2006" .Page.Date}}` | formatted date string |
| `lower` / `upper` / `title` | `{{title .Page.Section}}` | case conversion |
| `urlize` | `{{urlize .}}` | URL slug, e.g. `Hello World` → `hello-world` |
| `truncate` | `{{truncate 120 .Page.Description}}` | shorten to N runes + ellipsis (word-aware) |
| `first` | `{{range first 5 .Page.Pages}}` | first N items of a slice |
| `where` | `{{where .Site.Pages "Featured" true}}` | pages whose field equals a value |
| `sortByDate` | `{{range sortByDate .Page.Pages}}` | pages sorted newest-first |

## CLI commands

```bash
nemi new <name>          # scaffold a new site from the built-in template
nemi new post "Title"    # scaffold a draft post in content/blog/ (-s for another section)
nemi build               # build site into public/
nemi serve               # dev server at http://localhost:3000 with live reload
nemi check               # validate content: broken links, dead assets, missing titles
nemi version             # print the Nemi version
```

## Common tasks

| Task | How |
|---|---|
| New blog post | Create `content/blog/my-post.md` with frontmatter |
| New page | Create `content/my-page.md` with frontmatter |
| New section | Create `content/projects/` directory with `.md` files |
| Change nav | Edit the `[[menu]]` entries in `nemi.toml` |
| Add styles | Edit `static/style.css` or add files to `static/` |
| Add structured data | Create `data/<name>.yaml` → read as `.Site.Data.<name>` |
| Change site title | Edit `title` in `nemi.toml` |
| Change base URL | Edit `baseURL` in `nemi.toml` |

## Theme

The default theme is minimal and typographic: a single ~680px column, system fonts, near-monochrome (links are underlined ink — code blocks are the only color). All colors are CSS custom properties at the top of `static/style.css` (`--bg`, `--text`, `--muted`, `--border`, `--accent`-free by design) — restyle the whole site there.

**Dark mode** follows the OS preference and has a manual toggle (◐ in the header). It's driven by `data-theme` on `<html>`: a no-flash inline script in `partials/head.html` applies the saved choice before paint, and `partials/theme-toggle.html` flips/persists it. To change palettes, edit the `:root`, `@media (prefers-color-scheme: dark)`, and `[data-theme="dark"]` blocks in `static/style.css`.

The home page (`home.html`) pulls its hero from `[author]` in `nemi.toml` and lists recent posts / projects using the built-in template functions (`first`, `where`, `sortByDate`) — no data wiring needed.

## Rules

- Never edit `public/` — it is deleted and regenerated on every build
- Static assets go in `static/` and are referenced as `/filename.ext`
- `nemi.toml` is TOML, not YAML
- `.Page.Content` is `template.HTML` — it is already safe, output with `{{.Page.Content}}`
- A page with `draft: true` is excluded from all builds
