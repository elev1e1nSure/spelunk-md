package scanner

import (
	"bufio"
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
		f, err := os.Open(filepath.Join(root, e))
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if isGoExported(line) {
				if len(line) > 120 {
					line = line[:120] + "..."
				}
				result = append(result, line)
				total += len(line)
				if total > maxSigBytes {
					f.Close()
					return result
				}
			}
		}
		f.Close()
	}
	return result
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
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			for _, kw := range keywords {
				if strings.HasPrefix(line, kw) {
					if len(line) > 120 {
						line = line[:120] + "..."
					}
					result = append(result, line)
					total += len(line)
					break
				}
			}
			if total > maxSigBytes {
				f.Close()
				return result
			}
		}
		f.Close()
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
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := sc.Text()
			trimmed := strings.TrimLeft(line, " \t")
			if strings.HasPrefix(trimmed, "def ") || strings.HasPrefix(trimmed, "class ") || strings.HasPrefix(trimmed, "async def ") {
				sig := strings.TrimSpace(trimmed)
				if len(sig) > 120 {
					sig = sig[:120] + "..."
				}
				result = append(result, sig)
				total += len(sig)
				if total > maxSigBytes {
					f.Close()
					return result
				}
			}
		}
		f.Close()
	}
	return result
}
