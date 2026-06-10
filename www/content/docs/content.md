---
title: "Content & frontmatter"
date: 2026-02-02
description: "How pages, sections, frontmatter, and resolved links work."
---

Every page is a Markdown file in `content/`. The file's location determines its
URL: `content/blog/hello.md` → `/blog/hello/`.

## Frontmatter

Pages start with a YAML frontmatter block:

```yaml
---
title: "Hello, World"
date: 2026-01-01
description: "Shown in search results and social cards."
tags: [go, web]
draft: false
---
```

Common keys:

| Key | Purpose |
|---|---|
| `title` | Page title (also the `<h1>`) |
| `date` | Publish date; orders list pages |
| `description` | SEO meta + RSS summary |
| `tags` | Generates `/tags/<tag>/` pages |
| `draft` | Excluded from builds unless `--drafts` |
| `slug` | Override the output URL only |
| `layout` | Force a specific layout |
| `math` | Load KaTeX on this page |

## Sections

A subdirectory of `content/` is a *section*. Add an `_index.md` to give it a
landing page and intro text; otherwise Nemi auto-generates a list page.
Sections paginate automatically past `paginate` items.

## Resolved links

Link to other content by its source path and Nemi resolves it to the final URL
at build time:

```markdown
See [the announcement](@/blog/hello.md) for details.
```

If the target doesn't exist, **the build fails** — so internal links can't
silently rot. Fragments are preserved: `@/blog/hello.md#section`.

## Drafts

Mark a page `draft: true` to keep it out of production builds. Preview drafts
with `nemi serve -D` or `nemi build -D`.
