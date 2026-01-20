package main

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
	"mavgo-flight/pkg/common"
	view "mavgo-flight/sqliter"
)

func main() {
	http.HandleFunc("/", handler)
	log.Println("Serving local sqlite files from sample_data at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	bq, err := banquet.ParseNested(r.URL.String())
	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing URL: %v", err), http.StatusBadRequest)
		return
	}

	dataSetPath := strings.TrimPrefix(bq.DataSetPath, "/")

	if dataSetPath == "" {
		listFiles(w)
		return
	}

	// Security check: simple directory traversal prevention
	if strings.Contains(dataSetPath, "..") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	dbPath := filepath.Join("sample_data", dataSetPath)
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
		listTables(w, db, bq.DataSetPath)
	} else {
		queryTable(w, db, bq)
	}
}

func listFiles(w http.ResponseWriter) {
	entries, err := os.ReadDir("sample_data")
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading sample_data: %v", err), http.StatusInternalServerError)
		return
	}

	view.StartTableList(w)
	for _, entry := range entries {
		if !entry.IsDir() && (strings.HasSuffix(entry.Name(), ".db") || strings.HasSuffix(entry.Name(), ".sqlite") || strings.HasSuffix(entry.Name(), ".csv.db") || strings.HasSuffix(entry.Name(), ".xlsx.db")) {
			view.WriteTableLink(w, entry.Name(), "/"+entry.Name())
		}
	}
	view.EndTableList(w)
}

func listTables(w http.ResponseWriter, db *sql.DB, dbUrlPath string) {
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

	view.StartTableList(w)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		// Link format: /dbfile.db/tablename
		view.WriteTableLink(w, name, dbUrlPath+"/"+name)
	}
	view.EndTableList(w)
}

func queryTable(w http.ResponseWriter, db *sql.DB, bq *banquet.Banquet) {
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

	view.StartHTMLTable(w, columns)

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

		view.WriteHTMLRow(w, strValues)
	}

	view.EndHTMLTable(w)
}
