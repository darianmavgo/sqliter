package main

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

	_ "modernc.org/sqlite"
)

// Row represents a row in the tb0 table
type Row struct {
	Path        sql.NullString `json:"path"`
	Name        sql.NullString `json:"name"`
	Size        sql.NullString `json:"size"`
	Extension   sql.NullString `json:"extension"`
	ModTime     sql.NullString `json:"mod_time"`
	CreateTime  sql.NullString `json:"create_time"`
	Permissions sql.NullString `json:"permissions"`
	IsDir       sql.NullString `json:"is_dir"`
	MimeType    sql.NullString `json:"mime_type"`
}

type Response struct {
	Rows       []Row `json:"rows"`
	TotalCount int   `json:"totalCount"`
}

var db *sql.DB

func main() {
	var err error

	// Open the database
	// Assuming running from the root or handling path correctly.
	// We'll use absolute path or relative to binary if strictly defined,
	// but user has provided absolute path in previous steps. Keeping absolute for safety.
	dbPath := "/Users/darianhickman/Documents/Index.sqlite"
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Cannot connect to DB:", err)
	}

	// Serve React Static Files
	// Assuming the binary is run from /go-server/, the dist is at ../react-client/dist
	// Use an absolute path or robust relative path resolution
	cwd, _ := os.Getwd()
	distPath := filepath.Join(cwd, "../react-client/dist")
	fs := http.FileServer(http.Dir(distPath))

	http.Handle("/", fs)
	http.HandleFunc("/rows", enableCors(handleRows))

	// Listen on IPv6 random high port
	// "tcp6" requests IPv6. ":0" requests a random available port.
	listener, err := net.Listen("tcp6", "[::1]:0")
	if err != nil {
		// Fallback to any interface if [::1] fails (though robust for localhost ipv6)
		listener, err = net.Listen("tcp6", ":0")
		if err != nil {
			log.Fatal("Failed to listen on IPv6:", err)
		}
	}

	port := listener.Addr().(*net.TCPAddr).Port
	url := fmt.Sprintf("http://[%s]:%d", "::1", port)

	log.Printf("Server starting on %s...", url)
	log.Printf("Serving static files from: %s", distPath)

	// We print just the URL to stdout so the calling script can capture it easily if needed,
	// but log uses stderr usually.
	fmt.Printf("SERVING_AT=%s\n", url)

	if err := http.Serve(listener, nil); err != nil {
		log.Fatal(err)
	}
}

func enableCors(next http.HandlerFunc) http.HandlerFunc {
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

func handleRows(w http.ResponseWriter, r *http.Request) {
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")
	sortCol := r.URL.Query().Get("sortCol")
	sortDir := r.URL.Query().Get("sortDir")

	start, _ := strconv.Atoi(startStr)
	end, _ := strconv.Atoi(endStr)
	if end == 0 {
		end = 100
	}
	limit := end - start

	query := "SELECT path, name, size, extension, mod_time, create_time, permissions, is_dir, mime_type FROM tb0"

	if sortCol != "" {
		validCols := map[string]bool{
			"path": true, "name": true, "size": true, "extension": true,
			"mod_time": true, "create_time": true, "permissions": true,
			"is_dir": true, "mime_type": true,
		}
		if validCols[sortCol] {
			if sortDir != "desc" {
				sortDir = "asc"
			}
			query += fmt.Sprintf(" ORDER BY %s %s", sortCol, sortDir)
		}
	}

	query += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, start)

	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var result []Row
	for rows.Next() {
		var row Row
		if err := rows.Scan(&row.Path, &row.Name, &row.Size, &row.Extension, &row.ModTime, &row.CreateTime, &row.Permissions, &row.IsDir, &row.MimeType); err != nil {
			log.Println("Scan error:", err)
			continue
		}
		result = append(result, row)
	}

	var total int
	err = db.QueryRow("SELECT COUNT(*) FROM tb0").Scan(&total)
	if err != nil {
		log.Println("Count error:", err)
	}

	resp := Response{
		Rows:       result,
		TotalCount: total,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
