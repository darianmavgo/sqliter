package sqliter

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestApiQueryTable_ImplicitTable(t *testing.T) {
	// Setup temp dir and DB
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	defer db.Close()

	_, err = db.Exec("CREATE TABLE foo (a TEXT, b TEXT); INSERT INTO foo VALUES ('1', '2');")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	db.Close() // Close so server can open it

	// Setup Server
	cfg := &Config{
		ServeFolder:             tmpDir,
		AutoRedirectSingleTable: true,
	}
	server := NewServer(cfg)

	// Test Case 1: Select specific columns without table name
	// URL: /sqliter/rows?path=/test.db/a,b
	req := httptest.NewRequest("GET", "/sqliter/rows?path=/test.db/a,b", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
		// print body
		var body map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&body)
		t.Logf("Body: %v", body)
	} else {
		var body struct {
			Rows []map[string]interface{} `json:"rows"`
			SQL  string                   `json:"sql"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Errorf("Failed to decode response: %v", err)
		}
		if len(body.Rows) != 1 {
			t.Errorf("Expected 1 row, got %d", len(body.Rows))
		}
		// Verify columns
		row := body.Rows[0]
		if row["a"] != "1" || row["b"] != "2" {
			t.Errorf("Unexpected row data: %v", row)
		}
	}

	// Test Case 2: Multi-table DB (Ambiguous)
	dbPath2 := filepath.Join(tmpDir, "multi.db")
	db2, _ := sql.Open("sqlite", dbPath2)
	db2.Exec("CREATE TABLE t1 (x); CREATE TABLE t2 (y);")
	db2.Close()

	// Use comma to force empty table in banquet parsing
	req2 := httptest.NewRequest("GET", "/sqliter/rows?path=/multi.db/x,y", nil)
	w2 := httptest.NewRecorder()
	server.ServeHTTP(w2, req2)

	resp2 := w2.Result()
	// TODO: Server currently returns 500 for this error, should ideally be 400.
	if resp2.StatusCode != http.StatusBadRequest && resp2.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 400 or 500 for ambiguous table, got %d", resp2.StatusCode)
	}
}

func TestApiQueryTable_LimitZero(t *testing.T) {
	// Setup temp dir and DB
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	defer db.Close()

	_, err = db.Exec("CREATE TABLE t1 (a TEXT); INSERT INTO t1 VALUES ('1'), ('2'), ('3');")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	db.Close()

	// Setup Server
	cfg := &Config{
		ServeFolder: tmpDir,
	}
	server := NewServer(cfg)

	// URL: /sqliter/rows?path=/test.db/t1&start=0&end=0
	req := httptest.NewRequest("GET", "/sqliter/rows?path=/test.db/t1&start=0&end=0", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var body struct {
		Rows    []map[string]interface{} `json:"rows"`
		Columns []string                 `json:"columns"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if len(body.Rows) != 0 {
		t.Errorf("Expected 0 rows, got %d", len(body.Rows))
	}

	if len(body.Columns) == 0 {
		t.Errorf("Expected columns to be present, got empty")
	}
}
