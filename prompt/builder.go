package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/elev1e1n/spelunk-md/scanner"
)

const maxPromptBytes = 320_000 // ~80k tokens

// manifests parsed into Meta — skip raw dump in prompt.
var skipRaw = map[string]bool{
	"go.mod": true, "package.json": true, "Cargo.toml": true,
	"pyproject.toml": true, "requirements.txt": true, "setup.py": true,
}

// Context holds all collected project data.
type Context struct {
	ProjectName string
	Tree        *scanner.FileTree
	Stack       *scanner.Stack
	Git         *scanner.GitInfo
	Meta        *scanner.ProjectMeta
	Sigs        *scanner.Signatures
}

// Build constructs the full prompt for the LLM.
func Build(ctx *Context) string {
	trimConfigFiles(ctx)

	var sb strings.Builder

	// ── 1. DATA AT THE TOP ───────────────────────────────────────────────
	// Long inputs go above the instruction: the model keeps the instruction
	// fresher when it sits last. Everything here is raw material, tagged by
	// how the model is allowed to use it.
	sb.WriteString("<codebase_scan>\n")
	sb.WriteString("<!-- RAW MATERIAL for writing CLAUDE.md. Section tags:\n")
	sb.WriteString("     [authoritative] — maintainer's own words. Must be reflected in the\n")
	sb.WriteString("        output; preserve meaning, rephrase minimally, never drop.\n")
	sb.WriteString("     [usable] — may surface directly in the output (e.g. commands).\n")
	sb.WriteString("     [reference-only] — infer from this, but do NOT copy verbatim or\n")
	sb.WriteString("        reproduce as a list (e.g. dependency graph, language inventory). -->\n\n")

	// User context — written by maintainer, highest priority.
	if ctx.Meta != nil && ctx.Meta.UserContext != "" {
		sb.WriteString("## USER CONTEXT [authoritative]\n")
		sb.WriteString("> Written by the project maintainer. Highest priority.\n\n")
		sb.WriteString(ctx.Meta.UserContext)
		sb.WriteString("\n\n")
	}

	// Structured project facts.
	sb.WriteString(fmt.Sprintf("## PROJECT [usable]\nName: %s\n", ctx.ProjectName))
	if ctx.Git.IsRepo && ctx.Git.RemoteURL != "" {
		sb.WriteString(fmt.Sprintf("Repository: %s\n", ctx.Git.RemoteURL))
	}
	if ctx.Meta != nil {
		if ctx.Meta.RuntimeVersion != "" {
			sb.WriteString(fmt.Sprintf("Runtime: %s\n", ctx.Meta.RuntimeVersion))
		}
		if ctx.Meta.License != "" {
			sb.WriteString(fmt.Sprintf("License: %s\n", ctx.Meta.License))
		}
		if ctx.Meta.HasTests {
			sb.WriteString("Tests: yes\n")
		} else {
			sb.WriteString("Tests: none detected\n")
		}
		if len(ctx.Meta.CI) > 0 {
			sb.WriteString(fmt.Sprintf("CI: %s\n", strings.Join(ctx.Meta.CI, ", ")))
		}
		if len(ctx.Meta.Authors) > 0 {
			sb.WriteString(fmt.Sprintf("Authors: %s\n", strings.Join(ctx.Meta.Authors, ", ")))
		}
	}
	sb.WriteString("\n")

	// Stack inventory — model reads this from config files itself, so it's
	// reference-only: used to understand the project, never echoed as a list.
	if ctx.Stack != nil {
		if len(ctx.Stack.Languages) > 0 {
			sb.WriteString(fmt.Sprintf("## LANGUAGES [reference-only]\n%s\n\n", strings.Join(ctx.Stack.Languages, ", ")))
		}
		if len(ctx.Stack.Frameworks) > 0 {
			sb.WriteString(fmt.Sprintf("## FRAMEWORKS & LIBRARIES [reference-only]\n%s\n\n", strings.Join(ctx.Stack.Frameworks, ", ")))
		}
		if len(ctx.Stack.Tools) > 0 {
			sb.WriteString(fmt.Sprintf("## TOOLS [reference-only]\n%s\n\n", strings.Join(ctx.Stack.Tools, ", ")))
		}
	}

	// Build commands — core of CLAUDE.md, may surface directly.
	if ctx.Meta != nil && len(ctx.Meta.BuildTargets) > 0 {
		sb.WriteString(fmt.Sprintf("## BUILD COMMANDS [usable]\n%s\n\n", strings.Join(ctx.Meta.BuildTargets, ", ")))
	}

	// Key deps — inventory, reference-only.
	if ctx.Meta != nil && len(ctx.Meta.KeyDeps) > 0 {
		sb.WriteString("## KEY DEPENDENCIES [reference-only]\n")
		for _, d := range ctx.Meta.KeyDeps {
			sb.WriteString("- " + d + "\n")
		}
		sb.WriteString("\n")
	}

	// Internal import structure — the model infers architecture and module
	// boundaries from this. Renamed from "package dependencies" so the model
	// reads it as structure, not as a list to reproduce.
	deps := scanner.ScanPackageDeps(ctx.Tree.Root, ctx.Tree.Entries)
	if len(deps) > 0 {
		sb.WriteString("## MODULE GRAPH (internal import structure) [reference-only]\n")
		var pkgs []string
		for pkg := range deps {
			pkgs = append(pkgs, pkg)
		}
		sort.Strings(pkgs)
		for _, pkg := range pkgs {
			if len(deps[pkg]) == 0 {
				continue
			}
			sb.WriteString(pkg + " → " + strings.Join(deps[pkg], ", ") + "\n")
		}
		sb.WriteString("\n")
	}

	// File tree — infer layout from it, don't reprint it.
	sb.WriteString("## FILE STRUCTURE [reference-only]\n```\n")
	sb.WriteString(ctx.Tree.Render())
	sb.WriteString("```\n\n")

	// Full source of the most architecturally important files.
	keySources := []string{"main.go", "prompt/builder.go"}
	var sourceBlocks []string
	for _, src := range keySources {
		content, err := os.ReadFile(filepath.Join(ctx.Tree.Root, src))
		if err != nil {
			continue
		}
		sourceBlocks = append(sourceBlocks, fmt.Sprintf("### %s\n```go\n%s\n```\n", src, string(content)))
	}
	if len(sourceBlocks) > 0 {
		sb.WriteString("## KEY SOURCE FILES [reference-only]\n")
		for _, block := range sourceBlocks {
			sb.WriteString(block)
		}
		sb.WriteString("\n")
	}

	if ctx.Sigs != nil && len(ctx.Sigs.Lines) > 0 {
		sb.WriteString(fmt.Sprintf("## CODE SIGNATURES (%s) [reference-only]\n```\n", ctx.Sigs.Lang))
		for _, line := range ctx.Sigs.Lines {
			sb.WriteString(line + "\n")
		}
		sb.WriteString("```\n\n")
	}

	// Raw config files — manifests skipped, already parsed into Meta.
	if ctx.Stack != nil && len(ctx.Stack.ConfigFiles) > 0 {
		priority := []string{
			"Makefile", "justfile", "Dockerfile", "docker-compose.yml",
			"tsconfig.json", "vite.config.ts", "vite.config.js",
			"tailwind.config.ts", "tailwind.config.js", ".env.example", "README.md",
		}
		printed := map[string]bool{}
		firstHeader := true
		for _, name := range priority {
			if skipRaw[name] {
				continue
			}
			content, ok := ctx.Stack.ConfigFiles[name]
			if !ok {
				continue
			}
			if firstHeader {
				sb.WriteString("## KEY CONFIGURATION FILES [reference-only]\n\n")
				firstHeader = false
			}
			fmt.Fprintf(&sb, "### %s\n```\n%s\n```\n\n", name, content)
			printed[name] = true
		}
		for name, content := range ctx.Stack.ConfigFiles {
			if printed[name] || skipRaw[name] {
				continue
			}
			if firstHeader {
				sb.WriteString("## KEY CONFIGURATION FILES [reference-only]\n\n")
				firstHeader = false
			}
			fmt.Fprintf(&sb, "### %s\n```\n%s\n```\n\n", name, content)
		}
	}

	if ctx.Git.IsRepo && ctx.Git.RecentCommits != "" {
		sb.WriteString("## RECENT GIT HISTORY (last 30 commits) [reference-only]\n```\n")
		sb.WriteString(ctx.Git.RecentCommits)
		sb.WriteString("\n```\n\n")
	}

	sb.WriteString("</codebase_scan>\n\n---\n\n")

	// ── 2. INSTRUCTION AT THE BOTTOM ─────────────────────────────────────
	sb.WriteString(`You are an expert software engineer writing a CLAUDE.md file for the codebase in <codebase_scan> above.

CLAUDE.md is injected into the AI assistant's context at the start of EVERY session and costs tokens on every turn, so it must be dense and high-signal. The test for every line: does it change how the assistant works in this repo, OR is it something the assistant would discover on its own within one session of reading the code? If the latter, leave it out.

The <codebase_scan> is tagged by section:
- [authoritative]: reflect this in the output. It is the maintainer's intent — preserve its meaning, rephrase minimally, never drop it.
- [usable]: may appear directly in your output.
- [reference-only]: use it to understand the project, but never copy it verbatim and never reproduce it as a list. Languages, frameworks, and dependencies are reference-only — the assistant reads those from config files itself, so listing them wastes context. Instead, use the module graph and file tree to describe architecture and layout in your own words.

Include:
- What the project is and does — 2-3 bullets, no more
- Exact build / run / test / lint commands
- Project layout: path → what lives there, for the few directories that matter
- Conventions, constraints, and gotchas — especially the non-obvious ones a new engineer would get wrong

Do NOT:
- List languages, frameworks, tools, or dependencies (all reference-only — the assistant reads these itself)
- Duplicate the README — reference it with @README.md instead
- Pad with generic advice that applies to any project ("write clean code", "follow best practices")

How to write it:
- Imperatives, not observations. "Use X. Never Y." — not "the project generally uses X."
- Every line must be verifiable: a command, a path, or a concrete rule. If a line can't be checked against the repo, cut it.
- Negative rules carry as much weight as positive ones. State what NOT to do where the obvious default is wrong.
- Keep the whole file under ~150 lines. Denser and shorter beats long and complete.

Here is the shape of a good CLAUDE.md (study the density, not the content):

<example>
# quill-commit

TUI git monitor that auto-commits on a timer. Go + Bubbletea.
Details in @README.md.

## Commands
- Build: ` + "`go build -o quill ./cmd/quill`" + `
- Test: ` + "`go test ./...`" + `
- Lint: ` + "`golangci-lint run`" + `

## Layout
- ` + "`cmd/quill/`" + ` — entrypoint, flag parsing
- ` + "`internal/watcher/`" + ` — fs polling + debounce logic
- ` + "`internal/llm/`" + ` — OpenRouter client, commit-message generation

## Conventions
- Commit messages follow Conventional Commits. The LLM prompt enforces this — don't loosen it.
- Never push. This tool commits only; pushing is the user's job.
- API keys come from the OS keyring via ` + "`internal/secret`" + `. Never read keys from env or flags.
- Bubbletea models stay pure: no I/O in Update(), side effects go through tea.Cmd.
</example>

Output ONLY the CLAUDE.md content — no preamble, no explanation, no code fences around the whole document. Use ## headings.`)
	return sb.String()
}

