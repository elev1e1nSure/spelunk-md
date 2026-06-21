package scanner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Snapshot represents a point-in-time check of the project's output context files.
type Snapshot struct {
	Commit     string    `json:"commit"`
	AnalyzedAt time.Time `json:"analyzed_at"`
	Output     string    `json:"output"`
}

// ReadSnapshot reads the snapshot metadata from the project root.
func ReadSnapshot(root string) (*Snapshot, error) {
	path := filepath.Join(root, ".spelunk", "snapshot.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, err
	}
	return &snap, nil
}

// WriteSnapshot writes the snapshot metadata to the project root.
func WriteSnapshot(root string, snap *Snapshot) error {
	dir := filepath.Join(root, ".spelunk")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	path := filepath.Join(dir, "snapshot.json")
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// GetHeadCommit returns the current git HEAD commit hash.
func GetHeadCommit(root string) (string, error) {
	out, err := runGit(root, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// DiffFiles returns paths of files modified between the snapshot commit and HEAD,
// relative to the given project root.
func DiffFiles(root string, commit string) ([]string, error) {
	// Get prefix of root relative to git repository root.
	prefixOut, err := runGit(root, "rev-parse", "--show-prefix")
	if err != nil {
		return nil, err
	}
	prefix := strings.TrimSpace(prefixOut)

	// Run git diff <commit> HEAD --name-only.
	diffOut, err := runGit(root, "diff", commit, "HEAD", "--name-only")
	if err != nil {
		return nil, err
	}

	var files []string
	lines := strings.Split(diffOut, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Normalize line to use forward slashes for matching the prefix.
		lineSlash := filepath.ToSlash(line)
		if prefix != "" {
			if !strings.HasPrefix(lineSlash, prefix) {
				continue
			}
			rel := strings.TrimPrefix(lineSlash, prefix)
			files = append(files, filepath.FromSlash(rel))
		} else {
			files = append(files, filepath.FromSlash(lineSlash))
		}
	}
	return files, nil
}
