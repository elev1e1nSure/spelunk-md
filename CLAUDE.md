## Project Overview

`spelunk-md` (CLI tool `spelunk-md`) scans a project directory, gathers structure, tech stack, configuration files, git history, and README, then sends a structured prompt to an OpenRouter AI model to generate a `CLAUDE.md` file. Designed for simplicity — one command, no setup.

## Tech Stack

- **Language:** Go 1.26.3
- **CLI Framework:** Cobra (`github.com/spf13/cobra`)
- **Key Dependencies:**
  - `github.com/zalando/go-keyring` — secure API key storage
  - `github.com/sabhiram/go-gitignore` — `.gitignore` parsing
  - `golang.org/x/sys` — platform-specific system calls
  - `github.com/spf13/pflag` — flag parsing (Cobra dependency)
- **Build Tool:** `just` (via `justfile`)

## Project Structure

```
.
├── main.go              # Entry point
├── claude.go            # Top-level CLI command logic
├── config.go            # Configuration handling
├── builder.go           # Prompt building
├── files.go             # File traversal and scanning
├── git.go               # Git history retrieval
├── stack.go             # Tech stack detection
├── ui.go                # Terminal UI (spinner, colors)
├── ansi_other.go        # ANSI support (non-Windows)
├── ansi_windows.go      # ANSI support (Windows)
├── go.mod / go.sum
├── justfile             # Dev commands
├── README.md
├── LICENSE
└── .gitignore
```

## Development Commands

All commands are run via `just` (or directly with `go`). Binary name: `spelunk-md` (or `spelunk-md.exe` on Windows).

| Command        | Description                                      |
|----------------|--------------------------------------------------|
| `just build`   | Compile binary with version info                 |
| `just install` | `go install` to `$GOPATH/bin`                    |
| `just check`   | `go vet ./...` + `go build ./...`                |
| `just tidy`    | `go mod tidy` to clean dependencies              |
| `just dry`     | Dry-run `go run . --dry-run` on current dir      |
| `just clean`   | Remove built binary                              |

Direct Go commands:

- Run in dev: `go run . [flags]`
- Build: `go build -ldflags "-X main.version=$(git describe --tags --always --dirty)" -o spelunk-md .`

## Architecture & Key Decisions

- **Single binary** with Cobra CLI. Currently uses a single root command; subcommands may be added later.
- **OpenRouter API** for LLM generation (default model `deepseek/deepseek-v4-flash`). Model can be overridden with `--model`.
- **API key** stored in OS keyring via `go-keyring`, fallback to environment variable `OPENROUTER_API_KEY` (useful for CI/WSL).
- **Git-aware scanning:** respects `.gitignore` (including nested), retrieves last 30 commits.
- **Tech stack detection** by file extensions and manifest files (`go.mod`, `package.json`, `Cargo.toml`).
- **Output:** Writes `CLAUDE.md` to project root; prompts before overwrite (disable with `--force`).
- **Version injection** via `ldflags` at build time (package `main` variable `version`).

## Code Conventions

- **Go style:** standard `gofmt`, use `go vet ./...` before committing.
- **Error handling:** explicit error returns; use `log.Fatal` sparingly (prefer returning errors to caller).
- **Package layout:** flat package (all Go files in root) — no internal subpackages yet.
- **UI output:** use `ui.go` functions for colored/spinner output; platform-specific ANSI in `ansi_*.go`.
- **Environment variables:** `OPENROUTER_API_KEY` for API key fallback.
- **Testing:** no test files observed yet; add tests under `*_test.go` following Go conventions.

## Important Notes / Gotchas

- **Binary naming:** output binary is `spelunk-md` (with `.exe` on Windows). The module path is `github.com/elev1e1n/spelunk-md`.
- **API key security:** `go-keyring` may require additional setup on headless systems or WSL. Use `OPENROUTER_API_KEY` env var as fallback.
- **Git history:** requires a git repository; fails gracefully if not present.
- **CLAUDE.md overwrite:** prompts for confirmation unless `--force` is provided.
- **Timeout:** configurable via `--timeout` flag (default unknown, check `--help`).
- **Justfile uses `sh -cu` shell** for strict mode — ensure commands are POSIX-compatible.
- **Platform-specific ANSI:** `ansi_windows.go` handles Windows console; `ansi_other.go` for Unix/macOS.