package sqliter

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/darianmavgo/banquet"
	"github.com/darianmavgo/banquet/sqlite"
)

// Engine handles the core logic, agnostic of HTTP or Wails
type Engine struct {
	config  *Config
	dbCache map[string]*sql.DB
	dbMutex sync.Mutex
}

func NewEngine(cfg *Config) *Engine {
	return &Engine{
		config:  cfg,
		dbCache: make(map[string]*sql.DB),
	}
}

// getDB returns a cached database connection or opens a new one
func (e *Engine) getDB(path string) (*sql.DB, error) {
	e.dbMutex.Lock()
	defer e.dbMutex.Unlock()

	if e.dbCache == nil {
		e.dbCache = make(map[string]*sql.DB)
	}

	if db, ok := e.dbCache[path]; ok {
		return db, nil
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	// Limit to 1 connection to prevent locking issues and ensure consistent state
	db.SetMaxOpenConns(1)

	// Apply PRAGMAs
	// Note: With SetMaxOpenConns(1), this is safe as there is only one connection.
	if _, err := db.Exec("PRAGMA page_size = 65536; PRAGMA cache_size = -2000; PRAGMA case_sensitive_like = OFF;"); err != nil {
		// Log error if possible, but don't fail hard
		fmt.Printf("Error setting PRAGMA: %v\n", err)
	}

	e.dbCache[path] = db
	return db, nil
}

type FileEntry struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// ListFiles returns a list of files in a directory (safe, strict relative paths)
func (e *Engine) ListFiles(dirRelPath string) ([]FileEntry, error) {
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

func (e *Engine) ListTables(dbRelPath string) ([]TableInfo, error) {
	if strings.Contains(dbRelPath, "..") {
		return nil, fmt.Errorf("invalid path")
	}

	dbPath := filepath.Join(e.config.ServeFolder, dbRelPath)
	db, err := e.getDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("error opening DB: %w", err)
	}
	// db is cached, do not close

	rows, err := db.Query("SELECT name, type FROM sqlite_master WHERE type IN ('table', 'view') ORDER BY name")
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

func (e *Engine) Query(opts QueryOptions) (*QueryResult, error) {
	bq, err := banquet.ParseNested(opts.BanquetPath)
	if err != nil {
		return nil, fmt.Errorf("error parsing URL: %w", err)
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
	db, err := e.getDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("error opening DB: %w", err)
	}
	// db is cached, do not close

	// Handle case where table name is missing
	if bq.Table == "" {
		rows, err := db.Query("SELECT name FROM sqlite_master WHERE type IN ('table', 'view') ORDER BY name")
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
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", sqlite.QuoteIdentifier(bq.Table))
		if bq.Where != "" {
			countQuery += " WHERE " + bq.Where
		}
		_ = db.QueryRow(countQuery).Scan(&totalCount)
	}

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

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

	return resp, nil
}
