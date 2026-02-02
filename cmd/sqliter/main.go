package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/darianmavgo/sqliter/sqliter"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 {
		if os.Args[1] == "--version" {
			fmt.Printf("sqliter version %s\n", version)
			return
		}
		if os.Args[1] == "--help" {
			fmt.Println("Usage: sqliter [file or url]")
			fmt.Println("  file: path to local sqlite database file or directory containing database files")
			fmt.Println("  url:  url to a remote sqlite database file")
			return
		}
	}

	var arg string
	if len(os.Args) < 2 {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("Failed to get user home dir: %v", err)
		}
		arg = home
	} else {
		arg = os.Args[1]
	}
	var dataDir string
	var fileName string

	if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
		// Handle URL
		fmt.Printf("Downloading %s...\n", arg)
		tmpDir, err := os.MkdirTemp("", "sqliter-download")
		if err != nil {
			log.Fatalf("Failed to create temp dir: %v", err)
		}
		// We usually want to clean up, but maybe not if we are serving?
		// defer os.RemoveAll(tmpDir)

		// Infer filename and suffix (for Banquet deep links)
		// We look for common sqlite extensions to split the URL
		var downloadURL, suffix string

		exts := []string{".db", ".sqlite", ".sqlite3", ".sdb", ".s3db", ".csv.db", ".xlsx.db"}
		splitIdx := -1

		lowerArg := strings.ToLower(arg)
		for _, ext := range exts {
			idx := strings.Index(lowerArg, ext)
			if idx != -1 {
				// Verify it's a valid end of segment (followed by / or ? or end of string)
				end := idx + len(ext)
				if end == len(arg) || arg[end] == '/' || arg[end] == '?' {
					splitIdx = end
					break
				}
			}
		}

		if splitIdx != -1 {
			downloadURL = arg[:splitIdx]
			suffix = arg[splitIdx:]
		} else {
			downloadURL = arg
			suffix = ""
		}

		parts := strings.Split(downloadURL, "/")
		if len(parts) > 0 {
			fileName = parts[len(parts)-1]
		}
		if fileName == "" {
			fileName = "downloaded.db"
		}

		destPath := filepath.Join(tmpDir, fileName)
		if err := downloadFile(downloadURL, destPath); err != nil {
			log.Fatalf("Failed to download file: %v", err)
		}

		// If we have a suffix, append it to filename for the browser URL
		if suffix != "" {
			fileName += suffix
		}

		dataDir = tmpDir
	} else {
		// Handle Local File
		absPath, err := filepath.Abs(arg)
		if err != nil {
			log.Fatalf("Failed to get absolute path: %v", err)
		}

		info, err := os.Stat(absPath)
		if err != nil {
			log.Fatalf("File not found: %v", err)
		}

		if info.IsDir() {
			dataDir = absPath
			fileName = "" // Serve directory listing
		} else {
			dataDir = filepath.Dir(absPath)
			fileName = filepath.Base(absPath)
		}
	}

	// Get a random available port (preferring IPv4)
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close() // Release it so server can bind

	cfg := sqliter.DefaultConfig()
	cfg.ServeFolder = dataDir
	cfg.Verbose = true

	srv := sqliter.NewServer(cfg)

	url := fmt.Sprintf("http://127.0.0.1:%d", port)
	if fileName != "" {
		url = fmt.Sprintf("%s/%s", url, fileName)
	}

	fmt.Printf("Launching sqliter view...\n")
	fmt.Printf("Data Directory: %s\n", dataDir)
	fmt.Printf("Listening at: %s\n", url)
	fmt.Printf("SERVING_AT=%s\n", url)

	// Attempt to open browser (best effort)
	openBrowser(url)

	// Setup HTTP routes
	mux := http.NewServeMux()

	// Main table viewer routes
	mux.Handle("/", srv)

	log.Fatal(http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", port), mux))
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	// Assume macOS for now based on context
	cmd = exec.Command("open", url)
	if err := cmd.Start(); err != nil {
		fmt.Printf("Failed to open browser: %v\n", err)
	}
}

func downloadFile(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
