package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/darianmavgo/banquet"
	"github.com/darianmavgo/sqliter/pkg/common"
	view "github.com/darianmavgo/sqliter/sqliter"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("sqlite3", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Initialize Database
	initDB(db)

	http.HandleFunc("/", handler)
	log.Println("Serving in-memory database at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func initDB(db *sql.DB) {
	_, err := db.Exec(`
		CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER);
		INSERT INTO users (name, age) VALUES ('Alice', 30);
		INSERT INTO users (name, age) VALUES ('Bob', 25);
		INSERT INTO users (name, age) VALUES ('Charlie', 35);
		INSERT INTO users (name, age) VALUES ('Diana', 28);

		CREATE TABLE products (id INTEGER PRIMARY KEY, name TEXT, price REAL);
		INSERT INTO products (name, price) VALUES ('Laptop', 999.99);
		INSERT INTO products (name, price) VALUES ('Mouse', 19.99);
		INSERT INTO products (name, price) VALUES ('Keyboard', 49.99);
	`)
	if err != nil {
		log.Fatal("Failed to init DB:", err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	// Parse URL using Banquet
	// ParseNested expects the URL string
	bq, err := banquet.ParseNested(r.URL.String())
	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing URL: %v", err), http.StatusBadRequest)
		return
	}

	// If no table is specified, list tables
	// banquet sometimes returns empty table if parsing fails to find one
	if bq.Table == "" {
		listTables(w, r)
		return
	}

	// Query the table
	queryTable(w, bq)
}

func listTables(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		tables = append(tables, name)
	}

	// Simulate AutoRedirectSingleTable = true for demo
	if len(tables) == 1 {
		http.Redirect(w, r, "/"+tables[0], http.StatusFound)
		return
	}

	view.StartTableList(w)
	for _, name := range tables {
		// Link to the table.
		// If we are at root, link is /name
		view.WriteTableLink(w, name, "/"+name)
	}
	view.EndTableList(w)
}

func queryTable(w http.ResponseWriter, bq *banquet.Banquet) {
	// Construct SQL
	query := common.ConstructSQL(bq)

	// Debug logging
	common.DebugLog(bq, query)

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

	w.Header().Set("X-Banquet", common.GetBanquetJSON(bq))
	w.Header().Set("X-Query", query)

	view.StartHTMLTable(w, columns)

	// Prepare a slice of interface{} to hold values
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

		// Convert to strings for view
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
