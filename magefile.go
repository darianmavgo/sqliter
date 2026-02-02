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

// MacApp builds a macOS application bundle
func MacApp() error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("MacApp target is only supported on macOS")
	}

	mg.Deps(Build)

	fmt.Println("🍎 Building macOS App Bundle...")

	appName := "Sqliter.app"
	appPath := filepath.Join("bin", appName)

	// AppleScript content
	scriptContent := `
on run
	set myPath to POSIX path of (path to me)
	set binPath to myPath & "Contents/Resources/sqliter"
	do shell script quoted form of binPath & " > /dev/null 2>&1 &"
end run

on open argv
	set myPath to POSIX path of (path to me)
	set binPath to myPath & "Contents/Resources/sqliter"
	repeat with aFile in argv
		set filePath to POSIX path of aFile
		do shell script quoted form of binPath & " " & quoted form of filePath & " > /dev/null 2>&1 &"
	end repeat
end open
`
	// Create temp script file
	tmpScript, err := os.CreateTemp("", "sqliter-launcher-*.scpt")
	if err != nil {
		return fmt.Errorf("failed to create temp script: %w", err)
	}
	defer os.Remove(tmpScript.Name())

	if _, err := tmpScript.WriteString(scriptContent); err != nil {
		return fmt.Errorf("failed to write script content: %w", err)
	}
	tmpScript.Close()

	// Compile AppleScript to App
	if err := sh.Run("osacompile", "-o", appPath, tmpScript.Name()); err != nil {
		return fmt.Errorf("osacompile failed: %w", err)
	}

	// Copy binary to Resources
	destBin := filepath.Join(appPath, "Contents/Resources", "sqliter")
	if err := sh.Copy(destBin, "bin/sqliter"); err != nil {
		return fmt.Errorf("failed to copy binary to app bundle: %w", err)
	}
	if err := os.Chmod(destBin, 0755); err != nil {
		return fmt.Errorf("failed to chmod binary: %w", err)
	}

	// Update Info.plist
	infoPlistPath := filepath.Join(appPath, "Contents/Info.plist")

	// Set Bundle Identifier
	if err := sh.Run("plutil", "-replace", "CFBundleIdentifier", "-string", "com.darianmavgo.sqliter", infoPlistPath); err != nil {
		return fmt.Errorf("failed to set CFBundleIdentifier: %w", err)
	}

	// Set Bundle Name
	if err := sh.Run("plutil", "-replace", "CFBundleName", "-string", "Sqliter", infoPlistPath); err != nil {
		return fmt.Errorf("failed to set CFBundleName: %w", err)
	}

	// Add Document Types
	docTypesXML := `
<array>
    <dict>
        <key>CFBundleTypeName</key>
        <string>SQLite Database</string>
        <key>CFBundleTypeRole</key>
        <string>Editor</string>
        <key>LSHandlerRank</key>
        <string>Owner</string>
        <key>CFBundleTypeExtensions</key>
        <array>
            <string>sqlite</string>
            <string>db</string>
            <string>sqlite3</string>
        </array>
    </dict>
</array>
`
	if err := sh.Run("plutil", "-insert", "CFBundleDocumentTypes", "-xml", docTypesXML, infoPlistPath); err != nil {
		// Try replace if insert failed
		if err := sh.Run("plutil", "-replace", "CFBundleDocumentTypes", "-xml", docTypesXML, infoPlistPath); err != nil {
			return fmt.Errorf("failed to set CFBundleDocumentTypes: %w", err)
		}
	}

	fmt.Printf("✅ Created %s\n", appPath)
	fmt.Println("To make this the default app for .sqlite files:")
	fmt.Println("1. Right-click a .sqlite file in Finder.")
	fmt.Println("2. Select 'Get Info'.")
	fmt.Println("3. In 'Open with:', select 'Sqliter.app'.")
	fmt.Println("4. Click 'Change All...'.")

	return nil
}

