package scanner

import (
	"fmt"
	"os/exec"
	"strings"
)

// GitInfo holds relevant git metadata.
type GitInfo struct {
	RecentCommits string
	Branch        string
	RemoteURL     string
	IsRepo        bool
}

// ScanGit reads git metadata from the given root.
func ScanGit(root string) *GitInfo {
	g := &GitInfo{}

	// Check if it's a git repo
	cmd := exec.Command("git", "-C", root, "rev-parse", "--is-inside-work-tree")
	if err := cmd.Run(); err != nil {
		return g // not a repo, return empty
	}
	g.IsRepo = true

	// Current branch
	out, err := runGit(root, "rev-parse", "--abbrev-ref", "HEAD")
	if err == nil {
		g.Branch = strings.TrimSpace(out)
	}

	// Remote URL
	out, err = runGit(root, "remote", "get-url", "origin")
	if err == nil {
		g.RemoteURL = strings.TrimSpace(out)
	}

	// Last 30 commits (oneline)
	out, err = runGit(root, "log", "--oneline", "-30")
	if err == nil {
		g.RecentCommits = strings.TrimSpace(out)
	}

	return g
}

func runGit(root string, args ...string) (string, error) {
	allArgs := append([]string{"-C", root}, args...)
	cmd := exec.Command("git", allArgs...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %v: %w", args, err)
	}
	return string(out), nil
}