func trimConfigFiles(ctx *Context) {
	if ctx.Stack == nil {
		return
	}
	dropOrder := []string{
		"docker-compose.yaml", "docker-compose.yml", ".env.example",
		"vite.config.ts", "vite.config.js",
		"tailwind.config.ts", "tailwind.config.js",
		"setup.py", "requirements.txt",
		"README.md", "Dockerfile", "tsconfig.json",
	}
	for estimatedSize(ctx) > maxPromptBytes && len(dropOrder) > 0 {
		delete(ctx.Stack.ConfigFiles, dropOrder[0])
		dropOrder = dropOrder[1:]
	}
}

func estimatedSize(ctx *Context) int {
	total := 3000
	total += len(ctx.Tree.Render())
	if ctx.Git.IsRepo {
		total += len(ctx.Git.RecentCommits)
	}
	for _, content := range ctx.Stack.ConfigFiles {
		total += len(content) + 50
	}
	if ctx.Sigs != nil {
		for _, l := range ctx.Sigs.Lines {
			total += len(l) + 1
		}
	}
	total += keySourceSize(ctx.Tree.Root)
	total += depsSize(ctx.Tree.Root, ctx.Tree.Entries)
	return total
}

func depsSize(root string, entries []string) int {
	deps := scanner.ScanPackageDeps(root, entries)
	total := 0
	for pkg, list := range deps {
		total += len(pkg) + 4
		for _, d := range list {
			total += len(d) + 2
		}
	}
	return total
}