// SyncWailsUI builds the React client and copies it to the Wails frontend directory
func SyncWailsUI() error {
	fmt.Println("📦 Syncing UI assets for Wails...")

	// Build the UI first
	if err := BuildUI(); err != nil {
		return err
	}

	wailsDistDir := filepath.Join("cmd", "wailssqliter", "frontend", "dist")
	if err := os.MkdirAll(wailsDistDir, 0755); err != nil {
		return err
	}

	// Copy files from react-client/dist/ to cmd/wailssqliter/frontend/dist/
	return sh.RunV("cp", "-R", "react-client/dist/", wailsDistDir+"/")
}

// WailsBuild builds the Wails desktop application
func WailsBuild() error {
	mg.Deps(SyncWailsUI)
	fmt.Println("🖥️  Building Wails desktop application...")

	// Save current directory
	oldDir, err := os.Getwd()
	if err != nil {
		return err
	}
	defer os.Chdir(oldDir)

	// We need to run wails build from cmd/wailssqliter
	wailsDir := filepath.Join("cmd", "wailssqliter")
	if err := os.Chdir(wailsDir); err != nil {
		return err
	}

	// Check if wails is installed
	if _, err := exec.LookPath("wails"); err != nil {
		return fmt.Errorf("wails command not found. Please install it with 'go install github.com/wailsapp/wails/v2/cmd/wails@latest'")
	}

	return sh.RunV("wails", "build", "-s")
}

// WailsDev runs the Wails desktop application in development mode
func WailsDev() error {
	fmt.Println("🚀 Starting Wails development mode...")

	// Save current directory
	oldDir, err := os.Getwd()
	if err != nil {
		return err
	}
	defer os.Chdir(oldDir)

	wailsDir := filepath.Join("cmd", "wailssqliter")
	if err := os.Chdir(wailsDir); err != nil {
		return err
	}

	if _, err := exec.LookPath("wails"); err != nil {
		return fmt.Errorf("wails command not found. Please install it with 'go install github.com/wailsapp/wails/v2/cmd/wails@latest'")
	}

	return sh.RunV("wails", "dev")
}

// WailsInstall builds the Wails application and installs it to the /Applications directory
func WailsInstall() error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("WailsInstall is only supported on macOS")
	}

	mg.Deps(WailsBuild)

	fmt.Println("🍎 Installing Wails App to /Applications...")

	srcApp := filepath.Join("cmd", "wailssqliter", "build", "bin", "wailssqliter.app")
	destApp := "/Applications/SQLiter.app"

	// 1. Remove existing app if it exists
	if _, err := os.Stat(destApp); err == nil {
		fmt.Printf("🗑️  Removing existing %s...\n", destApp)
		if err := sh.Run("rm", "-rf", destApp); err != nil {
			return fmt.Errorf("failed to remove existing app: %w", err)
		}
	}

	// 2. Copy the app bundle
	// Using cp -R to copy the bundle.
	if err := sh.RunV("cp", "-R", srcApp, destApp); err != nil {
		return fmt.Errorf("failed to copy app to /Applications: %w", err)
	}

	// 3. Register with Launch Services to ensure the system picks up the new app
	fmt.Println("🚀 Registering with Launch Services...")
	lsregister := "/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"
	// Fallback path for different macOS versions
	if _, err := os.Stat(lsregister); err != nil {
		lsregister = "/System/Library/Frameworks/CoreServices.framework/Versions/A/Frameworks/LaunchServices.framework/Versions/A/Support/lsregister"
	}

	if _, err := os.Stat(lsregister); err == nil {
		// -f forces re-registration
		if err := sh.Run(lsregister, "-f", destApp); err != nil {
			fmt.Printf("Warning: lsregister failed: %v\n", err)
		}
	} else {
		fmt.Println("⚠️  lsregister not found, system may not pick up changes immediately.")
	}

	fmt.Printf("✅ Wails App installed to: %s\n", destApp)
	return nil
}
