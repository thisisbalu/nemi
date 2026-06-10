# Nemi Site

Nemi static site: Markdown + Go templates → plain HTML.

## Structure

- `nemi.toml` — TOML config: title, baseURL, description
- `content/` — Markdown with YAML frontmatter; subdirs auto-generate list pages
- `layouts/` — Go html/template: base.html (outer shell), single.html, list.html
- `static/` — assets copied as-is; referenced as `/filename.ext`
- `public/` — generated output; never edit; gitignored

## Frontmatter fields

`title` · `date` · `draft` · `tags` · `description`

URL: `content/blog/my-post.md` → `/blog/my-post/`

## Commands

```bash
nemi build    # build → public/
nemi serve    # dev server localhost:3000 + live reload
```

## Rules

- Never edit `public/` — overwritten on every build
- `.Page.Content` is `template.HTML` — use `{{.Page.Content}}` directly

Full reference: **AGENTS.md**