func keySourceSize(root string) int {
	total := 0
	for _, src := range []string{"main.go", "prompt/builder.go"} {
		info, err := os.Stat(filepath.Join(root, src))
		if err == nil {
			total += int(info.Size()) + len(src) + 20 // wrapper overhead
		}
	}
	return total
}

// BuildIncremental constructs the prompt for updating an existing file based on changed files.
func BuildIncremental(ctx *Context, currentContent string, changedFiles []string) string {
	var sb strings.Builder

	sb.WriteString(`You are an expert software engineer updating an existing AI context file (CLAUDE.md / .cursorrules / etc.).

Your task is to update the file content based on the changes in the codebase.
Only modify or add sections that are affected by the changes (e.g. tech stack, key dependencies, build commands, architecture, or conventions).
Do not rewrite unrelated sections. Keep the existing formatting and style intact where possible.

Output ONLY the complete, updated file content. No preamble, no explanation, no code fences around the whole document.

---

`)

	sb.WriteString("## CURRENT CONTENT\n")
	sb.WriteString(currentContent)
	sb.WriteString("\n\n---\n\n")

	sb.WriteString("## CHANGES DETECTED\n")
	sb.WriteString("The following files have changed since the last snapshot:\n\n")

	for _, f := range changedFiles {
		sb.WriteString(fmt.Sprintf("### File: %s\n", f))

		// Check if the file exists
		fullPath := filepath.Join(ctx.Tree.Root, filepath.FromSlash(f))
		info, err := os.Stat(fullPath)
		if os.IsNotExist(err) {
			sb.WriteString("Status: Deleted / Removed\n\n")
			continue
		}

		if info.IsDir() {
			sb.WriteString("Status: Directory modified\n\n")
			continue
		}

		// Check if it's a config file we read
		if ctx.Stack != nil {
			if content, ok := ctx.Stack.ConfigFiles[f]; ok {
				sb.WriteString("New Content:\n```\n")
				sb.WriteString(content)
				sb.WriteString("\n```\n\n")
				continue
			}
		}

		// Check if it's the user context file (.spelunk/context.md)
		if filepath.ToSlash(f) == ".spelunk/context.md" {
			if ctx.Meta != nil && ctx.Meta.UserContext != "" {
				sb.WriteString("New User Context:\n```\n")
				sb.WriteString(ctx.Meta.UserContext)
				sb.WriteString("\n```\n\n")
				continue
			}
		}

		// Otherwise check if we have signatures for it
		if sigs := scanner.ScanSignatures(ctx.Tree.Root, []string{f}, ctx.Stack); sigs != nil && len(sigs.Lines) > 0 {
			sb.WriteString(fmt.Sprintf("New Signatures (%s):\n```\n", sigs.Lang))
			for _, line := range sigs.Lines {
				sb.WriteString(line + "\n")
			}
			sb.WriteString("```\n\n")
			continue
		}

		sb.WriteString("Status: Modified\n\n")
	}

	return sb.String()
}
