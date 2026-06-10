---
title: "Home"
description: "A fast, zero-config static site generator that builds beautiful sites out of the box."
---

```bash
brew install thisisbalu/tap/nemi
nemi new mysite && cd mysite
nemi serve
```

Nemi turns Markdown and Go templates into plain, fast HTML. Unlike other
generators, the things you usually have to wire up yourself — a polished theme,
full-text search, responsive images, SEO meta, sitemap, and RSS — are **on by
default**. One binary, no plugins, no config required.

- **Beautiful by default.** A minimal, typographic theme with automatic dark
  mode, syntax highlighting, a sticky table of contents, math, and diagrams.
- **Fast everywhere.** Responsive images and minified output out of the box;
  builds in milliseconds.
- **Links that can't break.** Reference content by path — Nemi resolves it and
  *fails the build* if the target is gone.
- **Search with zero setup.** A built-in client-side search index, no external
  service or extra binary.

[Get started →](/docs/installation/)
