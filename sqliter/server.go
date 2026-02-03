package sqliter

import (
	"bytes"
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

	_ "modernc.org/sqlite"
)

//go:embed ui/*
var uiFS embed.FS

type Server struct {
	config *Config
	engine *Engine
}

func NewServer(cfg *Config) *Server {
	return &Server{
		config: cfg,
		engine: NewEngine(cfg),
	}
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1. API Handling
	if strings.HasPrefix(r.URL.Path, "/sqliter/") {
		s.handleAPI(w, r)
		return
	}

	if r.URL.Path == "/" || r.URL.Path == "" {
		abs, _ := filepath.Abs(s.config.ServeFolder)
		log.Printf("[SERVER] Root accessed. Serving data from: %s", abs)
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

	// Check if we are serving index.html (either explicitly or via fallback)
	f, err := distFS.Open(path)
	if err == nil {
		// File exists
		defer f.Close()
		stat, _ := f.Stat()

		// If it's index.html, we need to inject config
		if path == "index.html" {
			s.serveIndexWithConfig(w, r, f, stat)
			return
		}

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
	s.serveIndexWithConfig(w, r, index, stat)
}

func (s *Server) serveIndexWithConfig(w http.ResponseWriter, r *http.Request, f fs.File, stat fs.FileInfo) {
	// Read full content
	content, err := io.ReadAll(f)
	if err != nil {
		http.Error(w, "Failed to read index.html", http.StatusInternalServerError)
		return
	}

	// Inject Config
	// Look for <head> to insert script
	htmlStr := string(content)
	injection := fmt.Sprintf("<script>window.SQLITER_CONFIG = { basePath: %q };</script>", s.config.BaseURL)

	// Prepend to <head> or <body>, or just append to head if found, else prepend to content
	if strings.Contains(htmlStr, "<head>") {
		htmlStr = strings.Replace(htmlStr, "<head>", "<head>"+injection, 1)
	} else {
		htmlStr = injection + htmlStr
	}

	// Verify we haven't messed up encoding (assuming UTF-8 for web)
	data := []byte(htmlStr)

	// Serve with ModTime from original file to respect some Caching logic,
	// though Content-Length will change so ServeContent handles that.
	http.ServeContent(w, r, "index.html", stat.ModTime(), bytes.NewReader(data))
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
	dir := r.URL.Query().Get("dir")
	files, err := s.engine.ListFiles(r.Context(), dir)
	if err != nil {
		s.writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

func (s *Server) apiListTables(w http.ResponseWriter, r *http.Request) {
	dbName := r.URL.Query().Get("db")
	if dbName == "" {
		http.Error(w, `{"error": "db parameter required"}`, http.StatusBadRequest)
		return
	}

	tables, err := s.engine.ListTables(r.Context(), dbName)
	if err != nil {
		s.writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type TableListResponse struct {
		Tables                  []TableInfo `json:"tables"`
		AutoRedirectSingleTable bool        `json:"autoRedirectSingleTable"`
	}

	json.NewEncoder(w).Encode(TableListResponse{
		Tables:                  tables,
		AutoRedirectSingleTable: s.config.AutoRedirectSingleTable,
	})
}

func (s *Server) apiQueryTable(w http.ResponseWriter, r *http.Request) {
	// Expecting 'path' parameter which is a Banquet URL, OR separate db/table/params
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

	opts := QueryOptions{
		BanquetPath:   path,
		AllowOverride: true,
	}

	qs := r.URL.Query()
	start := qs.Get("start")
	end := qs.Get("end")
	sortCol := qs.Get("sortCol")
	sortDir := qs.Get("sortDir")

	// Check for skipTotalCount
	if strings.ToLower(qs.Get("skipTotalCount")) == "true" {
		opts.SkipTotalCount = true
	}

	// Override limit/offset if provided by AgGrid params
	if start != "" && end != "" {
		sIdx, _ := strconv.Atoi(start)
		eIdx, _ := strconv.Atoi(end)
		limit := eIdx - sIdx
		opts.Limit = limit
		opts.Offset = sIdx
		if limit == 0 {
			opts.ForceZeroLimit = true
		}
	}

	if sortCol != "" {
		opts.SortCol = sortCol
		if sortDir != "" {
			opts.SortDir = sortDir
		}
	}

	// AgGrid Filter Model
	filterModel := r.URL.Query().Get("filterModel")
	if filterModel != "" {
		filterWhere, err := BuildWhereClause(filterModel)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "Error parsing filter: %v"}`, err), http.StatusBadRequest)
			return
		}
		opts.FilterWhere = filterWhere
	}

	result, err := s.engine.Query(r.Context(), opts)
	if err != nil {
		s.writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(result)
}

func (s *Server) writeJSONError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
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
