package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/darianmavgo/banquet"
	"github.com/darianmavgo/sqliter/common"
	"github.com/darianmavgo/sqliter/sqliter"
	_ "modernc.org/sqlite"
)

type Server struct {
	config *sqliter.Config
}

func NewServer(cfg *sqliter.Config) *Server {
	return &Server{
		config: cfg,
	}
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	bq, err := banquet.ParseNested(r.URL.String())
	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing URL: %v", err), http.StatusBadRequest)
		return
	}

	dataSetPath := strings.TrimPrefix(bq.DataSetPath, "/")
	title := filepath.Base(dataSetPath)
	if dataSetPath == "" || title == "." {
		title = strings.TrimPrefix(r.URL.Path, "/")
	}

	tw := sqliter.NewTableWriter(sqliter.GetDefaultTemplates(), s.config)

	if dataSetPath == "" {
		s.listFiles(w, tw, title)
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

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error opening DB: %v", err), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	if bq.Table == "sqlite_master" || bq.Table == "" {
		s.listTables(w, r, db, tw, bq.DataSetPath, title)
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

	s.queryTable(w, db, bq, tw, title)
}

func (s *Server) listFiles(w http.ResponseWriter, tw *sqliter.TableWriter, title string) {
	entries, err := os.ReadDir(s.config.DataDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading DataDir: %v", err), http.StatusInternalServerError)
		return
	}

	tw.StartHTMLTable(w, []string{"Database"}, title)
	for i, entry := range entries {
		if !entry.IsDir() && (strings.HasSuffix(entry.Name(), ".db") || strings.HasSuffix(entry.Name(), ".sqlite") || strings.HasSuffix(entry.Name(), ".csv.db") || strings.HasSuffix(entry.Name(), ".xlsx.db")) {
			link := fmt.Sprintf("<a href='/%s'>%s</a>", entry.Name(), entry.Name())
			tw.WriteHTMLRow(w, i, []string{link})
		}
	}
	tw.EndHTMLTable(w)
}

func (s *Server) listTables(w http.ResponseWriter, r *http.Request, db *sql.DB, tw *sqliter.TableWriter, dbUrlPath string, title string) {
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

	tw.StartHTMLTable(w, []string{"Table", "Type"}, title)
	for i, t := range tables {
		// Link format: /dbfile.db/tablename
		link := fmt.Sprintf("<a href='%s/%s'>%s</a>", dbUrlPath, t.Name, t.Name)
		tw.WriteHTMLRow(w, i, []string{link, t.Type})
	}
	tw.EndHTMLTable(w)
}

func (s *Server) log(format string, args ...interface{}) {
	if s.config.Verbose {
		log.Printf(format, args...)
	}
}

func (s *Server) queryTable(w http.ResponseWriter, db *sql.DB, bq *banquet.Banquet, tw *sqliter.TableWriter, title string) {
	editable := s.config.RowCRUD
	sticky := s.config.StickyHeader

	if bq.Table == "tb0" {
		editable = false
		sticky = false
	}

	tw.EnableEditable(editable)
	tw.SetStickyHeader(sticky)
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

	tw.StartHTMLTable(w, columns, title)

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	var rowIdx int
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

		if err := tw.WriteHTMLRow(w, rowIdx, strValues); err != nil {
			// Check for broken pipe (client disconnected)
			if strings.Contains(err.Error(), "broken pipe") {
				// Stop processing silentely or with a single debug log
				// s.log("Client disconnected (broken pipe), stopping response.")
				return
			}
			s.logError("Error writing HTML row: %v", err)
			return
		}
		rowIdx++

		// Flush every 100 rows to keep browser responsive
		if rowIdx%100 == 0 {
			flush(w)
		}
	}

	tw.EndHTMLTable(w)
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

func flush(w io.Writer) {
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}
