---
title: "Deploying"
date: 2026-01-27
description: "Build for production and deploy to GitHub Pages, a subpath, a custom domain, or any static host."
---

`nemi build` compiles your site into `public/` — minified HTML/CSS/JS,
responsive images, a search index, sitemap, and RSS. Deploy that folder
anywhere that serves static files.

## Set your base URL

Before a production build, set `baseURL` in `nemi.toml` to where the site will
live. It's used for canonical URLs, the sitemap, the RSS feed, and social-card
links:

```toml
baseURL = "https://example.com/"
```

## Deploying under a subpath

If your `baseURL` includes a path — common on GitHub **project** pages, e.g.
`https://you.github.io/your-repo/` — Nemi automatically rewrites every internal
link, asset, and the search index to sit under that prefix. No template changes,
no flags. `nemi serve` always runs at the root, so local development is
unaffected.

## GitHub Pages

This very site is built with Nemi and deployed to Pages. A minimal workflow:

```yaml
# .github/workflows/docs.yml
name: deploy
on:
  push: { branches: [main] }
permissions:
  contents: read
  pages: write
  id-token: write
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version-file: go.mod }
      - run: go run github.com/thisisbalu/nemi@latest build
      - uses: actions/configure-pages@v5
      - uses: actions/upload-pages-artifact@v3
        with: { path: public }
  deploy:
    needs: build
    runs-on: ubuntu-latest
    environment: { name: github-pages, url: "${{ steps.d.outputs.page_url }}" }
    steps:
      - id: d
        uses: actions/deploy-pages@v4
```

Enable Pages once under **Settings → Pages → Source: GitHub Actions**.

## Custom domains

Add a `CNAME` file to `static/` containing your domain, and point your DNS at
GitHub Pages. The site is then served at the domain's root (or under the repo
subpath for project pages) — set `baseURL` to match.

## Other hosts

For Netlify, Cloudflare Pages, Vercel, and friends, use:

- **Build command:** `nemi build`
- **Publish directory:** `public`

Any static host works — there's no server runtime to provision.
