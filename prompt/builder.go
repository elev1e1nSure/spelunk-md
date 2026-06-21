package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/elev1e1n/spelunk-md/scanner"
)

const maxPromptBytes = 80_000

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

	sb.WriteString(`You are an expert software engineer generating a CLAUDE.md file.

CLAUDE.md is read by Claude Code at the start of every session. It should help an AI assistant understand:
- What the project is and does
- Tech stack and key dependencies
- How to build, run, and test it
- Architecture decisions and conventions
- Gotchas and non-obvious constraints

Output ONLY the CLAUDE.md content. No preamble, no explanation, no code fences around the whole document.
Use ## headings. Be specific — generic boilerplate is useless.

---

`)

	// User context — written by maintainer, authoritative.
	if ctx.Meta != nil && ctx.Meta.UserContext != "" {
		sb.WriteString("## USER CONTEXT\n")
		sb.WriteString("> Written by the project maintainer. Treat as authoritative.\n\n")
		sb.WriteString(ctx.Meta.UserContext)
		sb.WriteString("\n\n")
	}

	// Structured project facts.
	sb.WriteString(fmt.Sprintf("## PROJECT\nName: %s\n", ctx.ProjectName))
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

	if ctx.Stack != nil {
		if len(ctx.Stack.Languages) > 0 {
			sb.WriteString(fmt.Sprintf("## LANGUAGES\n%s\n\n", strings.Join(ctx.Stack.Languages, ", ")))
		}
		if len(ctx.Stack.Frameworks) > 0 {
			sb.WriteString(fmt.Sprintf("## FRAMEWORKS & LIBRARIES\n%s\n\n", strings.Join(ctx.Stack.Frameworks, ", ")))
		}
		if len(ctx.Stack.Tools) > 0 {
			sb.WriteString(fmt.Sprintf("## TOOLS\n%s\n\n", strings.Join(ctx.Stack.Tools, ", ")))
		}
	}

	if ctx.Meta != nil && len(ctx.Meta.BuildTargets) > 0 {
		sb.WriteString(fmt.Sprintf("## BUILD COMMANDS\n%s\n\n", strings.Join(ctx.Meta.BuildTargets, ", ")))
	}

	if ctx.Meta != nil && len(ctx.Meta.KeyDeps) > 0 {
		sb.WriteString("## KEY DEPENDENCIES\n")
		for _, d := range ctx.Meta.KeyDeps {
			sb.WriteString("- " + d + "\n")
		}
		sb.WriteString("\n")
	}

	deps := scanner.ScanPackageDeps(ctx.Tree.Root, ctx.Tree.Entries)
	if len(deps) > 0 {
		sb.WriteString("## PACKAGE DEPENDENCIES\n")
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

	sb.WriteString("## FILE STRUCTURE\n```\n")
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
		sb.WriteString("## KEY SOURCE FILES\n")
		for _, block := range sourceBlocks {
			sb.WriteString(block)
		}
		sb.WriteString("\n")
	}

	if ctx.Sigs != nil && len(ctx.Sigs.Lines) > 0 {
		sb.WriteString(fmt.Sprintf("## CODE SIGNATURES (%s)\n```\n", ctx.Sigs.Lang))
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
				sb.WriteString("## KEY CONFIGURATION FILES\n\n")
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
				sb.WriteString("## KEY CONFIGURATION FILES\n\n")
				firstHeader = false
			}
			fmt.Fprintf(&sb, "### %s\n```\n%s\n```\n\n", name, content)
		}
	}

	if ctx.Git.IsRepo && ctx.Git.RecentCommits != "" {
		sb.WriteString("## RECENT GIT HISTORY (last 30 commits)\n```\n")
		sb.WriteString(ctx.Git.RecentCommits)
		sb.WriteString("\n```\n\n")
	}

	sb.WriteString(`---

Generate the CLAUDE.md now. Required sections (skip if not applicable):

## Project Overview
## Tech Stack
## Project Structure
## Development Commands
## Architecture & Key Decisions
## Code Conventions
## Important Notes / Gotchas

Use BUILD COMMANDS and KEY DEPENDENCIES verbatim where relevant — don't invent commands.
Code Conventions should reflect what's actually visible in the code, not generic language defaults.

Before outputting, verify the following and fix any mismatches:
- Project Structure must match FILE STRUCTURE exactly (same directories, same files, no invented paths).
- Do not claim all code is in package main if FILE STRUCTURE shows packages like config/, generator/, prompt/, scanner/, ui/.
- Mention only commands that exist in BUILD COMMANDS.
- Mention only dependencies that exist in KEY DEPENDENCIES or KEY SOURCE FILES.
- Architecture & Key Decisions must be supported by CODE SIGNATURES and KEY SOURCE FILES, not guessed.
`)

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
