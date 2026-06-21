package prompt

import (
	"fmt"
	"strings"

	"github.com/elev1e1n/claude-md-gen/scanner"
)

// Context holds all collected project data.
type Context struct {
	ProjectName string
	Tree        *scanner.FileTree
	Stack       *scanner.Stack
	Git         *scanner.GitInfo
}

// Build constructs the full prompt for the LLM.
func Build(ctx *Context) string {
	var sb strings.Builder

	sb.WriteString(`You are an expert software engineer tasked with generating a CLAUDE.md file.

CLAUDE.md is a special file that Claude (an AI coding assistant) reads at the start of every session to understand:
- The project structure and purpose
- The tech stack and key dependencies
- Development commands (build, test, run, lint)
- Architecture decisions and conventions
- Code style preferences
- Important warnings or gotchas

Generate a comprehensive, well-structured CLAUDE.md based on the project analysis below.

The output should be ONLY the CLAUDE.md content — no preamble, no explanation, no markdown code fences.
Use clear headings (##), be concise but complete. Include all commands you can infer from the project files.

---

`)

	// Project identity
	sb.WriteString(fmt.Sprintf("## PROJECT NAME\n%s\n\n", ctx.ProjectName))

	// Remote URL if available
	if ctx.Git.IsRepo && ctx.Git.RemoteURL != "" {
		sb.WriteString(fmt.Sprintf("## REPOSITORY\n%s\n\n", ctx.Git.RemoteURL))
	}

	// Stack
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

	// File tree
	sb.WriteString("## FILE STRUCTURE\n```\n")
	sb.WriteString(ctx.Tree.Render())
	sb.WriteString("```\n\n")

	// Key config files
	if ctx.Stack != nil && len(ctx.Stack.ConfigFiles) > 0 {
		sb.WriteString("## KEY CONFIGURATION FILES\n\n")
		priority := []string{"go.mod", "package.json", "Cargo.toml", "pyproject.toml", "requirements.txt", "Makefile", "justfile", "Dockerfile", "docker-compose.yml", "tsconfig.json", "README.md"}
		printed := map[string]bool{}

		for _, name := range priority {
			content, ok := ctx.Stack.ConfigFiles[name]
			if !ok {
				continue
			}
			sb.WriteString(fmt.Sprintf("### %s\n```\n%s\n```\n\n", name, content))
			printed[name] = true
		}
		for name, content := range ctx.Stack.ConfigFiles {
			if printed[name] {
				continue
			}
			sb.WriteString(fmt.Sprintf("### %s\n```\n%s\n```\n\n", name, content))
		}
	}

	// Git history
	if ctx.Git.IsRepo && ctx.Git.RecentCommits != "" {
		sb.WriteString("## RECENT GIT HISTORY (last 30 commits)\n```\n")
		sb.WriteString(ctx.Git.RecentCommits)
		sb.WriteString("\n```\n\n")
	}

	sb.WriteString(`---

Now generate the CLAUDE.md file. Structure it with these sections (include only what's relevant):

# CLAUDE.md

## Project Overview
## Tech Stack
## Project Structure  
## Development Commands
## Architecture & Key Decisions
## Code Conventions
## Important Notes / Gotchas

Be specific. Infer real commands from Makefile, package.json scripts, go.mod, etc.
If README.md was provided, extract the most relevant dev workflow information from it.
`)

	return sb.String()
}
