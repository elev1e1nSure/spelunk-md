# CLAUDE.md

## Project Overview
`spelunk-md` is a Go CLI tool that scans a codebase and generates a `CLAUDE.md` file using AI via OpenRouter. It analyzes file tree (respecting `.gitignore`), tech stack, configuration files, Git history (last 30 commits), and README. The generated output helps AI assistants understand project context.

**Repository**: https://github.com/elev1e1nSure/spelunk-md.git

## Tech Stack
- **Language**: Go 1.26.3
- **CLI Framework**: Cobra (`github.com/spf13/cobra`)
- **Keyring**: `github.com/zalando/go-keyring` (macOS Keychain / Windows Credential Manager / Linux Secret Service)
- **System**: `golang.org/x/sys` for ANSI support
- **Build Tool**: `just` (command runner)

## Project Structure
```
.
├── main.go                  # Entry point
├── go.mod / go.sum          # Module dependencies
├── justfile                 # Dev commands (build, check, tidy, etc.)
├── plan.md                  # Project planning notes
├── README.md                # User documentation
└── src/                     # (implied from imports)
    ├── config.go            # CLI flags & configuration
    ├── claude.go            # Core logic for generating CLAUDE.md
    ├── builder.go           # Builds analysis output
    ├── files.go             # File tree scanning (respects .gitignore)
    ├── git.go               # Git history retrieval
    ├── stack.go             # Technology stack detection
    ├── ui.go                # Terminal UI (colored output, spinner)
    ├── ansi_other.go        # ANSI escape codes (non-Windows)
    └── ansi_windows.go      # ANSI escape codes (Windows)
```

## Development Commands
All commands are run via `just` or plain Go.

| Command | Description |
|---------|-----------|
| `just build` | Compile binary (`spelunk-md` on Linux/macOS, `spelunk-md.exe` on Windows) |
| `just install` | `go install .` – install to `$GOPATH/bin` |
| `just check` | Run `go vet ./...` + `go build ./...` |
| `just tidy` | Run `go mod tidy` to clean dependencies |
| `just dry` | Run `go run . --dry-run` on current directory |
| `just clean` | Remove the built binary |
| `just` (default) | Show available commands with colored output |

**Go commands directly**:
- `go build -o spelunk-md .`
- `go run .` (with optional flags like `--path`, `--model`, `--dry-run`)
- `go test ./...` (if tests are added)

## Architecture & Key Decisions
- **Cobra CLI**: All flags (`--api-key`, `--model`, `--path`, `--output`, `--dry-run`) defined in `config.go`.
- **Keyring for API key**: The OpenRouter API key is stored securely in the system keyring (not in files or env vars). Use `--api-key clear` to delete.
- **Default model**: `deepseek/deepseek-v4-flash` – fast, cheap, code-aware. Override via `--model`.
- **Output file**: Defaults to `CLAUDE.md` in the target project.
- **Analysis scope**:
  - File tree with `.gitignore` filtering.
  - Tech stack (extensions, `go.mod`, `package.json`, `Cargo.toml`, etc.).
  - Key config files (`Makefile`, `Dockerfile`, `tsconfig.json`, etc.).
  - Last 30 Git commits.
  - README.md if present.
- **UI package**: Provides colored output, spinner, ANSI support with platform-specific implementations (`ansi_other.go` / `ansi_windows.go`).
- **No tests currently** – consider adding them.

## Code Conventions
- **Language**: Go (idiomatic style, use `go fmt`).
- **Naming**: CamelCase for exported functions, lowerCamelCase for unexported. Package-level variables/flags in `config.go`.
- **Error handling**: Return errors; Cobra handles usage display.
- **Imports**: Group standard library, third-party, internal.
- **Comments**: Document public functions; use `// Package` doc comments.
- **UI output**: Consistent colored formatting via `ui` package (no raw terminal escapes outside that package).
- **Build**: Binary named `spelunk-md` (lowercase with hyphen).

## Important Notes / Gotchas
1. **API key is never stored in code or env** – it goes to OS keyring. On first run use `--api-key <your_key>`.
2. **`.gitignore` must include `CLAUDE.md`** – otherwise the generated file will be scanned in subsequent runs (the project's own `.gitignore` already excludes it).
3. **Dry-run mode** (`--dry-run`) prints the prompt sent to AI without making an API call – useful for debugging or prompt review.
4. **Module name** in `go.mod` is `github.com/elev1e1n/spelunk-md` – use this for import paths.
5. **Windows terminal support** – ANSI detection is handled separately (`ansi_windows.go`); may require an ANSI-aware terminal (Windows Terminal, ConEmu, etc.).
6. **Git history** – requires a Git repository; if none exists, the tool may skip Git analysis.
7. **Model selection** – any OpenRouter-compatible model; default is fast/cheap. For higher quality but slower responses, use `--model anthropic/claude-opus-4` or similar.
8. **Output file overwrite** – `CLAUDE.md` will be overwritten on each run; no backup is created.
9. **No tests yet** – use `go vet` and manual dry-runs to validate changes.
10. **Dependencies** – `go-keyring` requires D-Bus on Linux, wincred on Windows, Keychain on macOS. Ensure system services are running.