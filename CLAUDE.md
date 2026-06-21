# CLAUDE.md

## Project Overview

**spelunk-md** is a Go CLI tool that scans a codebase, analyzes its structure, tech stack, config files, git history, and README, then generates a `CLAUDE.md` file via OpenRouter AI (default model: `deepseek/deepseek-v4-flash`). API key is stored securely in the system keyring (macOS Keychain, Windows Credential Manager, Linux Secret Service). Built with [Cobra](https://github.com/spf13/cobra) for CLI commands and [go-keyring](https://github.com/zalando/go-keyring) for credential management.

## Tech Stack

- **Language:** Go 1.26.3  
- **CLI Framework:** Cobra (v1.10.2)  
- **Keyring:** `zalando/go-keyring` (v0.2.8)  
- **System libs:** `golang.org/x/sys` (v0.27.0)  
- **Task runner:** Just (see `justfile`)  
- **Recommended model (OpenRouter):** `deepseek/deepseek-v4-flash`

## Project Structure

```
spelunk-md/
├── main.go               # Entry point
├── config.go             # Configuration handling (flags, keyring)
├── claude.go             # AI prompt generation and API call logic
├── builder.go            # File tree / project structure builder
├── files.go              # File scanning helpers
├── git.go                # Git history extraction
├── stack.go              # Tech stack detection
├── ui.go                 # Terminal UI (spinner, colors, etc.)
├── ansi_other.go         # ANSI support (non-Windows)
├── ansi_windows.go       # ANSI support (Windows)
├── go.mod / go.sum       # Module dependencies
├── justfile              # Dev task runner
├── README.md             # User documentation
├── LICENSE
└── .gitignore
```

## Development Commands

All commands are run from the project root via [Just](https://github.com/casey/just).

| Command        | Description                                    |
|----------------|------------------------------------------------|
| `just`         | Show available commands (default)              |
| `just build`   | Compile binary (`spelunk-md` or `spelunk-md.exe`) |
| `just install` | `go install` to `$GOPATH/bin`                  |
| `just check`   | `go vet ./...` + `go build ./...`              |
| `just tidy`    | `go mod tidy`                                  |
| `just dry`     | Dry run on current directory (no API call)     |
| `just clean`   | Remove built binary                            |

Other useful commands:
- `go build -o spelunk-md .` — manual build
- `go run . --dry-run` — dry run without just
- `go vet ./...` — static analysis
- `go mod tidy` — clean dependencies

## Architecture & Key Decisions

- **API key security:** Key is stored in the OS keyring (via `go-keyring`) — never in files or environment variables. Use `--api-key` to save; `--api-key clear` to remove.
- **Prompt construction:** The tool reads the project tree (respecting `.gitignore`), detects tech stack from extension maps and config files (`go.mod`, `package.json`, `Cargo.toml`, etc.), extracts last 30 git commits, and includes `README.md` content. This is assembled into a prompt sent to OpenRouter.
- **Output:** Writes a `CLAUDE.md` file (customizable via `--output`).
- **Cross-platform ANSI:** Two files (`ansi_other.go` / `ansi_windows.go`) provide terminal color and spinner support on all platforms.
- **No tests yet** — no test files or test infrastructure observed.

## Code Conventions

- **Language:** Go, standard formatting (`gofmt`).
- **Error handling:** Idiomatic Go (return errors, check them).
- **Flag style:** Cobra persistent flags, `--kebab-case` naming.
- **Imports:** Grouped: stdlib → third-party → internal.
- **Comments:** Go-style doc comments for exported functions.
- **No separate `cmd/` package** — CLI logic is in top-level files (e.g., `config.go`, `claude.go`).
- **UI output:** Uses the internal `ui` package for consistent colored output (see `justfile` recipes, `ui.go`).
- **Module path:** `github.com/elev1e1n/spelunk-md`

## Important Notes / Gotchas

- **First run requires API key:** `spelunk-md --api-key sk-or-...` — key persisted in OS keyring.
- **OpenRouter model:** Default is `deepseek/deepseek-v4-flash`. Change with `--model` (e.g., `--model anthropic/claude-opus-4`).
- **Dry-run is safe:** `--dry-run` prints the prompt without making an API call — useful for debugging.
- **.gitignore is respected:** File scanning honors `.gitignore` entries.
- **Windows ANSI:** Separate ANSI handling file for Windows compatibility.
- **Binary naming:** `just build` produces `spelunk-md` (Linux/macOS) or `spelunk-md.exe` (Windows).
- **No test coverage:** Project currently lacks automated tests — run `go vet` before committing.
- **Go version:** Requires Go 1.26.3 (check `go.mod`). Ensure your toolchain matches.
- **Module name changed:** Recent commits renamed the project from `spelunk` to `spelunk-md`. Ensure all imports and builds reference `github.com/elev1e1n/spelunk-md`.