---
title: "CLI commands"
date: 2026-02-04
description: "Reference for every Nemi command and flag."
---

## new

```bash
nemi new <name>                  # scaffold a new site into ./<name>
nemi new post "Title" [-s sec]   # scaffold a draft post (default section: blog)
```

## build

```bash
nemi build [-D]
```

Compiles `content/` into `public/`. Pass `--drafts` / `-D` to include pages
marked `draft: true`. Prints the page count and build time.

## serve

```bash
nemi serve [-D]
```

Builds the site and starts a dev server on `:3000` with live reload. Output is
left unminified and served at the root for fast, readable rebuilds.

## check

```bash
nemi check [-D]
```

A content linter — reports broken internal links and assets, duplicate URLs,
and missing titles. Exits non-zero on any error, so it fits in CI.

## migrate

```bash
nemi migrate hugo <src> [dest]     # convert a Hugo site
nemi migrate jekyll <src> [dest]   # convert a Jekyll site
```

Converts an existing site to Nemi format. `dest` defaults to `<src>-nemi`.
See [Migrating](/docs/) for details.

## version

```bash
nemi version
```
