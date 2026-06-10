---
title: "Theme & customization"
date: 2026-01-30
description: "Restyle the default theme with CSS custom properties, dark mode, and partial overrides."
---

The default theme is minimal and typographic, and everything visual is driven by
CSS custom properties in `static/style.css`. You rarely need to touch the
markup to make it yours.

## Color & layout tokens

The top of `style.css` defines the palette and measurements:

```css
:root {
  --bg: #ffffff;
  --text: #18181b;
  --muted: #71717a;
  --border: #e4e4e7;
  --code-bg: #f6f8fa;
  --maxw: 60rem;        /* overall site width */
  --prose-maxw: 65ch;   /* reading measure */
}
```

Change these and the whole site follows. The site keeps **one consistent
width** (`--maxw`) across every page.

## Dark mode

Dark mode is automatic and also toggleable. There are three token blocks:

- `:root` — light defaults.
- `@media (prefers-color-scheme: dark)` — follows the OS.
- `:root[data-theme="dark"]` — the manual toggle (the ◐ button), persisted to
  `localStorage`.

A tiny inline script in `head.html` applies the saved choice before first paint,
so there's no flash.

## Fonts & type

The theme uses the system sans stack by default. Swap the `font-family` on
`body` (or add `@font-face` rules and a `<link>` in `head.html`) to use your
own.

## Overriding markup

For anything beyond styling, edit the templates in `layouts/` — see
[Templating](@/docs/templating.md). Common tweaks:

- Header / nav: `layouts/base.html`
- Post layout (TOC, meta, tags): `layouts/single.html`
- A reusable fragment: add a file under `layouts/partials/`

Because the theme lives entirely in your site, every change is yours to make.
