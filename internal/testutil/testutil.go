package testutil

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// CleanTestArtifacts recursively removes files and directories that start with "test_"
// or end with ".test" within the specified root directory.
func CleanTestArtifacts(root string, dryRun bool) error {
	log.Printf("Cleaning test artifacts in %s (dryRun=%v)...", root, dryRun)

	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the root dir itself
		if path == root {
			return nil
		}

		name := info.Name()

		if info.IsDir() {
			// Skip protected directories
			if name == "sample_data" || name == ".git" || name == ".idea" || name == ".vscode" || name == "test_output" {
				return filepath.SkipDir
			}
		}

		shouldDelete := false

		// Criteria for deletion
		if strings.HasPrefix(name, "test_") {
			shouldDelete = true
		}
		if strings.HasSuffix(name, ".test") {
			shouldDelete = true
		}

		if shouldDelete {
			if dryRun {
				log.Printf("[DRY RUN] Would delete: %s", path)
			} else {
				log.Printf("Deleting: %s", path)
				if info.IsDir() {
					if err := os.RemoveAll(path); err != nil {
						log.Printf("Failed to remove dir %s: %v", path, err)
					} else {
						return filepath.SkipDir // Don't walk into deleted dir
					}
				} else {
					if err := os.Remove(path); err != nil {
						log.Printf("Failed to remove file %s: %v", path, err)
					}
				}
			}
		}

		return nil
	})
}

// GetTestOutputDir returns a path to a directory within project_root/test_output
// it ensures the directory exists and is unique for the test run.
func GetTestOutputDir(t *testing.T, prefix string) string {
	// Find project root by looking for go.mod
	root := "."
	for i := 0; i < 5; i++ {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			break
		}
		abs, _ := filepath.Abs(root)
		if abs == "/" {
			break
		}
		root = filepath.Join("..", root)
	}

	testOutputDir := filepath.Join(root, "test_output")
	if err := os.MkdirAll(testOutputDir, 0755); err != nil {
		t.Fatalf("Failed to create test_output dir: %v", err)
	}

	// Create a subfolder for this specific test
	subDir := filepath.Join(testOutputDir, prefix+"_"+time.Now().Format("20060102_150405"))
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create test sub-dir: %v", err)
	}

	return subDir
}
