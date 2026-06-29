package fileops

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ReadFile reads a file and returns its contents as a string.
func ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading file %s: %w", path, err)
	}
	return string(data), nil
}

// WriteFile writes content to a file, creating directories as needed.
func WriteFile(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directories for %s: %w", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing file %s: %w", path, err)
	}
	return nil
}

// StrReplace replaces old string with new string in a file.
// Returns the number of replacements made.
func StrReplace(path, oldStr, newStr string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("reading file %s: %w", path, err)
	}

	content := string(data)
	count := strings.Count(content, oldStr)
	if count == 0 {
		return 0, fmt.Errorf("no match found in %s", path)
	}

	newContent := strings.ReplaceAll(content, oldStr, newStr)
	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		return 0, fmt.Errorf("writing file %s: %w", path, err)
	}

	return count, nil
}

// SearchResult represents a single search result.
type SearchResult struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Content string `json:"content"`
}

// CodeSearch searches for a pattern in files using ripgrep or fallback grep.
func CodeSearch(pattern string, flags ...string) ([]SearchResult, error) {
	// Try ripgrep first (faster)
	args := []string{"--line-number", "--with-filename"}
	args = append(args, flags...)
	args = append(args, pattern)

	cmd := exec.Command("rg", args...)
	output, err := cmd.Output()
	if err != nil {
		// rg returns exit code 1 if no matches found
		if exitErr, ok := err.(*exec.ExitError); ok {
			if len(exitErr.Stderr) > 0 {
				return nil, fmt.Errorf("search error: %s", string(exitErr.Stderr))
			}
			// No matches found
			return nil, nil
		}
		// If rg is not installed, fall back to grep
		return grepSearch(pattern)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var results []SearchResult
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 3)
		if len(parts) >= 3 {
			lineNum := 0
			fmt.Sscanf(parts[1], "%d", &lineNum)
			results = append(results, SearchResult{
				Path:    parts[0],
				Line:    lineNum,
				Content: parts[2],
			})
		} else if len(parts) == 2 {
			results = append(results, SearchResult{
				Path:    parts[0],
				Line:    0,
				Content: parts[1],
			})
		}
	}
	return results, nil
}

// grepSearch is a fallback search using grep.
func grepSearch(pattern string) ([]SearchResult, error) {
	cmd := exec.Command("grep", "-rn", "--include=*.go", "--include=*.js", "--include=*.ts",
		"--include=*.py", "--include=*.rs", "--include=*.java", "--include=*.md",
		"--include=*.json", "--include=*.yaml", "--include=*.yml",
		"--include=*.html", "--include=*.css", "--include=*.sh",
		pattern, ".")
	output, err := cmd.Output()
	if err != nil {
		return nil, nil // No matches
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var results []SearchResult
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 3)
		if len(parts) >= 3 {
			lineNum := 0
			fmt.Sscanf(parts[1], "%d", &lineNum)
			results = append(results, SearchResult{
				Path:    parts[0],
				Line:    lineNum,
				Content: parts[2],
			})
		}
	}
	return results, nil
}

// ListDirectory lists files and directories in the specified path.
func ListDirectory(path string) ([]string, []string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, nil, fmt.Errorf("listing directory %s: %w", path, err)
	}

	var files, dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry.Name())
		} else {
			files = append(files, entry.Name())
		}
	}
	return files, dirs, nil
}

// Glob finds files matching a glob pattern.
func Glob(pattern string) ([]string, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob %s: %w", pattern, err)
	}
	return matches, nil
}

// FileExists checks if a file exists.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// DeleteFile deletes a file.
func DeleteFile(path string) error {
	return os.Remove(path)
}
