# Nemi Site

This is a Nemi static site (Markdown + Go templates → plain HTML).

## Essentials

```
nemi.toml      # site config (TOML): title, baseURL, description
content/        # .md files with YAML frontmatter → pages
layouts/        # Go html/template: base.html, single.html, list.html
static/         # assets copied as-is to output
public/         # generated output — never edit, gitignored
```

Frontmatter: `title`, `date`, `draft`, `tags`, `description`
URL mapping: `content/blog/my-post.md` → `/blog/my-post/`
Subdirectories auto-generate list pages.

## Commands

```bash
nemi new <name>   # scaffold new site
nemi build        # build → public/
nemi serve        # dev server localhost:3000 + live reload
```

## Key rules

- Never edit `public/` — overwritten on every build
- `static/` assets referenced as `/filename.ext`
- `.Page.Content` is `template.HTML` — use `{{.Page.Content}}` directly

Full reference: **AGENTS.md**
