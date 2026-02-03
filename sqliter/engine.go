package sqliter

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/darianmavgo/banquet"
	"github.com/darianmavgo/banquet/sqlite"
	_ "modernc.org/sqlite" // Register modernc driver
)

// Engine handles the core logic, agnostic of HTTP or Wails
type Engine struct {
	config *Config

	mu    sync.Mutex
	conns map[string]*sql.DB
}

func NewEngine(cfg *Config) *Engine {
	return &Engine{
		config: cfg,
		conns:  make(map[string]*sql.DB),
	}
}

// CloseAll closes all cached database connections
func (e *Engine) CloseAll() {
	e.mu.Lock()
	defer e.mu.Unlock()
	for path, db := range e.conns {
		db.Close()
		delete(e.conns, path)
	}
}

// getDBConnection returns a cached connection or opens a new one
func (e *Engine) getDBConnection(ctx context.Context, dbPath string) (*sql.DB, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Check if already open
	if db, ok := e.conns[dbPath]; ok {
		// Ping to ensure it's still alive
		if err := db.PingContext(ctx); err == nil {
			return db, nil
		}
		// If ping fails, close and remove to reopen
		db.Close()
		delete(e.conns, dbPath)
	}

	// Open new connection with WAL mode enabled for better concurrency
	// Note: modernc.org/sqlite registers as "sqlite"
	dsn := fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	// Set some reasonable defaults for the pool
	// modernc_sqlite handles concurrency, but let's keep it reasonable per connection object
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute) // Close idle connections after 5 mins

	e.conns[dbPath] = db

	return db, nil
}

type FileEntry struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// ListFiles returns a list of files in a directory (safe, strict relative paths)
func (e *Engine) ListFiles(ctx context.Context, dirRelPath string) ([]FileEntry, error) {
	if strings.Contains(dirRelPath, "..") {
		return nil, fmt.Errorf("invalid path")
	}

	targetDir := filepath.Join(e.config.ServeFolder, dirRelPath)
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	files := make([]FileEntry, 0)
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}

		if entry.IsDir() {
			files = append(files, FileEntry{Name: name, Type: "directory"})
		} else {
			// Check extension
			if strings.HasSuffix(name, ".db") || strings.HasSuffix(name, ".sqlite") ||
				strings.HasSuffix(name, ".csv.db") || strings.HasSuffix(name, ".xlsx.db") {
				files = append(files, FileEntry{Name: name, Type: "database"})
			}
		}
	}
	return files, nil
}

type TableInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func (e *Engine) ListTables(ctx context.Context, dbRelPath string) ([]TableInfo, error) {
	if strings.Contains(dbRelPath, "..") {
		return nil, fmt.Errorf("invalid path")
	}

	dbPath := filepath.Join(e.config.ServeFolder, dbRelPath)

	// Use cached connection
	db, err := e.getDBConnection(ctx, dbPath)
	if err != nil {
		return nil, fmt.Errorf("error opening DB: %w", err)
	}

	rows, err := db.QueryContext(ctx, "SELECT name, type FROM sqlite_master WHERE type IN ('table', 'view') ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer rows.Close()

	var tables []TableInfo
	for rows.Next() {
		var name, type_ string
		if err := rows.Scan(&name, &type_); err != nil {
			continue
		}
		tables = append(tables, TableInfo{Name: name, Type: type_})
	}
	return tables, nil
}

type QueryOptions struct {
	BanquetPath     string
	FilterWhere     string // SQL fragment
	FilterModelJSON string // AgGrid Filter Model JSON
	SortCol         string
	SortDir         string
	Offset          int
	Limit           int
	ForceZeroLimit  bool // If true, explicitly set Limit to 0
	AllowOverride   bool // If true, Limit/Offset in options override BanquetPath defaults
	SkipTotalCount  bool // If true, skips the COUNT(*) query for performance
}

type QueryResult struct {
	Columns    []string        `json:"columns"`
	Values     [][]interface{} `json:"values"`
	TotalCount int             `json:"totalCount"`
	SQL        string          `json:"sql"`
}

func (e *Engine) Query(ctx context.Context, opts QueryOptions) (*QueryResult, error) {
	start := time.Now()
	last := start

	bq, err := banquet.ParseNested(opts.BanquetPath)
	if err != nil {
		return nil, fmt.Errorf("error parsing URL: %w", err)
	}
	fmt.Printf("[Engine.Query] ParseNested took %v\n", time.Since(last))
	last = time.Now()

	// Override limit/offset if provided
	if opts.AllowOverride {
		if opts.Limit > 0 {
			bq.Limit = fmt.Sprintf("%d", opts.Limit)
			bq.Offset = fmt.Sprintf("%d", opts.Offset)
		} else if opts.ForceZeroLimit {
			bq.Limit = "0"
			bq.Offset = fmt.Sprintf("%d", opts.Offset)
		}
	}

	if opts.SortCol != "" {
		bq.OrderBy = opts.SortCol
		if opts.SortDir != "" {
			bq.SortDirection = opts.SortDir
		}
	}

	if opts.FilterModelJSON != "" {
		fmWhere, err := BuildWhereClause(opts.FilterModelJSON)
		if err != nil {
			return nil, fmt.Errorf("error building filter: %w", err)
		}
		if fmWhere != "" {
			if opts.FilterWhere != "" {
				opts.FilterWhere = fmt.Sprintf("(%s) AND (%s)", opts.FilterWhere, fmWhere)
			} else {
				opts.FilterWhere = fmWhere
			}
		}
	}

	if opts.FilterWhere != "" {
		if bq.Where != "" {
			bq.Where = fmt.Sprintf("(%s) AND (%s)", bq.Where, opts.FilterWhere)
		} else {
			bq.Where = opts.FilterWhere
		}
	}

	dataSetPath := strings.TrimPrefix(bq.DataSetPath, "/")
	if strings.Contains(dataSetPath, "..") {
		return nil, fmt.Errorf("invalid path")
	}

	dbPath := filepath.Join(e.config.ServeFolder, dataSetPath)

	// Use cached connection
	db, err := e.getDBConnection(ctx, dbPath)
	if err != nil {
		return nil, fmt.Errorf("error opening DB: %w", err)
	}

	fmt.Printf("[Engine.Query] DB Open/Fetch took %v\n", time.Since(last))
	last = time.Now()

	// Run harmless optimization pragmas once might be risky if we assume strict logic.
	// But `cache_size` is connection-local and safe.
	if _, err := db.ExecContext(ctx, "PRAGMA cache_size = -2000; PRAGMA case_sensitive_like = OFF;"); err != nil {
		// ignore
	}
	fmt.Printf("[Engine.Query] PRAGMAs took %v\n", time.Since(last))
	last = time.Now()

	// Handle case where table name is missing
	if bq.Table == "" {
		rows, err := db.QueryContext(ctx, "SELECT name FROM sqlite_master WHERE type IN ('table', 'view') ORDER BY name")
		if err != nil {
			return nil, fmt.Errorf("failed to list tables: %w", err)
		}
		var tables []string
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err == nil {
				tables = append(tables, name)
			}
		}
		rows.Close()

		if len(tables) == 1 {
			bq.Table = tables[0]
		} else if len(tables) == 0 {
			return nil, fmt.Errorf("no tables found in database")
		} else {
			return nil, fmt.Errorf("table name required. Available tables: %s", strings.Join(tables, ", "))
		}
	}

	query := sqlite.Compose(bq)

	// Get total count
	var totalCount int = -1
	if !opts.SkipTotalCount {
		countStart := time.Now()
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", sqlite.QuoteIdentifier(bq.Table))
		if bq.Where != "" {
			countQuery += " WHERE " + bq.Where
		}
		_ = db.QueryRowContext(ctx, countQuery).Scan(&totalCount)
		fmt.Printf("[Engine.Query] TotalCount query took %v\n", time.Since(countStart))
	} else {
		fmt.Printf("[Engine.Query] TotalCount SKIPPED\n")
	}
	last = time.Now()

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()
	fmt.Printf("[Engine.Query] Main Query Exec took %v\n", time.Since(last))
	last = time.Now()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("error getting columns: %w", err)
	}

	resp := &QueryResult{
		Columns:    columns,
		TotalCount: totalCount,
		SQL:        query,
		Values:     make([][]interface{}, 0),
	}

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}

		rowData := make([]interface{}, len(columns))
		for i := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				rowData[i] = string(b)
			} else {
				rowData[i] = val
			}
		}
		resp.Values = append(resp.Values, rowData)
	}
	fmt.Printf("[Engine.Query] Row Scan took %v\n", time.Since(last))
	fmt.Printf("[Engine.Query] TOTAL DURATION: %v\n", time.Since(start))

	return resp, nil
}
