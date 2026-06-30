<div align="center">

# spelunk-md

**A chronic habit of descending where no one asked.**

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![OpenRouter](https://img.shields.io/badge/OpenRouter-powered-7C3AED?style=flat-square&logo=openai&logoColor=white)](https://openrouter.ai)
[![CI](https://img.shields.io/github/actions/workflow/status/elev1e1n/spelunk-md/ci.yml?branch=main&style=flat-square)](https://github.com/elev1e1n/spelunk-md/actions)
[![License MIT](https://img.shields.io/github/license/elev1e1n/spelunk-md?style=flat-square)](LICENSE)
[![Release](https://img.shields.io/github/v/release/elev1e1n/spelunk-md?style=flat-square&color=22c55e)](https://github.com/elev1e1n/spelunk-md/releases)

</div>

---

Descends, sniffs out the manifests, leafs through commits, comes back up with a `.md` thing.

## Quick start

\`\`\`bash
go install github.com/elev1e1n/spelunk-md@latest
spelunk-md --api-key sk-or-xxxxxxxxxxxxxxxx
cd /path/to/your/project
spelunk-md
\`\`\`

> [!TIP]
> No need to save the key — `OPENROUTER_API_KEY` works too. The spelunker isn't picky.

## What happens while you wait

A file tree (respecting `.gitignore`), manifests (`go.mod`, `package.json`, `Cargo.toml`), key configs, the last 30 commits, a README if you're lucky. All of it goes into one prompt, the prompt goes to OpenRouter, out comes a map file.

It won't touch an existing file without asking. If manners get in the way — `--force`.

> [!NOTE]
> `deepseek/deepseek-v4-flash` runs by default — cheap, fast. For a more deliberate result — `--model anthropic/claude-sonnet-4.6` or `--model google/gemini-2.5-pro`.

## Examples

\`\`\`bash
spelunk-md --path ~/projects/my-app
spelunk-md --model anthropic/claude-opus-4
spelunk-md --dry-run
spelunk-md --output docs/PROJECT.md --force
spelunk-md --api-key clear
\`\`\`

Full flag list: `spelunk-md --help`.

## Install

\`\`\`bash
go install github.com/elev1e1n/spelunk-md@latest
go install github.com/elev1e1n/spelunk-md@v1.0.0
\`\`\`

Or from source:

\`\`\`bash
git clone https://github.com/elev1e1n/spelunk-md
cd spelunk-md
just build
\`\`\`

> [!IMPORTANT]
> The key lives in the system keyring — Keychain, Credential Manager, or Secret Service. It never appears in code or project files.

## License

[MIT](LICENSE)
