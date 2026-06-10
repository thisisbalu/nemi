---
title: "Migrating"
date: 2026-01-28
description: "Convert an existing Hugo, Jekyll, or raw HTML site to Nemi, and what to review afterward."
---

Nemi can convert an existing site so you're not starting from scratch.

```bash
nemi migrate hugo   ./my-hugo-site
nemi migrate jekyll ./my-jekyll-site
nemi migrate html   ./my-html-pages
```

The destination defaults to `<src>-nemi` (pass a second argument to choose your
own). The original site is never modified, and every migration scaffolds the
default theme so the result **builds and serves immediately**.

## What converts

**Config** — title, base URL, description, and author details are read from your
`hugo.toml`/`config.*` or Jekyll `_config.yml` and written to `nemi.toml`.

**Content** — Markdown files move into `content/`, with frontmatter normalized
to Nemi's YAML. Hugo TOML (`+++`), YAML, and JSON frontmatter are all handled.

**Jekyll specifics** — `_posts/YYYY-MM-DD-slug.md` becomes
`content/blog/slug.md`; `_pages/` and root pages move into `content/`. A
`published: false` post is marked `draft: true`.

**Categories → tags** — Nemi has no separate categories taxonomy (sections plus
tags cover it), so Hugo/Jekyll categories are merged into `tags` rather than
dropped.

**Assets** — `static/`, `assets/`, `images/`, and similar directories are
copied into Nemi's `static/`.

## Raw HTML

`nemi migrate html` converts a plain folder of `.html`/`.htm` files:

- Each page's **main content** is extracted (the `<main>`, else `<article>`,
  else `<body>`), dropping nav/header/footer chrome, and converted to Markdown.
- The page **title** comes from `<title>` (or the first `<h1>`), and a leading
  `<h1>` is dropped so it doesn't duplicate the title the theme renders.
- Relative `src`/image paths are rewritten to root-absolute, and images and
  other assets are copied into `static/`.
- Old stylesheets and scripts are skipped — Nemi ships its own theme.

`index.html` becomes the home page; `blog/post.html` becomes `/blog/post/`.

## What to review

The migrator flags things it can't translate automatically — watch the output
for warnings about:

- **Hugo shortcodes** (`{{< … >}}` / `{{% … %}}`) — these are Hugo-specific and
  are left in place for you to convert or remove.
- **Jekyll Liquid** (`{% … %}`, `{{ … }}`) — same; review by hand.
- **Permalinks** — custom `permalink:` values are reported but not applied; set
  a `slug:` in frontmatter if you need to preserve a specific URL.
- **Relative HTML links** — links like `about.html` between raw HTML pages are
  flagged; update them to Nemi URLs (e.g. `/about/`) or `@/`-style references.

## After migrating

Run the linter to catch anything that broke in translation, then preview:

```bash
cd my-hugo-site-nemi
nemi check
nemi serve
```
