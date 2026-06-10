# Nemi

A fast, zero-config static site generator that builds beautiful sites out of the box.

Nemi compiles Markdown + Go HTML templates into plain, fast HTML. It ships with a
polished default theme (auto dark mode, sticky table of contents, syntax
highlighting, math, diagrams) and the things other generators make you wire up
yourself — sitemap, RSS, tags, pagination, and a content linter — all on by
default.

## Install

### Homebrew (macOS / Linux)

```bash
brew install thisisbalu/tap/nemi
```

### Install script (macOS / Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/thisisbalu/nemi/main/install.sh | sh
```

### go install

```bash
go install github.com/thisisbalu/nemi@latest
```

### Binaries

Prebuilt binaries for macOS, Linux, and Windows are on the
[Releases](https://github.com/thisisbalu/nemi/releases) page.

## Quickstart

```bash
nemi new mysite        # scaffold a new site
cd mysite
nemi serve             # dev server on http://localhost:3000 with live reload
nemi build             # compile content/ → public/
```

## Commands

| Command | Description |
|---|---|
| `nemi new <name>` | Scaffold a new site |
| `nemi new post "Title" [-s section]` | Scaffold a draft post |
| `nemi build [-D]` | Build `content/` → `public/` (`-D` includes drafts) |
| `nemi serve [-D]` | Dev server with live reload |
| `nemi check [-D]` | Lint content for broken links, duplicate URLs, missing titles |
| `nemi migrate hugo <src> [dest]` | Convert a Hugo site |
| `nemi migrate jekyll <src> [dest]` | Convert a Jekyll site |
| `nemi version` | Print the version |

## What you get by default

- **Links that can't break** — reference content by path (`[x](@/blog/post.md)`);
  Nemi resolves it to the final URL and **fails the build** if the target is gone.
- **Beautiful default theme** — minimal, typographic, one consistent width, auto
  dark mode + toggle, theme-aware code blocks, scroll progress, sticky TOC.
- **Rich content** — Chroma syntax highlighting, KaTeX math (`math: true`),
  Mermaid diagrams (```` ```mermaid ````), GFM, footnotes, smart typography.
- **Data-driven pages** — résumé/CV from `data/cv.yaml`; optional academic
  publications list from `data/publications.yaml`.
- **SEO out of the box** — sitemap, RSS feed, canonical URLs, robots.txt.
- **Tags, pagination, partials, menus, data files** — with zero configuration.

## Building from source

```bash
git clone https://github.com/thisisbalu/nemi
cd nemi
go build -o nemi .
go test ./...
```

## License

[MIT](LICENSE)
