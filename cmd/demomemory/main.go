package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/darianmavgo/banquet"
	"github.com/darianmavgo/sqliter/common"
	"github.com/darianmavgo/sqliter/sqliter"
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

	// Serve static files for the theme
	http.Handle("/style1/", http.StripPrefix("/style1/", http.FileServer(http.Dir("themes/style1"))))

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

		CREATE VIEW user_names AS SELECT name FROM users;
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

	// If it's a POST request, handle CRUD
	if r.Method == http.MethodPost {
		handleCRUD(w, r, bq)
		return
	}

	// Query the table
	queryTable(w, bq)
}

func listTables(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT name, type FROM sqlite_master WHERE type IN ('table', 'view') ORDER BY name")
	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

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

	// Simulate AutoRedirectSingleTable = true for demo
	if len(tables) == 1 {
		http.Redirect(w, r, "/"+tables[0].Name, http.StatusFound)
		return
	}

	sqliter.StartHTMLTable(w, []string{"Table", "Type"}, "Database")
	for i, t := range tables {
		// Link format: /tablename
		link := fmt.Sprintf("<a href='/%s'>%s</a>", t.Name, t.Name)
		sqliter.WriteHTMLRow(w, i, []string{link, t.Type})
	}
	sqliter.EndHTMLTable(w)
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

	tw := sqliter.NewTableWriter(sqliter.GetDefaultTemplates(), sqliter.DefaultConfig())
	tw.EnableEditable(true)
	tw.StartHTMLTable(w, columns, bq.Table)

	// Prepare a slice of interface{} to hold values
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	var rowIdx int
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

		tw.WriteHTMLRow(w, rowIdx, strValues)
		rowIdx++
	}

	tw.EndHTMLTable(w)
}

func handleCRUD(w http.ResponseWriter, r *http.Request, bq *banquet.Banquet) {
	var payload struct {
		Action string                 `json:"action"`
		Data   map[string]interface{} `json:"data"`
		Where  map[string]interface{} `json:"where"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
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

	log.Printf("Executing CRUD %s on %s: %s", payload.Action, bq.Table, query)

	if _, err := db.Exec(query, args...); err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
