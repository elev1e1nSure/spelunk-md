package scanner

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// ProjectMeta holds facts extracted deterministically — no model needed.
type ProjectMeta struct {
	RuntimeVersion string
	License        string
	HasTests       bool
	CI             []string
	Authors        []string
	BuildTargets   []string
	KeyDeps        []string
	UserContext    string // content of .spelunk/context.md if present
}

func ScanMeta(root string, entries []string, stack *Stack) *ProjectMeta {
	m := &ProjectMeta{}
	m.RuntimeVersion = stack.RuntimeVersion
	m.License = detectLicense(root)
	m.HasTests = detectTests(entries)
	m.CI = detectCI(entries)
	m.Authors = gitAuthors(root)
	m.BuildTargets = extractBuildTargets(stack)
	m.KeyDeps = extractDeps(stack)
	m.UserContext = readUserContext(root)
	return m
}

func detectRuntime(root string, stack *Stack) string {
	if raw, ok := stack.ConfigFiles["go.mod"]; ok {
		re := regexp.MustCompile(`(?m)^go\s+(\S+)`)
		if m := re.FindStringSubmatch(raw); len(m) > 1 {
			return "Go " + m[1]
		}
	}
	for _, f := range []string{".node-version", ".nvmrc"} {
		data, err := os.ReadFile(filepath.Join(root, f))
		if err == nil {
			return "Node " + strings.TrimSpace(strings.TrimPrefix(string(data), "v"))
		}
	}
	if raw, ok := stack.ConfigFiles["package.json"]; ok {
		var pkg struct {
			Engines struct {
				Node string `json:"node"`
			} `json:"engines"`
		}
		if json.Unmarshal([]byte(raw), &pkg) == nil && pkg.Engines.Node != "" {
			return "Node " + pkg.Engines.Node
		}
	}
	if data, err := os.ReadFile(filepath.Join(root, ".python-version")); err == nil {
		return "Python " + strings.TrimSpace(string(data))
	}
	if data, err := os.ReadFile(filepath.Join(root, "rust-toolchain.toml")); err == nil {
		re := regexp.MustCompile(`channel\s*=\s*"([^"]+)"`)
		if m := re.FindStringSubmatch(string(data)); len(m) > 1 {
			return "Rust " + m[1]
		}
	}
	return ""
}

func detectLicense(root string) string {
	for _, name := range []string{"LICENSE", "LICENSE.md", "LICENSE.txt"} {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(strings.NewReader(string(data)))
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" {
				continue
			}
			switch {
			case strings.Contains(line, "MIT"):
				return "MIT"
			case strings.Contains(line, "Apache"):
				return "Apache-2.0"
			case strings.Contains(line, "GPL-3"), strings.Contains(line, "GNU General Public License, version 3"):
				return "GPL-3.0"
			case strings.Contains(line, "GPL"):
				return "GPL-2.0"
			case strings.Contains(line, "BSD"):
				return "BSD"
			case strings.Contains(line, "ISC"):
				return "ISC"
			case strings.Contains(line, "MPL"):
				return "MPL-2.0"
			}
			if len(line) > 50 {
				return line[:50]
			}
			return line
		}
	}
	return ""
}

func detectTests(entries []string) bool {
	for _, e := range entries {
		name := filepath.Base(e)
		dir := filepath.Base(filepath.Dir(e))
		if strings.HasSuffix(name, "_test.go") ||
			strings.Contains(name, ".test.") ||
			strings.Contains(name, ".spec.") ||
			strings.HasPrefix(name, "test_") ||
			dir == "tests" || dir == "test" || dir == "__tests__" {
			return true
		}
	}
	return false
}

func detectCI(entries []string) []string {
	var result []string
	seen := map[string]bool{}
	add := func(s string) {
		if !seen[s] {
			result = append(result, s)
			seen[s] = true
		}
	}
	for _, e := range entries {
		slash := filepath.ToSlash(e)
		name := filepath.Base(e)
		switch {
		case strings.HasPrefix(slash, ".github/workflows/"):
			add("GitHub Actions")
		case name == ".gitlab-ci.yml":
			add("GitLab CI")
		case name == "Jenkinsfile":
			add("Jenkins")
		case name == ".travis.yml":
			add("Travis CI")
		case strings.HasPrefix(slash, ".circleci/"):
			add("CircleCI")
		case name == "azure-pipelines.yml":
			add("Azure Pipelines")
		}
	}
	return result
}

func gitAuthors(root string) []string {
	out, err := exec.Command("git", "-C", root, "shortlog", "-sn", "--no-merges").Output()
	if err != nil {
		return nil
	}
	var authors []string
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		parts := strings.SplitN(strings.TrimSpace(sc.Text()), "\t", 2)
		if len(parts) == 2 {
			authors = append(authors, fmt.Sprintf("%s (%s)", strings.TrimSpace(parts[1]), strings.TrimSpace(parts[0])))
		}
		if len(authors) >= 5 {
			break
		}
	}
	return authors
}

