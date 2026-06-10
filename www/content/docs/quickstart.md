---
title: "Quick start"
date: 2026-02-05
description: "Scaffold a site, run the dev server, and build to static HTML in three commands."
---

## Create a site

```bash
nemi new mysite
cd mysite
```

This scaffolds a complete site — config, content, the default theme, and static
assets.

## Run the dev server

```bash
nemi serve
```

Open [localhost:3000](http://localhost:3000). The server watches your files and
live-reloads the browser on every change.

## Build for production

```bash
nemi build
```

Your site is compiled into `public/` — minified HTML/CSS/JS, responsive images,
a search index, sitemap, and RSS feed. Deploy that folder anywhere.

## Project layout

```text
mysite/
├── nemi.toml      # site configuration
├── content/        # Markdown — one file per page
├── layouts/        # Go html/template files (the theme)
├── static/         # assets copied as-is
└── data/           # structured data (YAML/TOML/JSON)
```

A file at `content/blog/my-post.md` becomes the page `/blog/my-post/`.
Subdirectories automatically get list pages.

## Write a post

```bash
nemi new post "My First Post"
```

This creates a dated draft in `content/blog/`. Drafts are excluded from builds
until you remove `draft: true` (or build with `--drafts`).
