package sqliter

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/darianmavgo/banquet"
	_ "modernc.org/sqlite"
)

type OldResponse struct {
	Rows       []map[string]interface{} `json:"rows"`
	TotalCount int                      `json:"totalCount"`
	Columns    []string                 `json:"columns"`
	Sql        string                   `json:"sql"`
	Error      string                   `json:"error,omitempty"`
	Banquet    *banquet.Banquet         `json:"banquet,omitempty"`
}

type OldLogEntry struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

// StartOldServer starts the deprecated server implementation.
func StartOldServer() {
	// Serve React Static Files
	cwd, _ := os.Getwd()
	distPath := filepath.Join(cwd, "../react-client/dist")

	// DIAGNOSTIC LOGGING
	log.Printf("Current Working Directory: %s", cwd)
	log.Printf("Calculated Dist Path: %s", distPath)

	if info, err := os.Stat(distPath); err != nil {
		log.Printf("ERROR: Dist path does not exist or cannot be accessed: %v", err)
	} else if !info.IsDir() {
		log.Printf("ERROR: Dist path is not a directory")
	} else {
		log.Printf("Dist path exists and is a directory")
		// List files in dist just to be sure
		files, _ := os.ReadDir(distPath)
		var fileNames []string
		for _, f := range files {
			fileNames = append(fileNames, f.Name())
		}
		log.Printf("Files in dist: %v", fileNames)
	}

	fs := http.FileServer(http.Dir(distPath))

	// Wrap handler to log requests to root
	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[STATIC] Request: %s", r.URL.Path)
		fs.ServeHTTP(w, r)
	}))
	mux.HandleFunc("/sqliter/rows", enableCorsOld(handleRowsOld))
	mux.HandleFunc("/sqliter/tables", enableCorsOld(handleTablesOld))
	mux.HandleFunc("/sqliter/fs", enableCorsOld(handleFSOld))
	mux.HandleFunc("/sqliter/logs", enableCorsOld(handleLogsOld))

	// NEW: Banquet Handler
	// We use /banquet/ as prefix to avoid conflict with React routes if we ever do wildcard serving
	mux.HandleFunc("/banquet/", enableCorsOld(handleBanquetOld))

	// Listen on IPv6 random high port
	listener, err := net.Listen("tcp6", "[::1]:0")
	if err != nil {
		listener, err = net.Listen("tcp6", ":0")
		if err != nil {
			log.Fatal("Failed to listen on IPv6:", err)
		}
	}

	port := listener.Addr().(*net.TCPAddr).Port
	url := fmt.Sprintf("http://[%s]:%d", "::1", port)

	log.Printf("Server starting on %s...", url)
	log.Printf("Serving static files from: %s", distPath)
	fmt.Printf("SERVING_AT=%s\n", url)

	if err := http.Serve(listener, mux); err != nil {
		log.Fatal(err)
	}
}

func enableCorsOld(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func handleLogsOld(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var entry OldLogEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		return // ignore bad logs
	}
	log.Printf("[CLIENT LOG] %s: %s", strings.ToUpper(entry.Level), entry.Message)
	w.WriteHeader(http.StatusOK)
}

