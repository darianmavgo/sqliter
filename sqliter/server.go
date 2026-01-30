package sqliter

import (
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/darianmavgo/banquet"
	_ "modernc.org/sqlite"
)

//go:embed ui/*
var uiFS embed.FS

type Server struct {
	config *Config
}

func NewServer(cfg *Config) *Server {
	return &Server{
		config: cfg,
	}
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1. API Handling
	if strings.HasPrefix(r.URL.Path, "/sqliter/") {
		s.handleAPI(w, r)
		return
	}

	// 2. Serve Static Assets (React App)
	// We want to serve files from "ui" directory in the embedded FS.
	distFS, err := fs.Sub(uiFS, "ui")
	if err != nil {
		s.logError("Failed to get dist FS: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Clean path to prevent .. traversal (though fs.Open handles this)
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}

	// Try to open the file
	f, err := distFS.Open(path)
	if err == nil {
		// File exists, serve it
		defer f.Close()
		stat, _ := f.Stat()
		http.ServeContent(w, r, path, stat.ModTime(), f.(io.ReadSeeker))
		return
	}

	// 3. SPA Fallback: If not found and not an API call, serve index.html
	// This allows React Router to handle /mydb.db/table path
	index, err := distFS.Open("index.html")
	if err != nil {
		http.Error(w, "UI not found. Please run build.", http.StatusNotFound)
		return
	}
	defer index.Close()
	stat, _ := index.Stat()
	http.ServeContent(w, r, "index.html", stat.ModTime(), index.(io.ReadSeeker))
}

func (s *Server) handleAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*") // For development

	if strings.HasPrefix(r.URL.Path, "/sqliter/fs") {
		s.apiListFiles(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/sqliter/tables") {
		s.apiListTables(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/sqliter/rows") {
		s.apiQueryTable(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/sqliter/logs") {
		s.handleClientLogs(w, r)
		return
	}
	http.Error(w, "Not found", http.StatusNotFound)
}

func (s *Server) handleClientLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Level   string      `json:"level"`
		Message interface{} `json:"message"` // Helper to accept strings or objects
	}

	// Limit body size to prevent abuse
	r.Body = http.MaxBytesReader(w, r.Body, 1024*10) // 10KB max log

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		// Silently fail or log distinct error
		return
	}

	// Format: [CLIENT] [LEVEL] Message
	// We use standard log.Printf which goes to stderr/stdout
	log.Printf("[CLIENT] [%s] %v", strings.ToUpper(payload.Level), payload.Message)

	w.WriteHeader(http.StatusOK)
}

func (s *Server) apiListFiles(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(s.config.ServeFolder)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%v"}`, err), http.StatusInternalServerError)
		return
	}

	type FileEntry struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	var files []FileEntry

	for _, entry := range entries {
		if !entry.IsDir() && (strings.HasSuffix(entry.Name(), ".db") || strings.HasSuffix(entry.Name(), ".sqlite") || strings.HasSuffix(entry.Name(), ".csv.db") || strings.HasSuffix(entry.Name(), ".xlsx.db")) {
			files = append(files, FileEntry{Name: entry.Name(), Type: "database"})
		}
	}
	json.NewEncoder(w).Encode(files)
}

func (s *Server) apiListTables(w http.ResponseWriter, r *http.Request) {
	dbName := r.URL.Query().Get("db")
	if dbName == "" {
		http.Error(w, `{"error": "db parameter required"}`, http.StatusBadRequest)
		return
	}

	if strings.Contains(dbName, "..") {
		http.Error(w, `{"error": "Invalid path"}`, http.StatusBadRequest)
		return
	}

	dbPath := filepath.Join(s.config.ServeFolder, dbName)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Error opening DB: %v"}`, err), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT name, type FROM sqlite_master WHERE type IN ('table', 'view') ORDER BY name")
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Database error: %v"}`, err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type TableInfo struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}

	var tables []TableInfo
	for rows.Next() {
		var name, type_ string
		if err := rows.Scan(&name, &type_); err != nil {
			continue
		}
		tables = append(tables, TableInfo{Name: name, Type: type_})
	}
	json.NewEncoder(w).Encode(tables)
}

