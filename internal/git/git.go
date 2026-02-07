package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GetRootPath finds the Git root directory using git rev-parse --show-toplevel
func GetRootPath() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository or git not found: %w", err)
	}
	
	rootPath := strings.TrimSpace(string(output))
	
	// Convert to absolute path
	absPath, err := filepath.Abs(rootPath)
	if err != nil {
		return rootPath, nil // Return original if abs fails
	}
	
	return absPath, nil
}

// HasChanges checks if there are uncommitted changes
func HasChanges() (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check git status: %w", err)
	}
	
	return len(strings.TrimSpace(string(output))) > 0, nil
}

// GetDiff returns the diff of uncommitted changes
func GetDiff() (string, error) {
	cmd := exec.Command("git", "diff")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git diff: %w", err)
	}
	
	return string(output), nil
}

// AddAll stages all changes
func AddAll() error {
	cmd := exec.Command("git", "add", ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Commit creates a commit with the given message
func Commit(message string) error {
	// Escape the message properly for git commit
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Push pushes changes to remote
func Push() error {
	cmd := exec.Command("git", "push")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// GetRepoName extracts repository name from the root path
func GetRepoName(rootPath string) string {
	return filepath.Base(rootPath)
}

// ChangeToRoot changes the working directory to the Git root
func ChangeToRoot(rootPath string) error {
	return os.Chdir(rootPath)
}

