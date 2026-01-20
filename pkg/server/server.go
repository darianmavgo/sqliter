package server

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/darianmavgo/banquet"
	_ "github.com/mattn/go-sqlite3"
	"github.com/darianmavgo/sqliter/pkg/common"
	"github.com/darianmavgo/sqliter/sqliter"
)

// Server handles serving SQLite files.
type Server struct {
	config      *sqliter.Config
	tableWriter *sqliter.TableWriter
}

// NewServer creates a new Server with the given configuration.
func NewServer(cfg *sqliter.Config) *Server {
	t := sqliter.LoadTemplates(cfg.TemplateDir)
	return &Server{
		config:      cfg,
		tableWriter: sqliter.NewTableWriter(t),
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

	// Security check: simple directory traversal prevention
	if strings.Contains(dataSetPath, "..") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	if dataSetPath == "" {
		s.listFiles(w, "")
		return
	}

	fullPath := filepath.Join(s.config.DataDir, dataSetPath)
	info, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		http.Error(w, "File not found: "+dataSetPath, http.StatusNotFound)
		return
	}

	if info.IsDir() {
		s.listFiles(w, dataSetPath)
		return
	}

	db, err := sql.Open("sqlite3", fullPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error opening DB: %v", err), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	if bq.Table == "sqlite_master" || bq.Table == "" {
		s.listTables(w, db, bq.DataSetPath)
	} else {
		s.queryTable(w, db, bq)
	}
}

func (s *Server) listFiles(w http.ResponseWriter, dirPath string) {
	fullPath := filepath.Join(s.config.DataDir, dirPath)
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading directory: %v", err), http.StatusInternalServerError)
		return
	}

	// Normalize dirPath for links
	prefix := "/"
	if dirPath != "" {
		prefix = "/" + dirPath + "/"
	}

	sqliter.StartTableList(w)
	for _, entry := range entries {
		if entry.IsDir() {
			sqliter.WriteTableLink(w, entry.Name()+"/", prefix+entry.Name())
		} else if strings.HasSuffix(entry.Name(), ".db") || strings.HasSuffix(entry.Name(), ".sqlite") || strings.HasSuffix(entry.Name(), ".csv.db") || strings.HasSuffix(entry.Name(), ".xlsx.db") {
			sqliter.WriteTableLink(w, entry.Name(), prefix+entry.Name())
		}
	}
	sqliter.EndTableList(w)
}

func (s *Server) listTables(w http.ResponseWriter, db *sql.DB, dbUrlPath string) {
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Ensure absolute path
	if !strings.HasPrefix(dbUrlPath, "/") {
		dbUrlPath = "/" + dbUrlPath
	}

	sqliter.StartTableList(w)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		// Link format: /dbfile.db/tablename
		sqliter.WriteTableLink(w, name, dbUrlPath+"/"+name)
	}
	sqliter.EndTableList(w)
}

func (s *Server) queryTable(w http.ResponseWriter, db *sql.DB, bq *banquet.Banquet) {
	query := common.ConstructSQL(bq)
	log.Printf("Executing query: %s", query)

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

	s.tableWriter.StartHTMLTable(w, columns)

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			log.Println("Error scanning row:", err)
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

		s.tableWriter.WriteHTMLRow(w, strValues)
	}

	s.tableWriter.EndHTMLTable(w)
}