func (s *Server) apiQueryTable(w http.ResponseWriter, r *http.Request) {
	// Expecting 'path' parameter which is a Banquet URL, OR separate db/table/params
	// For simplicity and alignment with the plan, let's look for 'path' or construct it.

	// If 'path' is provided, parse it.
	path := r.URL.Query().Get("path")
	if path == "" {
		// Fallback: try to construct from db/table params for basic usage
		db := r.URL.Query().Get("db")
		table := r.URL.Query().Get("table")
		if db != "" && table != "" {
			path = "/" + db + "/" + table
		} else {
			http.Error(w, `{"error": "path or db+table parameters required"}`, http.StatusBadRequest)
			return
		}
	}

	// Append standard grid params if not already in path
	// AgGrid sends: start, end, sortCol, sortDir
	qs := r.URL.Query()
	start := qs.Get("start")
	end := qs.Get("end")
	sortCol := qs.Get("sortCol")
	sortDir := qs.Get("sortDir")

	// We can rely on Banquet parsing, but we might need to inject these if they are separate.
	// simpler to just re-parse the path and then override specifics if provided.

	bq, err := banquet.ParseNested(path)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Error parsing URL: %v"}`, err), http.StatusBadRequest)
		return
	}

	// Override limit/offset if provided by AgGrid params
	if start != "" && end != "" {
		sIdx, _ := strconv.Atoi(start)
		eIdx, _ := strconv.Atoi(end)
		limit := eIdx - sIdx
		bq.Limit = fmt.Sprintf("%d", limit)
		bq.Offset = fmt.Sprintf("%d", sIdx)
	}

	if sortCol != "" {
		bq.OrderBy = sortCol
		if sortDir != "" {
			bq.SortDirection = sortDir
		}
	}

	dataSetPath := strings.TrimPrefix(bq.DataSetPath, "/")
	if strings.Contains(dataSetPath, "..") {
		http.Error(w, `{"error": "Invalid path"}`, http.StatusBadRequest)
		return
	}

	dbPath := filepath.Join(s.config.ServeFolder, dataSetPath)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Error opening DB: %v"}`, err), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	if _, err := db.Exec("PRAGMA page_size = 65536; PRAGMA cache_size = -2000;"); err != nil {
		s.logError("Error setting PRAGMAs: %v", err)
	}

	query := ConstructSQL(bq)

	// Get total count for pagination (optional, but good for AgGrid infinite model)
	// We'll do a separate count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", bq.Table)
	if bq.Where != "" {
		countQuery += " WHERE " + bq.Where
	}
	var totalCount int
	_ = db.QueryRow(countQuery).Scan(&totalCount)

	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Query error: %v", "sql": "%s"}`, err, query), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Error getting columns: %v"}`, err), http.StatusInternalServerError)
		return
	}

	// Result structure
	type APIResponse struct {
		Columns    []string                 `json:"columns"`
		Rows       []map[string]interface{} `json:"rows"`
		TotalCount int                      `json:"totalCount"`
		SQL        string                   `json:"sql"`
	}

	resp := APIResponse{
		Columns:    columns,
		TotalCount: totalCount,
		SQL:        query,
		Rows:       []map[string]interface{}{},
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

		rowMap := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				rowMap[col] = string(b)
			} else {
				rowMap[col] = val
			}
		}
		resp.Rows = append(resp.Rows, rowMap)
	}

	json.NewEncoder(w).Encode(resp)
}

func (s *Server) logError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log.Printf("[ERROR] %s", msg)

	// Try to log to a file
	logDir := s.config.LogDir
	if logDir == "" {
		logDir = "logs"
	}
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(logDir, "server_error.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		defer f.Close()
		log.New(f, "", log.LstdFlags).Println(msg)
	}
}
