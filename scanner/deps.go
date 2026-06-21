package scanner

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// PackageDeps maps a package path (relative to the module root) to its internal dependencies.
type PackageDeps map[string][]string

// ScanPackageDeps reads all .go files and builds a dependency graph between internal packages.
func ScanPackageDeps(root string, entries []string) PackageDeps {
	module := modulePath(root)
	if module == "" {
		return nil
	}

	deps := make(map[string]map[string]bool)

	for _, e := range entries {
		if !strings.HasSuffix(e, ".go") {
			continue
		}

		pkg := filepath.ToSlash(filepath.Dir(e))
		if pkg == "." {
			pkg = "main"
		}

		f, err := os.Open(filepath.Join(root, e))
		if err != nil {
			continue
		}
		imports := extractImports(f)
		f.Close()

		for _, imp := range imports {
			if !strings.HasPrefix(imp, module) {
				continue
			}
			dep := strings.TrimPrefix(imp, module)
			dep = strings.TrimPrefix(dep, "/")
			if dep == "" {
				dep = "main"
			}
			if dep == pkg {
				continue
			}
			if deps[pkg] == nil {
				deps[pkg] = make(map[string]bool)
			}
			deps[pkg][dep] = true
		}
	}

	result := make(PackageDeps)
	for pkg, set := range deps {
		list := make([]string, 0, len(set))
		for d := range set {
			list = append(list, d)
		}
		sort.Strings(list)
		result[pkg] = list
	}
	return result
}

func modulePath(root string) string {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return ""
	}
	re := regexp.MustCompile(`(?m)^module\s+(\S+)`)
	m := re.FindStringSubmatch(string(data))
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

func extractImports(f *os.File) []string {
	data, err := os.ReadFile(f.Name())
	if err != nil {
		return nil
	}
	content := string(data)

	var imports []string
	inMulti := false

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "import (") {
			inMulti = true
			continue
		}
		if inMulti {
			if trimmed == ")" {
				inMulti = false
				continue
			}
			if imp := parseImportLine(trimmed); imp != "" {
				imports = append(imports, imp)
			}
			continue
		}
		if strings.HasPrefix(trimmed, "import ") {
			if imp := parseImportLine(trimmed[len("import "):]); imp != "" {
				imports = append(imports, imp)
			}
		}
	}
	return imports
}

func parseImportLine(line string) string {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "//") {
		return ""
	}
	// Handle aliased imports: alias "path" or "path"
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return ""
	}
	last := parts[len(parts)-1]
	return strings.Trim(last, `"`)
}
