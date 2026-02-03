package wails

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/darianmavgo/sqliter/sqliter"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx         context.Context
	engine      *sqliter.Engine
	pendingFile string
}

// NewApp creates a new App application struct
func NewApp() *App {
	cfg := &sqliter.Config{
		ServeFolder: "/",
	}
	return &App{
		engine: sqliter.NewEngine(cfg),
	}
}

// Startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	if a.pendingFile != "" {
		// Small delay or just reliance on frontend polling/ready event
		runtime.EventsEmit(a.ctx, "open-file", a.pendingFile)
	}
}

// Shutdown is called at termination
func (a *App) Shutdown(ctx context.Context) {
}

// OpenDatabase prompts the user to select a SQLite file
func (a *App) OpenDatabase() (string, error) {
	selection, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select SQLite Database",
		Filters: []runtime.FileFilter{
			{DisplayName: "SQLite Files", Pattern: "*.db;*.sqlite;*.sqlite3"},
			{DisplayName: "All Files", Pattern: "*.*"},
		},
	})

	if err != nil {
		return "", err
	}

	if selection == "" {
		return "", nil // User cancelled
	}

	return selection, nil
}

// expandHome expands the tilde (~) in the path to the user's home directory
func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			if path == "~" {
				return home
			}
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func (a *App) ListFiles(dir string) ([]sqliter.FileEntry, error) {
	dir = expandHome(dir)
	// Use app context or TODO. App context is cancelled on shutdown.
	return a.engine.ListFiles(a.ctx, dir)
}

func (a *App) ListTables(db string) ([]sqliter.TableInfo, error) {
	db = expandHome(db)
	return a.engine.ListTables(a.ctx, db)
}

func (a *App) Query(opts sqliter.QueryOptions) (*sqliter.QueryResult, error) {
	start := time.Now()
	opts.BanquetPath = expandHome(opts.BanquetPath)
	res, err := a.engine.Query(a.ctx, opts)
	fmt.Printf("[Wails.App.Query] Took %v\n", time.Since(start))
	return res, err
}

// OpenFile is called when macOS sends a file open event
func (a *App) OpenFile(filePath string) {
	fmt.Println("Received OpenFile:", filePath)
	a.pendingFile = filePath
	if a.ctx != nil {
		// Emit event to frontend with the file path
		runtime.EventsEmit(a.ctx, "open-file", filePath)
	}
}

// GetPendingFile returns the file that was opened before the frontend was ready
func (a *App) GetPendingFile() string {
	p := a.pendingFile
	a.pendingFile = "" // Clear it once read
	return p
}
