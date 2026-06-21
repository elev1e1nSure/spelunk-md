package scanner

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileTree holds the project directory structure.
type FileTree struct {
	Root    string
	Entries []string // relative paths
}

var defaultIgnoreDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true,
	".idea": true, ".vscode": true, "dist": true,
	"build": true, "out": true, "__pycache__": true,
	".next": true, "target": true, ".cache": true,
	"coverage": true, ".nyc_output": true,
}

var defaultIgnoreFiles = map[string]bool{
	".DS_Store": true, "Thumbs.db": true,
}

var binaryExtensions = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	".ico": true, ".svg": true, ".woff": true, ".woff2": true,
	".ttf": true, ".eot": true, ".pdf": true, ".zip": true,
	".tar": true, ".gz": true, ".exe": true, ".bin": true,
	".so": true, ".dylib": true, ".dll": true, ".lock": true,
}

// ScanFiles walks the project root and returns a FileTree.
func ScanFiles(root string) (*FileTree, error) {
	ignorePatterns := loadGitignore(root)

	var entries []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable
		}

		rel, _ := filepath.Rel(root, path)
		if rel == "." {
			return nil
		}

		name := info.Name()

		if info.IsDir() {
			if defaultIgnoreDirs[name] || isGitignored(rel, ignorePatterns) {
				return filepath.SkipDir
			}
			return nil
		}

		if defaultIgnoreFiles[name] || binaryExtensions[filepath.Ext(name)] {
			return nil
		}
		if isGitignored(rel, ignorePatterns) {
			return nil
		}

		entries = append(entries, rel)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk error: %w", err)
	}

	sort.Strings(entries)
	return &FileTree{Root: root, Entries: entries}, nil
}

// Render returns an ASCII tree representation.
func (ft *FileTree) Render() string {
	if len(ft.Entries) == 0 {
		return "(empty)"
	}

	// Cap at 200 entries to avoid token explosion
	entries := ft.Entries
	truncated := false
	if len(entries) > 200 {
		entries = entries[:200]
		truncated = true
	}

	var sb strings.Builder
	for _, e := range entries {
		depth := strings.Count(e, string(os.PathSeparator))
		sb.WriteString(strings.Repeat("  ", depth))
		sb.WriteString("├── ")
		sb.WriteString(filepath.Base(e))
		sb.WriteString("\n")
	}
	if truncated {
		sb.WriteString(fmt.Sprintf("  ... (%d more files)\n", len(ft.Entries)-200))
	}
	return sb.String()
}

func loadGitignore(root string) []string {
	var patterns []string
	path := filepath.Join(root, ".gitignore")
	f, err := os.Open(path)
	if err != nil {
		return patterns
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns
}

func isGitignored(rel string, patterns []string) bool {
	for _, p := range patterns {
		// Simple glob matching
		p = strings.TrimPrefix(p, "/")
		if matched, _ := filepath.Match(p, rel); matched {
			return true
		}
		if matched, _ := filepath.Match(p, filepath.Base(rel)); matched {
			return true
		}
		if strings.HasSuffix(p, "/") {
			if strings.HasPrefix(rel, strings.TrimSuffix(p, "/")) {
				return true
			}
		}
	}
	return false
}
