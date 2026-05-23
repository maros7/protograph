// Package projectstore manages global per-project, per-branch protograph storage.
// Graphs are stored at ~/.protograph/projects/<repo-id>/<branch>/graph.json
package projectstore

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// GraphPath returns the global storage path for the current project + branch.
func GraphPath(repoRoot string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	repoID := resolveRepoID(repoRoot)
	branch := detectBranch(repoRoot)
	safeBranch := strings.ReplaceAll(branch, "/", "--")
	return filepath.Join(home, ".protograph", "projects", repoID, safeBranch, "graph.json"), nil
}

// Write stores graph data to the global project store.
func Write(repoRoot string, data []byte) error {
	p, err := GraphPath(repoRoot)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o640)
}

// Read loads graph data from the global project store.
func Read(repoRoot string) ([]byte, error) {
	p, err := GraphPath(repoRoot)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		// Fallback: legacy local .protograph/graph.json
		local := filepath.Join(repoRoot, ".protograph", "graph.json")
		return os.ReadFile(local)
	}
	return data, nil
}

// BranchEntry holds metadata about a stored branch index.
type BranchEntry struct {
	Branch  string    `json:"branch"`
	Size    int64     `json:"size_bytes"`
	ModTime time.Time `json:"modified"`
	AgeDays int       `json:"age_days"`
}

// ListBranches lists all indexed branches for a project.
func ListBranches(repoRoot string) ([]BranchEntry, error) {
	home, _ := os.UserHomeDir()
	repoID := resolveRepoID(repoRoot)
	projDir := filepath.Join(home, ".protograph", "projects", repoID)

	entries, err := os.ReadDir(projDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var branches []BranchEntry
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		graphPath := filepath.Join(projDir, e.Name(), "graph.json")
		info, err := os.Stat(graphPath)
		if err != nil {
			continue
		}
		branches = append(branches, BranchEntry{
			Branch:  strings.ReplaceAll(e.Name(), "--", "/"),
			Size:    info.Size(),
			ModTime: info.ModTime(),
			AgeDays: int(time.Since(info.ModTime()).Hours() / 24),
		})
	}
	return branches, nil
}

// CleanStale removes branch indices older than maxAgeDays.
func CleanStale(repoRoot string, maxAgeDays int) (int, error) {
	home, _ := os.UserHomeDir()
	repoID := resolveRepoID(repoRoot)
	projDir := filepath.Join(home, ".protograph", "projects", repoID)

	branches, err := ListBranches(repoRoot)
	if err != nil {
		return 0, err
	}

	removed := 0
	for _, b := range branches {
		if isProtected(b.Branch) {
			continue
		}
		if b.AgeDays > maxAgeDays {
			dir := filepath.Join(projDir, strings.ReplaceAll(b.Branch, "/", "--"))
			if err := os.RemoveAll(dir); err == nil {
				removed++
			}
		}
	}
	return removed, nil
}

func resolveRepoID(repoRoot string) string {
	if url := gitCmd(repoRoot, "remote", "get-url", "origin"); url != "" {
		url = strings.TrimSuffix(url, ".git")
		url = strings.TrimPrefix(url, "https://")
		url = strings.TrimPrefix(url, "http://")
		url = strings.TrimPrefix(url, "git@")
		url = strings.Replace(url, ":", "/", 1)
		return url
	}
	abs, _ := filepath.Abs(repoRoot)
	return strings.ReplaceAll(abs, "/", "_")
}

func detectBranch(repoRoot string) string {
	if branch := gitCmd(repoRoot, "rev-parse", "--abbrev-ref", "HEAD"); branch != "" && branch != "HEAD" {
		return branch
	}
	if commit := gitCmd(repoRoot, "rev-parse", "--short", "HEAD"); commit != "" {
		return "detached-" + commit
	}
	return "unknown"
}

func gitCmd(dir string, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func isProtected(branch string) bool {
	switch branch {
	case "main", "master", "develop", "release":
		return true
	}
	return false
}

func init() {
	_ = fmt.Sprintf // keep fmt imported
}
