package scanner

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const maxSigBytes = 6000

// Signatures holds extracted public symbol signatures for the dominant language.
type Signatures struct {
	Lines []string
	Lang  string
}

func ScanSignatures(root string, entries []string, stack *Stack) *Signatures {
	if len(stack.Languages) == 0 {
		return nil
	}
	lang := stack.Languages[0]
	var lines []string
	switch lang {
	case "Go":
		lines = goSignatures(root, entries)
	case "TypeScript", "JavaScript":
		lines = tsSignatures(root, entries)
	case "Python":
		lines = pySignatures(root, entries)
	default:
		return nil
	}
	if len(lines) == 0 {
		return nil
	}
	return &Signatures{Lines: lines, Lang: lang}
}

func goSignatures(root string, entries []string) []string {
	var result []string
	total := 0
	for _, e := range entries {
		if !strings.HasSuffix(e, ".go") || strings.HasSuffix(e, "_test.go") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, e))
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		var fileSigs []string
		for i := 0; i < len(lines); i++ {
			line := strings.TrimSpace(lines[i])
			if line == "" {
				continue
			}

			// Export groups: const ( ... ) / var ( ... )
			if strings.HasPrefix(line, "const (") || strings.HasPrefix(line, "var (") {
				kind := "const"
				if strings.HasPrefix(line, "var (") {
					kind = "var"
				}
				var names []string
				for j := i + 1; j < len(lines); j++ {
					inner := strings.TrimSpace(lines[j])
					if inner == ")" {
						break
					}
					if inner == "" || strings.HasPrefix(inner, "//") {
						continue
					}
					parts := strings.Fields(inner)
					if len(parts) > 0 && isExportedName(parts[0]) {
						names = append(names, parts[0])
					}
				}
				if len(names) > 0 {
					fileSigs = append(fileSigs, fmt.Sprintf("%s (%s)", kind, strings.Join(names, ", ")))
				}
				continue
			}

			if !isGoExported(line) {
				continue
			}

			sig := line
			// Include struct fields for exported struct types.
			if strings.HasPrefix(line, "type ") && strings.HasSuffix(line, " struct {") {
				var fields []string
				for j := i + 1; j < len(lines); j++ {
					f := strings.TrimSpace(lines[j])
					if f == "}" {
						break
					}
					if f == "" || strings.HasPrefix(f, "//") {
						continue
					}
					fields = append(fields, f)
				}
				if len(fields) > 0 {
					sig = line + "\n    " + strings.Join(fields, "\n    ")
				}
			}

			// Collect preceding godoc comments.
			var doc []string
			for j := i - 1; j >= 0; j-- {
				prev := strings.TrimSpace(lines[j])
				if strings.HasPrefix(prev, "// ") {
					doc = append([]string{strings.TrimPrefix(prev, "// ")}, doc...)
				} else if prev == "" {
					break
				} else {
					break
				}
			}

			var entry strings.Builder
			for _, d := range doc {
				entry.WriteString("// " + d + "\n")
			}
			entry.WriteString(sig)
			s := entry.String()
			if len(s) > 240 {
				s = s[:240] + "..."
			}
			fileSigs = append(fileSigs, s)
		}
		if len(fileSigs) == 0 {
			continue
		}
		header := "// " + filepath.ToSlash(e)
		result = append(result, header)
		total += len(header)
		if total > maxSigBytes {
			return result
		}
		for _, s := range fileSigs {
			result = append(result, s)
			total += len(s)
			if total > maxSigBytes {
				return result
			}
		}
	}
	return result
}

func isExportedName(name string) bool {
	return len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z'
}

func isGoExported(line string) bool {
	for _, prefix := range []string{"func ", "type ", "var ", "const "} {
		if strings.HasPrefix(line, prefix) {
			rest := line[len(prefix):]
			// Handle method receivers: func (r Recv) Name(
			if prefix == "func " && strings.HasPrefix(rest, "(") {
				// find method name after receiver
				end := strings.Index(rest, ")")
				if end > 0 && end+1 < len(rest) {
					after := strings.TrimSpace(rest[end+1:])
					if len(after) > 0 && after[0] >= 'A' && after[0] <= 'Z' {
						return true
					}
				}
				return false
			}
			if len(rest) > 0 && rest[0] >= 'A' && rest[0] <= 'Z' {
				return true
			}
		}
	}
	return false
}

func tsSignatures(root string, entries []string) []string {
	keywords := []string{
		"export function ", "export async function ", "export class ",
		"export interface ", "export type ", "export const ", "export enum ",
		"export abstract class ", "export default function", "export default class",
	}
	var result []string
	total := 0
	for _, e := range entries {
		ext := filepath.Ext(e)
		if ext != ".ts" && ext != ".tsx" && ext != ".js" && ext != ".jsx" {
			continue
		}
		if strings.Contains(e, ".test.") || strings.Contains(e, ".spec.") {
			continue
		}
		f, err := os.Open(filepath.Join(root, e))
		if err != nil {
			continue
		}
		var fileSigs []string
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			for _, kw := range keywords {
				if strings.HasPrefix(line, kw) {
					if len(line) > 120 {
						line = line[:120] + "..."
					}
					fileSigs = append(fileSigs, line)
					break
				}
			}
		}
		f.Close()
		if len(fileSigs) == 0 {
			continue
		}
		header := "// " + filepath.ToSlash(e)
		result = append(result, header)
		total += len(header)
		if total > maxSigBytes {
			return result
		}
		for _, line := range fileSigs {
			result = append(result, line)
			total += len(line)
			if total > maxSigBytes {
				return result
			}
		}
	}
	return result
}

func pySignatures(root string, entries []string) []string {
	var result []string
	total := 0
	for _, e := range entries {
		if !strings.HasSuffix(e, ".py") || strings.HasPrefix(filepath.Base(e), "test_") {
			continue
		}
		f, err := os.Open(filepath.Join(root, e))
		if err != nil {
			continue
		}
		var fileSigs []string
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := sc.Text()
			trimmed := strings.TrimLeft(line, " \t")
			if strings.HasPrefix(trimmed, "def ") || strings.HasPrefix(trimmed, "class ") || strings.HasPrefix(trimmed, "async def ") {
				sig := strings.TrimSpace(trimmed)
				if len(sig) > 120 {
					sig = sig[:120] + "..."
				}
				fileSigs = append(fileSigs, sig)
			}
		}
		f.Close()
		if len(fileSigs) == 0 {
			continue
		}
		header := "# " + filepath.ToSlash(e)
		result = append(result, header)
		total += len(header)
		if total > maxSigBytes {
			return result
		}
		for _, sig := range fileSigs {
			result = append(result, sig)
			total += len(sig)
			if total > maxSigBytes {
				return result
			}
		}
	}
	return result
}
