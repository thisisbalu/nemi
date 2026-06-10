---
title: "Templating"
date: 2026-01-31
description: "Layouts, partials, the template functions, and the page fields available to your templates."
---

Nemi renders pages with Go's `html/template`. The theme lives in `layouts/`, and
you can edit any of it — there's no hidden magic.

## How a layout is chosen

For each page, Nemi resolves a layout in this order:

1. A `layout:` key in the page's frontmatter wins outright.
2. Known slugs map to named layouts — `about`, `resume`, `now`, `uses`,
   `contact`, `publications`, etc.
3. Section pages use `<section>-list` (for `_index.md`) or `<section>-single`.
4. Everything else falls back to `single.html`; list pages fall back to
   `list.html`.

So `content/blog/hello.md` renders with `blog-single.html` if it exists,
otherwise `single.html`.

## The base template

Every layout is injected into `base.html`, which must define the `base`
template and yield a `content` block:

```html
{{define "base"}}<!doctype html>
<html>
  <head>{{template "head" .}}</head>
  <body>
    <main>{{block "content" .}}{{end}}</main>
  </body>
</html>
{{end}}
```

A layout fills that block:

```html
{{define "content"}}
  <h1>{{.Page.Title}}</h1>
  <div class="prose">{{.Page.Content}}</div>
{{end}}
```

`.Page.Content` is already-sanitized `template.HTML` — output it directly with
`{{.Page.Content}}`, never `{{html .Page.Content}}`.

## Partials

Files in `layouts/partials/` are reusable fragments. A partial is named by its
path under `partials/` minus `.html`, so `partials/head.html` is used as
`{{template "head" .}}` and `partials/icons/logo.html` as
`{{template "icons/logo" .}}`.

## Template functions

Beyond Go's built-ins, these are available everywhere:

| Function | Purpose |
|---|---|
| `now` / `dateFormat` | current time / format a time (`{{dateFormat "Jan 2, 2006" .Page.Date}}`) |
| `lower` `upper` `title` | case helpers |
| `urlize` | turn text into a URL slug |
| `truncate` | shorten to N runes on a word boundary |
| `first` | first N items of a slice |
| `where` | filter pages by field (`{{where .Site.Pages "Section" "blog"}}`) |
| `sortByDate` | newest-first sort |
| `uniqueTags` | de-duplicated, sorted tags across pages |

## Page fields

`.Page` exposes (less obvious ones):

| Field | Source | Purpose |
|---|---|---|
| `Pages` | — | child pages on a section list page |
| `IsSection` | — | true for `_index.md` |
| `TOC` | — | nested `<nav>` of h2/h3 (empty if < 2 headings) |
| `Headings` | — | `[]Heading{Level, ID, Text}` |
| `Summary` | — | first-paragraph excerpt (≤ 60 words) |
| `WordCount` / `ReadingTime` | — | reading time = ceil(words / 200) |
| `Permalink` | — | absolute canonical URL |
| `Tech` / `GitHub` / `Live` | frontmatter | project metadata |
| `Featured` | `featured` | highlight on list pages |
| `Math` / `Mermaid` | — | whether to load KaTeX / mermaid |

`.Site` exposes `.Config` (everything from `nemi.toml`), `.Pages` (all pages),
and `.Data` (your `data/` directory). `.ServeMode` is true under `nemi serve`.

## Overriding the theme

Just edit the files in `layouts/`. To change the markup of every post, edit
`single.html`; to change the header, edit `base.html`. Since the whole theme
ships in your site, nothing is locked away.
