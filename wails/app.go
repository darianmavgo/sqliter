package wails

import (
	"context"

	"github.com/darianmavgo/sqliter/sqliter"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx    context.Context
	engine *sqliter.Engine
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

func (a *App) ListFiles(dir string) ([]sqliter.FileEntry, error) {
	return a.engine.ListFiles(dir)
}

func (a *App) ListTables(db string) ([]sqliter.TableInfo, error) {
	return a.engine.ListTables(db)
}

func (a *App) Query(opts sqliter.QueryOptions) (*sqliter.QueryResult, error) {
	return a.engine.Query(opts)
}

// OpenFile is called when macOS sends a file open event
func (a *App) OpenFile(filePath string) {
	if filePath != "" {
		// Emit event to frontend with the file path
		runtime.EventsEmit(a.ctx, "open-file", filePath)
	}
}
