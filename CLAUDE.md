# CLAUDE.md

## Project Overview

spelunk-md is a Go CLI tool that scans a codebase and generates a `CLAUDE.md` file via AI (OpenRouter). It analyzes file trees (respecting `.gitignore`), technology stacks, configuration files, git history (last 30 commits), and README.md to produce comprehensive project documentation for AI coding assistants.

## Tech Stack

- **Language:** Go 1.26.3
- **CLI Framework:** Cobra (`github.com/spf13/cobra`)
- **Key Dependencies:**
  - `github.com/zalando/go-keyring` — secure API key storage (macOS Keychain, Windows Credential Manager, Linux Secret Service)
  - `golang.org/x/sys` — system-level operations
  - `github.com/danieljoos/wincred` — Windows credential management
  - `github.com/godbus/dbus/v5` — Linux D-Bus integration
- **Build Tool:** Just (command runner)

## Project Structure

```
├── main.go              # Entry point
├── config.go            # Configuration handling
├── claude.go            # CLAUDE.md generation logic
├── builder.go           # Prompt building / file analysis
├── files.go             # File tree scanning
├── git.go               # Git history analysis
├── stack.go             # Technology stack detection
├── ui.go                # Terminal UI (colors, spinner, ANSI)
├── ansi_other.go        # ANSI support (non-Windows)
├── ansi_windows.go      # ANSI support (Windows)
├── go.mod / go.sum      # Module dependencies
├── justfile             # Development task runner
└── README.md            # User documentation
```

## Development Commands

All commands are run via `just`:

| Command | Description |
|---------|-------------|
| `just build` | Compile binary (`spelunk-md` or `spelunk-md.exe`) |
| `just install` | Install to `$GOPATH/bin` |
| `just check` | Run `go vet ./...` + `go build ./...` |
| `just tidy` | Run `go mod tidy` |
| `just dry` | Dry-run on current directory (`go run . --dry-run`) |
| `just clean` | Remove built binary |

Direct Go commands:

- `go build -o spelunk-md .` — manual build
- `go install .` — install to GOPATH
- `go vet ./...` — static analysis
- `go mod tidy` — clean dependencies
- `go run . --dry-run` — preview prompt without API call

## Architecture & Key Decisions

1. **API Key Security:** Keys are stored in the system keyring (not in files or code) via `go-keyring`. Use `--api-key` to set, `--api-key clear` to remove.

2. **Default Model:** `deepseek/deepseek-v4-flash` — fast, cheap, good at code analysis. Override with `--model`.

3. **Analysis Scope:**
   - File tree (respects `.gitignore`)
   - Technology stack detection (extensions, `go.mod`, `package.json`, `Cargo.toml`, etc.)
   - Key config files (`Makefile`, `Dockerfile`, `tsconfig.json`, etc.)
   - Last 30 git commits
   - `README.md` if present

4. **Output:** Generates `CLAUDE.md` (configurable via `--output`).

5. **UI Package:** Custom terminal UI with colored output, spinner animation, and cross-platform ANSI support (separate implementations for Windows and other OS).

6. **Cobra CLI:** All flags are defined via Cobra (`--api-key`, `--model`, `--path`, `--output`, `--dry-run`).

## Code Conventions

- **Language:** Go standard formatting (`gofmt`/`go vet` compliant)
- **Error Handling:** Standard Go error returns; no panics in production paths
- **Imports:** Grouped: stdlib, external, internal (no explicit convention enforced, but follow existing patterns)
- **Naming:** CamelCase for exported, camelCase for unexported
- **Comments:** English comments for exported functions/types; Russian comments in justfile and README
- **File Organization:** One primary concern per file (config, git, files, stack, ui)
- **Platform-specific code:** Use build tags (`ansi_other.go` vs `ansi_windows.go`)

## Important Notes / Gotchas

- **Go 1.26.3 required** — module uses modern Go features; ensure correct version
- **Keyring dependency** — requires system keyring service (may fail in minimal Docker containers or headless environments without dbus)
- **Windows ANSI** — separate implementation in `ansi_windows.go`; ensure Windows builds handle console properly
- **Git history** — requires git to be installed and accessible; fails gracefully if not available
- **`.gitignore` respect** — file scanning automatically excludes patterns from `.gitignore`
- **Binary naming** — `justfile` auto-detects OS for binary name (`spelunk-md.exe` on Windows)
- **No tests yet** — project has no test files; be cautious when refactoring
- **Russian comments** — justfile and README contain Russian text; code comments are in English