var justLineRe = regexp.MustCompile(`(?m)^([a-z][a-z0-9_-]*)`)

func extractBuildTargets(stack *Stack) []string {
	if raw, ok := stack.ConfigFiles["justfile"]; ok {
		var result []string
		seen := map[string]bool{}
		for _, line := range strings.SplitAfter(raw, "\n") {
			m := justLineRe.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			name := m[1]
			rest := strings.TrimPrefix(line, name)
			rest = strings.TrimSpace(rest)
			// must be a recipe line (has `:` but not `:=`)
			if !strings.HasPrefix(rest, ":") || strings.HasPrefix(rest, ":=") {
				continue
			}
			if name == "set" || name == "export" || seen[name] {
				continue
			}
			seen[name] = true
			result = append(result, name)
		}
		return result
	}
	if raw, ok := stack.ConfigFiles["Makefile"]; ok {
		re := regexp.MustCompile(`(?m)^([a-zA-Z][a-zA-Z0-9_-]*)\s*:(?!=)`)
		var result []string
		seen := map[string]bool{}
		for _, m := range re.FindAllStringSubmatch(raw, -1) {
			name := m[1]
			if seen[name] || name == "PHONY" {
				continue
			}
			seen[name] = true
			result = append(result, name)
		}
		return result
	}
	if raw, ok := stack.ConfigFiles["package.json"]; ok {
		var pkg struct {
			Scripts map[string]string `json:"scripts"`
		}
		clean := strings.TrimSuffix(raw, "\n... (truncated)")
		if json.Unmarshal([]byte(clean), &pkg) == nil {
			var targets []string
			for k := range pkg.Scripts {
				targets = append(targets, k)
			}
			sort.Strings(targets)
			return targets
		}
	}
	return nil
}

func extractDeps(stack *Stack) []string {
	var deps []string
	if raw, ok := stack.ConfigFiles["go.mod"]; ok {
		deps = append(deps, parseGoMod(raw)...)
	}
	if raw, ok := stack.ConfigFiles["package.json"]; ok {
		deps = append(deps, parsePackageJSON(raw)...)
	}
	if raw, ok := stack.ConfigFiles["Cargo.toml"]; ok {
		deps = append(deps, parseCargo(raw)...)
	}
	if raw, ok := stack.ConfigFiles["requirements.txt"]; ok {
		deps = append(deps, parseRequirements(raw)...)
	}
	if len(deps) > 20 {
		deps = deps[:20]
	}
	return deps
}

func parseGoMod(raw string) []string {
	var deps []string
	inRequire := false
	sc := bufio.NewScanner(strings.NewReader(raw))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "require (") || line == "require (" {
			inRequire = true
			continue
		}
		if inRequire && line == ")" {
			inRequire = false
			continue
		}
		if !inRequire && strings.HasPrefix(line, "require ") {
			line = strings.TrimPrefix(line, "require ")
			line = strings.TrimPrefix(line, "(")
		}
		if (inRequire || strings.HasPrefix(line, "github.com/") || strings.HasPrefix(line, "golang.org/")) &&
			!strings.Contains(line, "// indirect") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				deps = append(deps, shortenMod(parts[0])+" "+parts[1])
			}
		}
	}
	return deps
}

func shortenMod(mod string) string {
	segs := strings.Split(mod, "/")
	if len(segs) >= 2 {
		return strings.Join(segs[len(segs)-2:], "/")
	}
	return mod
}

func parsePackageJSON(raw string) []string {
	var pkg struct {
		Dependencies map[string]string `json:"dependencies"`
	}
	if json.Unmarshal([]byte(strings.TrimSuffix(raw, "\n... (truncated)")), &pkg) != nil {
		return nil
	}
	var deps []string
	for k, v := range pkg.Dependencies {
		deps = append(deps, k+" "+v)
	}
	sort.Strings(deps)
	return deps
}

func parseCargo(raw string) []string {
	var deps []string
	inDeps := false
	re := regexp.MustCompile(`^([a-z][a-z0-9_-]*)\s*=`)
	sc := bufio.NewScanner(strings.NewReader(raw))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "[dependencies]" {
			inDeps = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			inDeps = false
			continue
		}
		if inDeps {
			if m := re.FindStringSubmatch(line); m != nil {
				deps = append(deps, m[1])
			}
		}
	}
	return deps
}

func parseRequirements(raw string) []string {
	var deps []string
	sc := bufio.NewScanner(strings.NewReader(raw))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		deps = append(deps, line)
	}
	return deps
}

func readUserContext(root string) string {
	data, err := os.ReadFile(filepath.Join(root, ".spelunk", "context.md"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
