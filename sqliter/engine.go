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
		// Just return existing connection.
		// Aggressive Pinging here can cause "database is locked" errors to look like connectivity errors,
		// triggering a Close() which kills other active queries (like our stream!).
		return db, nil
	}

	// Open new connection with WAL mode enabled for better concurrency
	// Note: modernc.org/sqlite registers as "sqlite"
	// Increase busy_timeout to reduce "database is locked" errors
	dsn := fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=cache_size(10000)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	// Set some reasonable defaults for the pool
	// modernc_sqlite handles concurrency, but let's keep it reasonable per connection object
	db.SetMaxOpenConns(1) // SQLite is single-writer. Unbounded read connections can sometimes cause issues.
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0) // Keep connections alive indefinitely to avoid "database is closed" during long streams

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

	// Optimizations handled in DSN
	// last = time.Now()

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

type QueryResultChunk struct {
	Columns    []string        `json:"columns,omitempty"`
	Values     [][]interface{} `json:"values"`
	TotalCount int             `json:"totalCount,omitempty"`
	SQL        string          `json:"sql,omitempty"`
	Error      string          `json:"error,omitempty"`
}

// QueryStream executes a query and calls key callbacks during execution.
func (e *Engine) QueryStream(ctx context.Context, opts QueryOptions, onChunk func(QueryResultChunk)) error {
	start := time.Now()

	// --- 1. Query Preparation (Identical to Query) ---
	bq, err := banquet.ParseNested(opts.BanquetPath)
	if err != nil {
		return fmt.Errorf("error parsing URL: %w", err)
	}

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
			return fmt.Errorf("error building filter: %w", err)
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
		return fmt.Errorf("invalid path")
	}

	dbPath := filepath.Join(e.config.ServeFolder, dataSetPath)

	db, err := e.getDBConnection(ctx, dbPath)
	if err != nil {
		return fmt.Errorf("error opening DB: %w", err)
	}

	// Optimizations handled in DSN

	// Handle case where table name is missing
	if bq.Table == "" {
		rows, err := db.QueryContext(ctx, "SELECT name FROM sqlite_master WHERE type IN ('table', 'view') ORDER BY name")
		if err != nil {
			return fmt.Errorf("failed to list tables: %w", err)
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
			return fmt.Errorf("no tables found in database")
		} else {
			return fmt.Errorf("table name required. Available tables: %s", strings.Join(tables, ", "))
		}
	}

	query := sqlite.Compose(bq)

	// --- 2. Get Total Count (Optional) ---
	var totalCount int = -1
	if !opts.SkipTotalCount {
		countStart := time.Now()
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", sqlite.QuoteIdentifier(bq.Table))
		if bq.Where != "" {
			countQuery += " WHERE " + bq.Where
		}
		_ = db.QueryRowContext(ctx, countQuery).Scan(&totalCount)
		fmt.Printf("[Engine.QueryStream] TotalCount took %v\n", time.Since(countStart))
	}

	// --- 3. Execute Main Query ---
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("error getting columns: %w", err)
	}

	// --- 4. Send Initial Metadata Chunk ---
	onChunk(QueryResultChunk{
		Columns:    columns,
		TotalCount: totalCount,
		SQL:        query,
		Values:     [][]interface{}{}, // Empty values for first chunk
	})

	// --- 5. Stream Rows ---
	batchSize := 1000
	buffer := make([][]interface{}, 0, batchSize)

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	rowCount := 0
	lastEmit := time.Now()

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
		buffer = append(buffer, rowData)
		rowCount++

		// Emit chunk if full or if time has passed (streaming responsiveness)
		if len(buffer) >= batchSize || time.Since(lastEmit) > 100*time.Millisecond {
			onChunk(QueryResultChunk{
				Values: buffer,
			})
			buffer = make([][]interface{}, 0, batchSize) // Alloc new slice, let GC handle old one
			lastEmit = time.Now()
		}
	}

	// Flush remaining
	if len(buffer) > 0 {
		onChunk(QueryResultChunk{
			Values: buffer,
		})
	}

	fmt.Printf("[Engine.QueryStream] Finished. %d rows in %v\n", rowCount, time.Since(start))
	return nil
}
