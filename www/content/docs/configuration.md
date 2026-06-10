---
title: "Configuration"
date: 2026-02-03
description: "Every key in nemi.toml — site metadata, author, menu, images, search, and more."
---

Site configuration lives in `nemi.toml`. Everything has a sensible default, so
a minimal config is just a title and base URL.

## Site

```toml
title       = "My Site"
baseURL     = "https://example.com/"
description = "A short description used for SEO and the RSS feed."
paginate    = 10   # items per list page before /page/2/, etc.
```

If `baseURL` includes a path (e.g. `https://user.github.io/repo/`), Nemi
deploys under that subpath automatically — all internal links are rewritten to
match, so the same config works on GitHub project pages.

## Author

```toml
[author]
name     = "Ada Lovelace"
tagline  = "Engineer · Writer"
bio       = "One or two sentences for the home page."
email     = "ada@example.com"
github   = "ada"
twitter  = "ada"
linkedin = "ada"
```

## Navigation

```toml
[[menu]]
name   = "Blog"
url    = "/blog/"
weight = 1

[[menu]]
name     = "GitHub"
url      = "https://github.com/you"
weight   = 2
external = true   # opens in a new tab
```

Menu items are ordered by `weight`.

## Images, search, and social cards

These are on by default; configure or disable them as needed.

```toml
[images]
widths  = [480, 800, 1200]   # responsive srcset widths
quality = 82
# disable = true

[search]
# disable = true

[og]
# disable = true   # auto-generated social card images
```

## Comments (Giscus)

```toml
[giscus]
repo        = "you/your-repo"
repo_id     = "R_xxxxxxxx"
category    = "Announcements"
category_id = "DIC_xxxxxxxx"
```

Comments render on posts only when both `repo` and `repo_id` are set.
