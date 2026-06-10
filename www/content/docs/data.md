---
title: "Data files"
date: 2026-01-29
description: "Structured data from the data/ directory, plus the built-in résumé and publications pages."
---

The `data/` directory holds structured data — useful for content that's more
like a table than prose. Files are decoded and exposed to templates under
`.Site.Data`.

## How it works

Nemi walks `data/` and decodes every `.yaml`, `.yml`, `.toml`, and `.json` file,
keyed by its path:

```text
data/social.yaml        → .Site.Data.social
data/team/lead.json     → .Site.Data.team.lead
```

Use it in any template:

```html
<ul>
  {{range .Site.Data.social}}
  <li><a href="{{.url}}">{{.name}}</a></li>
  {{end}}
</ul>
```

Values arrive as plain maps, slices, and scalars, so you can range and index
them naturally.

## Résumé / CV

The default theme renders a structured résumé from `data/cv.yaml` on the
`/resume/` page — experience, education, and skills, plus an optional PDF link:

```yaml
# data/cv.yaml
experience:
  - role: Senior Engineer
    org: Company Name
    period: 2022 – Present
    summary: What you owned and shipped.
    highlights:
      - Led the redesign of X.
education:
  - degree: B.S. Computer Science
    org: University Name
    period: 2015 – 2019
skills:
  - group: Languages
    items: [Go, TypeScript, Python]
```

Edit the data in one place instead of hand-formatting Markdown.

## Publications (opt-in)

For academic sites, `data/publications.yaml` powers a publications page
(grouped by year, with PDF/code/link buttons). It's **off by default** — to
enable it, edit the data and create a `content/publications.md` page (the
`publications` slug resolves to the publications layout automatically), then add
a menu entry pointing to `/publications/`.

```yaml
# data/publications.yaml
- year: 2024
  papers:
    - title: A Title That Describes the Work
      authors: Your Name, A. Coauthor
      venue: Proceedings of Some Conference
      url: https://example.com/paper
      pdf: /papers/2024-title.pdf
```
