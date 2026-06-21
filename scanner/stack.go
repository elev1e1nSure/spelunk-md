package scanner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Stack holds detected technologies and key config files content.
type Stack struct {
	Languages   []string
	Frameworks  []string
	Tools       []string
	ConfigFiles map[string]string // filename -> content (trimmed)
}

type packageJSON struct {
	Name         string            `json:"name"`
	Dependencies map[string]string `json:"dependencies"`
	DevDeps      map[string]string `json:"devDependencies"`
	Scripts      map[string]string `json:"scripts"`
}

// DetectStack inspects the project root and file list.
func DetectStack(root string, entries []string) *Stack {
	s := &Stack{
		ConfigFiles: make(map[string]string),
	}

	extCounts := map[string]int{}
	for _, e := range entries {
		ext := strings.ToLower(filepath.Ext(e))
		if ext != "" {
			extCounts[ext]++
		}
	}

	// Language detection: include if dominant (>=2 files) or only detected language.
	langMap := map[string]string{
		".go":   "Go",
		".rs":   "Rust",
		".ts":   "TypeScript",
		".tsx":  "TypeScript",
		".js":   "JavaScript",
		".jsx":  "JavaScript",
		".py":   "Python",
		".java": "Java",
		".kt":   "Kotlin",
		".rb":   "Ruby",
		".cs":   "C#",
		".cpp":  "C++",
		".c":    "C",
		".zig":  "Zig",
		".dart": "Dart",
		".lua":  "Lua",
	}

	type langCount struct {
		name  string
		count int
	}
	seen := map[string]int{} // lang -> total file count
	for ext, lang := range langMap {
		if c := extCounts[ext]; c > 0 {
			seen[lang] += c
		}
	}

	var ranked []langCount
	for lang, count := range seen {
		ranked = append(ranked, langCount{lang, count})
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].count > ranked[j].count })

	for i, lc := range ranked {
		// Always include the dominant language; others need >=2 files.
		if i == 0 || lc.count >= 2 {
			s.Languages = append(s.Languages, lc.name)
		}
	}

	// Read key config files. Exclude lock files — they add tokens with no LLM value.
	configTargets := []string{
		"go.mod",
		"package.json",
		"Cargo.toml",
		"pyproject.toml", "requirements.txt", "setup.py",
		"Dockerfile", "docker-compose.yml", "docker-compose.yaml",
		"Makefile", "justfile",
		".env.example",
		"tsconfig.json",
		"vite.config.ts", "vite.config.js",
		"tailwind.config.ts", "tailwind.config.js",
		"README.md",
	}

	for _, target := range configTargets {
		fullPath := filepath.Join(root, target)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}
		content := string(data)
		if len(content) > 3000 {
			content = content[:3000] + "\n... (truncated)"
		}
		s.ConfigFiles[target] = content
	}

	// Framework/tool detection from package.json
	if raw, ok := s.ConfigFiles["package.json"]; ok {
		var pkg packageJSON
		if err := json.Unmarshal([]byte(strings.TrimSuffix(raw, "\n... (truncated)")), &pkg); err == nil {
			allDeps := map[string]string{}
			for k, v := range pkg.Dependencies {
				allDeps[k] = v
			}
			for k, v := range pkg.DevDeps {
				allDeps[k] = v
			}

			frameworkMap := map[string]string{
				"next":                  "Next.js",
				"react":                 "React",
				"vue":                   "Vue",
				"svelte":                "Svelte",
				"@angular/core":         "Angular",
				"express":               "Express",
				"fastify":               "Fastify",
				"hono":                  "Hono",
				"tailwindcss":           "Tailwind CSS",
				"prisma":                "Prisma",
				"drizzle-orm":           "Drizzle ORM",
				"@supabase/supabase-js": "Supabase",
				"electron":              "Electron",
			}
			seenFW := map[string]bool{}
			for dep := range allDeps {
				if fw, ok := frameworkMap[dep]; ok && !seenFW[fw] {
					s.Frameworks = append(s.Frameworks, fw)
					seenFW[fw] = true
				}
			}
		}
	}

	// Go framework detection from go.mod
	if raw, ok := s.ConfigFiles["go.mod"]; ok {
		goFrameworks := map[string]string{
			"github.com/gin-gonic/gin":           "Gin",
			"github.com/gofiber/fiber":            "Fiber",
			"github.com/charmbracelet/bubbletea": "Bubbletea",
			"github.com/spf13/cobra":              "Cobra",
			"github.com/labstack/echo":            "Echo",
			"gorm.io/gorm":                        "GORM",
			"github.com/tauri-apps":               "Tauri",
		}
		for pkg, fw := range goFrameworks {
			if strings.Contains(raw, pkg) {
				s.Frameworks = append(s.Frameworks, fw)
			}
		}
	}

	// Tool detection — use seen map to prevent duplicates.
	seenTools := map[string]bool{}
	addTool := func(name string) {
		if !seenTools[name] {
			s.Tools = append(s.Tools, name)
			seenTools[name] = true
		}
	}

	toolChecks := []struct{ file, tool string }{
		{"Dockerfile", "Docker"},
		{"docker-compose.yml", "Docker Compose"},
		{"docker-compose.yaml", "Docker Compose"},
		{"Makefile", "Make"},
		{"justfile", "Just"},
	}
	for _, tc := range toolChecks {
		if _, exists := s.ConfigFiles[tc.file]; exists {
			addTool(tc.tool)
		}
	}

	// GitHub Actions: directory-based detection, separate from ConfigFiles check.
	for _, e := range entries {
		if strings.HasPrefix(filepath.ToSlash(e), ".github/workflows/") {
			addTool("GitHub Actions")
			break
		}
	}

	return s
}
