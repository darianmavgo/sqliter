//go:build mage
// +build mage

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Default target to run when none is specified
var Default = Build

// Clean removes build artifacts
func Clean() error {
	fmt.Println("🧹 Cleaning build artifacts...")
	dirs := []string{"bin", "sqliter/ui", "react-client/dist", "react-client/node_modules"}
	for _, dir := range dirs {
		if err := sh.Rm(dir); err != nil {
			fmt.Printf("Warning: failed to remove %s: %v\n", dir, err)
		}
	}
	return nil
}

// BuildUI builds the React client
func BuildUI() error {
	fmt.Println("⚛️  Building React client...")

	// Install dependencies
	if err := sh.RunV("npm", "install", "--prefix", "react-client"); err != nil {
		return fmt.Errorf("npm install failed: %w", err)
	}

	// Build
	if err := sh.RunV("npm", "run", "build", "--prefix", "react-client"); err != nil {
		return fmt.Errorf("npm build failed: %w", err)
	}

	return nil
}

// SyncUI copies the built React client to the Go embed directory
func SyncUI() error {
	mg.Deps(BuildUI)

	fmt.Println("📦 Syncing UI assets to Go embed directory...")

	// Remove old UI
	if err := sh.Rm("sqliter/ui"); err != nil {
		fmt.Printf("Warning: failed to remove old UI: %v\n", err)
	}

	// Create directory
	if err := os.MkdirAll("sqliter/ui", 0755); err != nil {
		return fmt.Errorf("failed to create ui directory: %w", err)
	}

	// Copy files
	if err := sh.RunV("cp", "-R", "react-client/dist/", "sqliter/ui/"); err != nil {
		return fmt.Errorf("failed to copy UI files: %w", err)
	}

	return nil
}

// Build builds the sqliter binary
func Build() error {
	mg.Deps(SyncUI)

	fmt.Println("🔨 Building sqliter binary...")

	if err := os.MkdirAll("bin", 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	env := map[string]string{
		"CGO_ENABLED": "0",
	}

	return sh.RunWith(env, "go", "build", "-o", "bin/sqliter", "./cmd/sqliter")
}

// Dev builds and runs the server with sample data
func Dev() error {
	mg.Deps(Build)

	fmt.Println("🚀 Starting development server...")

	// Kill any existing instances
	sh.Run("killall", "sqliter")

	// Start server
	return sh.RunV("./bin/sqliter", "sample_data")
}

// Relaunch kills old instances, rebuilds, and starts the server
func Relaunch() error {
	fmt.Println("🔄 Relaunching sqliter...")

	// Kill old instances
	fmt.Println("Killing old instances...")
	sh.Run("killall", "sqliter")

	// Build
	if err := Build(); err != nil {
		return err
	}

	// Create logs directory
	os.MkdirAll("logs", 0755)

	// Start in background
	fmt.Println("Starting server...")
	cmd := exec.Command("./bin/sqliter", "sample_data")

	logFile, err := os.Create("logs/server.log")
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	defer logFile.Close()

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	fmt.Printf("Server running with PID %d\n", cmd.Process.Pid)
	fmt.Println("Check logs/server.log for output")

	return nil
}

// Install installs sqliter to the system
func Install() error {
	mg.Deps(Build)

	fmt.Println("📥 Installing sqliter...")

	var installPath string

	switch runtime.GOOS {
	case "darwin", "linux":
		// Check if user has write access to /usr/local/bin
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		// Try /usr/local/bin first, fall back to ~/bin
		if _, err := os.Stat("/usr/local/bin"); err == nil {
			installPath = "/usr/local/bin/sqliter"

			// Check if we can write
			testFile := "/usr/local/bin/.test_write"
			if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
				// No write access, use ~/bin instead
				installPath = filepath.Join(homeDir, "bin", "sqliter")
				fmt.Printf("⚠️  No write access to /usr/local/bin, installing to %s\n", installPath)
			} else {
				os.Remove(testFile)
			}
		} else {
			installPath = filepath.Join(homeDir, "bin", "sqliter")
		}

		// Ensure directory exists
		installDir := filepath.Dir(installPath)
		if err := os.MkdirAll(installDir, 0755); err != nil {
			return fmt.Errorf("failed to create install directory: %w", err)
		}

		// Copy binary
		if err := sh.Copy(installPath, "bin/sqliter"); err != nil {
			return fmt.Errorf("failed to copy binary: %w", err)
		}

		// Make executable
		if err := os.Chmod(installPath, 0755); err != nil {
			return fmt.Errorf("failed to make binary executable: %w", err)
		}

		fmt.Printf("✅ Installed to: %s\n", installPath)

		// Check if install directory is in PATH
		pathEnv := os.Getenv("PATH")
		if !strings.Contains(pathEnv, installDir) {
			fmt.Printf("\n⚠️  Warning: %s is not in your PATH\n", installDir)
			fmt.Printf("Add this to your shell profile (~/.zshrc or ~/.bashrc):\n")
			fmt.Printf("    export PATH=\"%s:$PATH\"\n", installDir)
		}

	case "windows":
		return fmt.Errorf("Windows installation not yet supported. Please copy bin/sqliter.exe to a directory in your PATH manually")
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	return nil
}

// Uninstall removes sqliter from the system
func Uninstall() error {
	fmt.Println("🗑️  Uninstalling sqliter...")

	var installPaths []string

	switch runtime.GOOS {
	case "darwin", "linux":
		homeDir, _ := os.UserHomeDir()
		installPaths = []string{
			"/usr/local/bin/sqliter",
			filepath.Join(homeDir, "bin", "sqliter"),
		}
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	removed := false
	for _, path := range installPaths {
		if _, err := os.Stat(path); err == nil {
			if err := os.Remove(path); err != nil {
				fmt.Printf("⚠️  Failed to remove %s: %v\n", path, err)
			} else {
				fmt.Printf("✅ Removed: %s\n", path)
				removed = true
			}
		}
	}

	if !removed {
		fmt.Println("ℹ️  sqliter not found in standard install locations")
	}

	return nil
}

// Test runs all tests
func Test() error {
	fmt.Println("🧪 Running tests...")
	return sh.RunV("go", "test", "./...")
}

// Fmt formats all Go code
func Fmt() error {
	fmt.Println("✨ Formatting code...")
	return sh.RunV("go", "fmt", "./...")
}

// Lint runs linters
func Lint() error {
	fmt.Println("🔍 Running linters...")

	// Check if golangci-lint is installed
	if _, err := exec.LookPath("golangci-lint"); err != nil {
		fmt.Println("⚠️  golangci-lint not found, skipping lint")
		return nil
	}

	return sh.RunV("golangci-lint", "run")
}

// All runs fmt, lint, test, and build
func All() error {
	mg.Deps(Fmt, Lint, Test, Build)
	return nil
}
