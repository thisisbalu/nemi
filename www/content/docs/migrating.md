---
title: "Migrating from Hugo or Jekyll"
date: 2026-01-28
description: "Convert an existing Hugo or Jekyll site to Nemi, and what to review afterward."
---

Nemi can convert an existing site so you're not starting from scratch.

```bash
nemi migrate hugo   ./my-hugo-site
nemi migrate jekyll ./my-jekyll-site
```

The destination defaults to `<src>-nemi` (pass a second argument to choose your
own). The original site is never modified.

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

## What to review

The migrator flags things it can't translate automatically — watch the output
for warnings about:

- **Hugo shortcodes** (`{{< … >}}` / `{{% … %}}`) — these are Hugo-specific and
  are left in place for you to convert or remove.
- **Jekyll Liquid** (`{% … %}`, `{{ … }}`) — same; review by hand.
- **Permalinks** — custom `permalink:` values are reported but not applied; set
  a `slug:` in frontmatter if you need to preserve a specific URL.

## After migrating

Run the linter to catch anything that broke in translation, then preview:

```bash
cd my-hugo-site-nemi
nemi check
nemi serve
```