func handleFSOld(w http.ResponseWriter, r *http.Request) {
	userHome, _ := os.UserHomeDir()
	docsDir := filepath.Join(userHome, "Documents") // Defaulting to Documents

	entries, err := os.ReadDir(docsDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var files []map[string]string
	for _, e := range entries {
		info, _ := e.Info()
		if !e.IsDir() && (strings.HasSuffix(e.Name(), ".sqlite") || strings.HasSuffix(e.Name(), ".db")) {
			files = append(files, map[string]string{
				"name": e.Name(),
				"type": "database",
				"size": humanizeBytesOld(info.Size()),
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

func humanizeBytesOld(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func handleTablesOld(w http.ResponseWriter, r *http.Request) {
	dbName := r.URL.Query().Get("db")
	if dbName == "" {
		http.Error(w, "Missing db param", http.StatusBadRequest)
		return
	}

	userHome, _ := os.UserHomeDir()
	dbPath := filepath.Join(userHome, "Documents", dbName)

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT name, type FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	var tables []map[string]string
	for rows.Next() {
		var name, type_ string
		rows.Scan(&name, &type_)
		tables = append(tables, map[string]string{"name": name, "type": type_})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tables)
}

func handleRowsOld(w http.ResponseWriter, r *http.Request) {
	// Keep existing for backward compat if needed, or redirect to Banquet?
	// For now, let's keep it but maybe users might switch to banquet.
	path := r.URL.Query().Get("path")
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")
	sortCol := r.URL.Query().Get("sortCol")
	sortDir := r.URL.Query().Get("sortDir")

	// path is like /DbName/TableName
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) < 2 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	dbName := parts[0]
	tableName := parts[1]

	userHome, _ := os.UserHomeDir()
	dbPath := filepath.Join(userHome, "Documents", dbName)

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	start, _ := strconv.Atoi(startStr)
	end, _ := strconv.Atoi(endStr)
	limit := end - start
	if limit < 0 {
		limit = 0
	}
	if limit == 0 && end == 0 {
		limit = 100
	} // Default

	query := fmt.Sprintf("SELECT * FROM \"%s\"", tableName)

	// Add sorting
	if sortCol != "" {
		// Whitelist checks could be here, but for local tool we might trust or just quote
		query += fmt.Sprintf(" ORDER BY \"%s\" %s", sortCol, sortDir)
	}

	query += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, start)

	rows, err := db.Query(query)
	if err != nil {
		resp := OldResponse{
			Error: err.Error(),
			Sql:   query,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer rows.Close()

	columns, _ := rows.Columns()

	var result []map[string]interface{}

	// Prepare for scanning generic rows
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		rows.Scan(valuePtrs...)
		entry := make(map[string]interface{})
		for i, col := range columns {
			var v interface{}
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				v = string(b)
			} else {
				v = val
			}
			entry[col] = v
		}
		result = append(result, entry)
	}

	// Total count (inefficient for large tables but standard for simple grid)
	var total int
	db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM \"%s\"", tableName)).Scan(&total)

	resp := OldResponse{
		Rows:       result,
		TotalCount: total,
		Columns:    columns,
		Sql:        query,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Handler for Banquet URLs: /banquet/<Path to DB>/<Table>?params...
func handleBanquetOld(w http.ResponseWriter, r *http.Request) {
	// 1. Parse URL
	reqURI := r.RequestURI
	// Strip /banquet/ prefix
	prefix := "/banquet/"
	if strings.HasPrefix(reqURI, prefix) {
		reqURI = reqURI[len(prefix):]
	}

	b, err := banquet.ParseNested(reqURI)
	if err != nil {
		json.NewEncoder(w).Encode(OldResponse{Error: "Invalid banquet URL: " + err.Error()})
		return
	}

	// 2. Resolve Local File
	userHome, _ := os.UserHomeDir()
	localFilePath := filepath.Join(userHome, "Documents", b.DataSetPath)

	// Check if exists
	info, err := os.Stat(localFilePath)
	if err != nil {
		json.NewEncoder(w).Encode(OldResponse{Error: "File not found: " + b.DataSetPath})
		return
	}

	// 3. Connect DB
	dbPath := localFilePath
	if info.IsDir() {
		// Only support index.sqlite inside dir
		dbPath = filepath.Join(localFilePath, "index.sqlite")
		b.Table = "tb0"
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		json.NewEncoder(w).Encode(OldResponse{Error: "Failed to open DB: " + err.Error()})
		return
	}
	defer db.Close()

	// 4. Infer Table
	if b.Table == "" {
		func() {
			rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")
			if err != nil {
				return
			}
			defer rows.Close()

			var tables []string
			for rows.Next() {
				var name string
				if err := rows.Scan(&name); err == nil {
					tables = append(tables, name)
				}
			}

			if len(tables) > 0 {
				hasTb0 := false
				for _, t := range tables {
					if t == "tb0" {
						hasTb0 = true
						break
					}
				}

				if hasTb0 {
					b.Table = "tb0"
				} else if len(tables) == 1 {
					b.Table = tables[0]
				} else {
					b.Table = "sqlite_master"
				}
			}
		}()
	}

	// 5. Build Sql
	query := ConstructSQL(b)

	// 6. Execute
	rows, err := db.Query(query)
	if err != nil {
		resp := OldResponse{
			Error:   err.Error(),
			Sql:     query,
			Banquet: b,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer rows.Close()

	columns, _ := rows.Columns()
	var result []map[string]interface{}
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		rows.Scan(valuePtrs...)
		entry := make(map[string]interface{})
		for i, col := range columns {
			var v interface{}
			val := values[i]
			bVal, ok := val.([]byte)
			if ok {
				v = string(bVal)
			} else {
				v = val
			}
			entry[col] = v
		}
		result = append(result, entry)
	}

	// Count is tricky with filters. For now, separate count query if feasible, or just return what we have.
	total := 0
	if b.Where == "" {
		db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", QuoteIdentifier(b.Table))).Scan(&total)
	}

	resp := OldResponse{
		Rows:       result,
		Columns:    columns,
		TotalCount: total,
		Sql:        query,
		Banquet:    b,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
