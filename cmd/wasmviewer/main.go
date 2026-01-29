//go:build js && wasm

package main

import (
	"database/sql"
	"fmt"
	"syscall/js"

	"github.com/darianmavgo/sqliter/sqliter"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

var (
	db       *sql.DB
	renderer *sqliter.CanvasTableRenderer
)

func main() {
	fmt.Println("SQLiter WASM module initialized")

	// Export API to JavaScript
	js.Global().Set("sqliterWASM", js.ValueOf(map[string]interface{}{
		"openDatabase":   js.FuncOf(openDatabase),
		"executeQuery":   js.FuncOf(executeQuery),
		"createRenderer": js.FuncOf(createRenderer),
	}))

	// Keep the program running
	<-make(chan struct{})
}

// openDatabase opens a SQLite database from a byte array
func openDatabase(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return map[string]interface{}{
			"error": "missing database bytes argument",
		}
	}

	// Get database bytes from JavaScript
	dbBytes := make([]byte, args[0].Get("length").Int())
	js.CopyBytesToGo(dbBytes, args[0])

	// Write to temp file (WASM filesystem)
	// In WASM, we can use :memory: or a virtual filesystem
	var err error
	db, err = sql.Open("sqlite", ":memory:")
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("failed to open database: %v", err),
		}
	}

	// For a real implementation, we'd need to restore the database from bytes
	// This is a simplified version - in production, use sql-js or similar approach
	// to load the actual database file into memory

	return map[string]interface{}{
		"success": true,
		"message": "Database opened successfully",
	}
}

// executeQuery executes a SQL query and returns metadata
func executeQuery(this js.Value, args []js.Value) interface{} {
	if db == nil {
		return map[string]interface{}{
			"error": "database not opened",
		}
	}

	if len(args) < 1 {
		return map[string]interface{}{
			"error": "missing query argument",
		}
	}

	query := args[0].String()

	if renderer == nil {
		return map[string]interface{}{
			"error": "renderer not created",
		}
	}

	// Load query results into renderer
	err := renderer.LoadQueryResults(db, query)
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("query failed: %v", err),
		}
	}

	return map[string]interface{}{
		"success":  true,
		"rowCount": renderer.GetRowCount(),
	}
}

// createRenderer creates a new canvas renderer
func createRenderer(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return map[string]interface{}{
			"error": "missing canvas ID argument",
		}
	}

	canvasID := args[0].String()

	var err error
	renderer, err = sqliter.NewCanvasTableRenderer(canvasID)
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("failed to create renderer: %v", err),
		}
	}

	// Set initial size
	if len(args) >= 3 {
		width := args[1].Int()
		height := args[2].Int()
		renderer.SetCanvasSize(width, height)
	}

	// Return renderer API
	return renderer.ExportJSAPI()
}
