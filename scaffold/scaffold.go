package scaffold

import "embed"

//go:embed nemi.toml content all:layouts static data .gitignore AGENTS.md CLAUDE.md GEMINI.md .cursorrules all:.github
var FS embed.FS
