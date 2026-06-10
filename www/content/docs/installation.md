---
title: "Installation"
date: 2026-02-06
description: "Install Nemi via Homebrew, the install script, go install, or a prebuilt binary."
---

Nemi is a single binary with no runtime dependencies. Pick whichever method
suits you.

## Homebrew (macOS / Linux)

```bash
brew install thisisbalu/tap/nemi
```

## Install script (macOS / Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/thisisbalu/nemi/main/install.sh | sh
```

Set `INSTALL_DIR` to choose where the binary lands (defaults to
`/usr/local/bin`).

## go install

```bash
go install github.com/thisisbalu/nemi@latest
```

## Prebuilt binaries

Download a binary for macOS, Linux, or Windows from the
[Releases](https://github.com/thisisbalu/nemi/releases) page and put it on your
`PATH`.

## Verify

```bash
nemi version
```
