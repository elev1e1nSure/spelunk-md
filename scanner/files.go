package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	gitignore "github.com/sabhiram/go-gitignore"
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
	".ico": true, ".woff": true, ".woff2": true,
	".ttf": true, ".eot": true, ".pdf": true, ".zip": true,
	".tar": true, ".gz": true, ".exe": true, ".bin": true,
	".so": true, ".dylib": true, ".dll": true, ".lock": true,
}

// dirIgnore pairs a directory prefix (slash-relative to root) with its compiled .gitignore.
type dirIgnore struct {
	dir string
	gi  *gitignore.GitIgnore
}

// attrPattern stores a single .gitattributes rule: the pattern and whether it marks
// files as binary, generated, or export-ignored.
type attrPattern struct {
	pattern  string
	skipFile bool
}

// attrIgnore collects all skip-worthy .gitattributes patterns across the tree.
type attrIgnore struct {
	dir      string
	patterns []attrPattern
}

// ScanFiles walks the project root and returns a FileTree.
func ScanFiles(root string) (*FileTree, error) {
	ignores := collectIgnores(root)
	attrs := collectAttributes(root)

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
		relSlash := filepath.ToSlash(rel)

		if info.IsDir() {
			if defaultIgnoreDirs[name] {
				return filepath.SkipDir
			}
			if matchesAnyIgnore(ignores, relSlash) {
				return filepath.SkipDir
			}
			return nil
		}

		if defaultIgnoreFiles[name] || binaryExtensions[filepath.Ext(name)] {
			return nil
		}
		if matchesAnyIgnore(ignores, relSlash) {
			return nil
		}
		if matchesAnyAttr(attrs, relSlash) {
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

// collectIgnores walks the root collecting all .gitignore files (root + nested).
func collectIgnores(root string) []dirIgnore {
	var result []dirIgnore
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return nil
		}
		if info.Name() == ".git" {
			return filepath.SkipDir
		}
		gi, err := gitignore.CompileIgnoreFile(filepath.Join(path, ".gitignore"))
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if rel == "." {
			rel = ""
		}
		result = append(result, dirIgnore{dir: rel, gi: gi})
		return nil
	})
	return result
}

// matchesAnyIgnore checks if a slash-relative path is covered by any collected .gitignore.
// Nested .gitignore files are matched against paths relative to their own directory.
func matchesAnyIgnore(ignores []dirIgnore, relSlash string) bool {
	for _, di := range ignores {
		var toMatch string
		if di.dir == "" {
			toMatch = relSlash
		} else if strings.HasPrefix(relSlash, di.dir+"/") {
			toMatch = relSlash[len(di.dir)+1:]
		} else {
			continue
		}
		if di.gi.MatchesPath(toMatch) {
			return true
		}
	}
	return false
}

// collectAttributes walks the root collecting all .gitattributes files and extracts
// patterns marked as binary, linguist-generated, or export-ignore.
func collectAttributes(root string) []attrIgnore {
	var result []attrIgnore
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return nil
		}
		if info.Name() == ".git" {
			return filepath.SkipDir
		}
		attrPath := filepath.Join(path, ".gitattributes")
		data, err := os.ReadFile(attrPath)
		if err != nil {
			return nil
		}
		var patterns []attrPattern
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.Fields(line)
			if len(parts) < 2 {
				continue
			}
			pat := parts[0]
			for _, attr := range parts[1:] {
				attr = strings.TrimSpace(attr)
				if attr == "binary" || attr == "-diff" || attr == "linguist-generated" ||
					attr == "linguist-generated=true" || attr == "export-ignore" {
					patterns = append(patterns, attrPattern{pattern: pat, skipFile: true})
					break
				}
			}
		}
		if len(patterns) == 0 {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if rel == "." {
			rel = ""
		}
		result = append(result, attrIgnore{dir: rel, patterns: patterns})
		return nil
	})
	return result
}

// matchesAnyAttr checks if a slash-relative path matches any .gitattributes skip rule.
func matchesAnyAttr(attrs []attrIgnore, relSlash string) bool {
	for _, ai := range attrs {
		var toMatch string
		if ai.dir == "" {
			toMatch = relSlash
		} else if strings.HasPrefix(relSlash, ai.dir+"/") {
			toMatch = relSlash[len(ai.dir)+1:]
		} else {
			continue
		}
		for _, ap := range ai.patterns {
			if !ap.skipFile {
				continue
			}
			matched, _ := filepath.Match(ap.pattern, toMatch)
			if matched {
				return true
			}
			// Also match against just the filename (common .gitattributes convention).
			matched, _ = filepath.Match(ap.pattern, filepath.Base(toMatch))
			if matched {
				return true
			}
		}
	}
	return false
}

type treeNode struct {
	name     string
	isFile   bool
	children []*treeNode
}

// Render returns an ASCII tree representation.
func (ft *FileTree) Render() string {
	if len(ft.Entries) == 0 {
		return "(empty)"
	}

	entries := ft.Entries
	truncated := false
	if len(entries) > 200 {
		entries = entries[:200]
		truncated = true
	}

	root := &treeNode{}
	for _, e := range entries {
		parts := strings.Split(filepath.ToSlash(e), "/")
		cur := root
		for i, part := range parts {
			if i == len(parts)-1 {
				cur.children = append(cur.children, &treeNode{name: part, isFile: true})
				break
			}
			var found *treeNode
			for _, c := range cur.children {
				if !c.isFile && c.name == part {
					found = c
					break
				}
			}
			if found == nil {
				found = &treeNode{name: part}
				cur.children = append(cur.children, found)
			}
			cur = found
		}
	}
	sortTree(root)

	var sb strings.Builder
	renderNode(root, "", true, &sb)
	if truncated {
		fmt.Fprintf(&sb, "... (%d more files)\n", len(ft.Entries)-200)
	}
	return sb.String()
}

func sortTree(n *treeNode) {
	sort.Slice(n.children, func(i, j int) bool {
		if n.children[i].isFile != n.children[j].isFile {
			return !n.children[i].isFile // directories first
		}
		return n.children[i].name < n.children[j].name
	})
	for _, c := range n.children {
		sortTree(c)
	}
}

func renderNode(n *treeNode, prefix string, isLast bool, sb *strings.Builder) {
	if n.name != "" {
		connector := "├── "
		if isLast {
			connector = "└── "
		}
		sb.WriteString(prefix)
		sb.WriteString(connector)
		sb.WriteString(n.name)
		if !n.isFile {
			sb.WriteString("/")
		}
		sb.WriteString("\n")
	}
	for i, c := range n.children {
		childIsLast := i == len(n.children)-1
		childPrefix := prefix
		if n.name != "" {
			if isLast {
				childPrefix += "    "
			} else {
				childPrefix += "│   "
			}
		}
		renderNode(c, childPrefix, childIsLast, sb)
	}
}
