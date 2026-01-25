package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/darianmavgo/banquet"
	"github.com/darianmavgo/sqliter/common"
	"github.com/darianmavgo/sqliter/sqliter"
	_ "github.com/mattn/go-sqlite3"
)

// Server handles serving SQLite files.
type Server struct {
	config      *sqliter.Config
	tableWriter *sqliter.TableWriter
}

// NewServer creates a new Server with the given configuration.
func NewServer(cfg *sqliter.Config) *Server {
	// Templates are now embedded and do not use the filesystem path from config
	t := sqliter.GetDefaultTemplates()
	return &Server{
		config:      cfg,
		tableWriter: sqliter.NewTableWriter(t, cfg),
	}
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	bq, err := banquet.ParseNested(r.URL.String())
	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing URL: %v", err), http.StatusBadRequest)
		return
	}

	title := strings.TrimPrefix(r.URL.Path, "/")
	dataSetPath := strings.TrimPrefix(bq.DataSetPath, "/")

	if dataSetPath == "" {
		s.listFiles(w, title)
		return
	}

	// Security check: simple directory traversal prevention
	if strings.Contains(dataSetPath, "..") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	dbPath := filepath.Join(s.config.DataDir, dataSetPath)
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		http.Error(w, "File not found: "+dataSetPath, http.StatusNotFound)
		return
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error opening DB: %v", err), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	if bq.Table == "sqlite_master" || bq.Table == "" {
		s.listTables(w, r, db, bq.DataSetPath, title)
		return
	}

	if r.Method == http.MethodPost {
		if !s.config.RowCRUD {
			http.Error(w, "Row CRUD is disabled", http.StatusForbidden)
			return
		}
		s.handleCRUD(w, r, db, bq)
		return
	}

	s.queryTable(w, db, bq, title)
}

func (s *Server) listFiles(w http.ResponseWriter, title string) {
	entries, err := os.ReadDir(s.config.DataDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading DataDir: %v", err), http.StatusInternalServerError)
		return
	}

	s.tableWriter.StartHTMLTable(w, []string{"Database"}, title)
	for i, entry := range entries {
		if !entry.IsDir() && (strings.HasSuffix(entry.Name(), ".db") || strings.HasSuffix(entry.Name(), ".sqlite") || strings.HasSuffix(entry.Name(), ".csv.db") || strings.HasSuffix(entry.Name(), ".xlsx.db")) {
			link := fmt.Sprintf("<a href='/%s'>%s</a>", entry.Name(), entry.Name())
			s.tableWriter.WriteHTMLRow(w, i, []string{link})
		}
	}
	s.tableWriter.EndHTMLTable(w)
}

func (s *Server) listTables(w http.ResponseWriter, r *http.Request, db *sql.DB, dbUrlPath string, title string) {
	rows, err := db.Query("SELECT name, type FROM sqlite_master WHERE type IN ('table', 'view') ORDER BY name")
	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Ensure absolute path
	if !strings.HasPrefix(dbUrlPath, "/") {
		dbUrlPath = "/" + dbUrlPath
	}

	type TableInfo struct {
		Name string
		Type string
	}

	var tables []TableInfo
	for rows.Next() {
		var name, type_ string
		if err := rows.Scan(&name, &type_); err != nil {
			continue
		}
		tables = append(tables, TableInfo{Name: name, Type: type_})
	}

	// Auto-redirect if enabled and only one table
	if s.config.AutoRedirectSingleTable && len(tables) == 1 {
		redirectUrl := dbUrlPath + "/" + tables[0].Name
		// Clean up double slashes just in case
		redirectUrl = strings.ReplaceAll(redirectUrl, "//", "/")
		http.Redirect(w, r, redirectUrl, http.StatusFound)
		return
	}

	s.tableWriter.StartHTMLTable(w, []string{"Table", "Type"}, title)
	for i, t := range tables {
		// Link format: /dbfile.db/tablename
		link := fmt.Sprintf("<a href='%s/%s'>%s</a>", dbUrlPath, t.Name, t.Name)
		s.tableWriter.WriteHTMLRow(w, i, []string{link, t.Type})
	}
	s.tableWriter.EndHTMLTable(w)
}

func (s *Server) log(format string, args ...interface{}) {
	if s.config.Verbose {
		log.Printf(format, args...)
	}
}

func (s *Server) queryTable(w http.ResponseWriter, db *sql.DB, bq *banquet.Banquet, title string) {
	if s.config.RowCRUD {
		s.tableWriter.EnableEditable(true)
	}
	query := common.ConstructSQL(bq)
	s.log("Executing query: %s", query)

	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Query error: %v\nQuery: %s", err, query), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting columns: %v", err), http.StatusInternalServerError)
		return
	}

	s.tableWriter.StartHTMLTable(w, columns, title)

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	var i int
	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			s.log("Error scanning row: %v", err)
			continue
		}

		strValues := make([]string, len(columns))
		for i, val := range values {
			if val == nil {
				strValues[i] = "NULL"
			} else {
				strValues[i] = fmt.Sprintf("%v", val)
			}
		}

		s.tableWriter.WriteHTMLRow(w, i, strValues)
		i++
	}

	s.tableWriter.EndHTMLTable(w)
}

func (s *Server) handleCRUD(w http.ResponseWriter, r *http.Request, db *sql.DB, bq *banquet.Banquet) {
	var payload struct {
		Action string                 `json:"action"`
		Data   map[string]interface{} `json:"data"`
		Where  map[string]interface{} `json:"where"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.logError("Error decoding JSON: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var query string
	var args []interface{}

	switch payload.Action {
	case "create":
		query, args = common.ConstructInsert(bq.Table, payload.Data)
	case "update":
		query, args = common.ConstructUpdate(bq.Table, payload.Data, payload.Where)
	case "delete":
		query, args = common.ConstructDelete(bq.Table, payload.Where)
	default:
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}

	s.log("Executing CRUD %s: %s", payload.Action, query)

	if _, err := db.Exec(query, args...); err != nil {
		s.logError("Error executing CRUD: %v", err)
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
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
