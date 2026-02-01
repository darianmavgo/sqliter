package sqliter

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// TestApiIntegration covers Sorting, Pagination, and basic Filtering via the API.
func TestApiIntegration(t *testing.T) {
	// 1. Setup Environment
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	// Create table with test data
	_, err = db.Exec(`
		CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER);
		INSERT INTO users (name, age) VALUES ('Alice', 30);
		INSERT INTO users (name, age) VALUES ('Bob', 25);
		INSERT INTO users (name, age) VALUES ('Charlie', 35);
		INSERT INTO users (name, age) VALUES ('Dave', 20);
		INSERT INTO users (name, age) VALUES ('Eve', 40);
	`)
	if err != nil {
		t.Fatalf("Failed to setup table: %v", err)
	}
	db.Close()

	cfg := &Config{
		ServeFolder:             tmpDir,
		AutoRedirectSingleTable: true,
	}
	server := NewServer(cfg)

	// Helper to make requests
	doRequest := func(query string) (*httptest.ResponseRecorder, map[string]interface{}) {
		req := httptest.NewRequest("GET", "/sqliter/rows?"+query, nil)
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)
		resp := w.Result()

		var body map[string]interface{}
		// We decode into interface{} to flexibly inspect rows
		if resp.StatusCode == http.StatusOK {
			json.NewDecoder(resp.Body).Decode(&body)
		} else {
			// If error, try to decode error message
			json.NewDecoder(resp.Body).Decode(&body)
			body["_status"] = resp.StatusCode
		}
		return w, body
	}

	// 2. Test Cases

	t.Run("Sorting (ASC)", func(t *testing.T) {
		// sortCol=age, sortDir=asc -> Dave (20) should be first
		_, body := doRequest("db=test.db&table=users&sortCol=age&sortDir=asc")
		rows := body["rows"].([]interface{})
		firstRow := rows[0].(map[string]interface{})

		if firstRow["name"] != "Dave" {
			t.Errorf("Expected Dave first, got %v", firstRow["name"])
		}
	})

	t.Run("Sorting (DESC)", func(t *testing.T) {
		// sortCol=age, sortDir=desc -> Eve (40) should be first
		_, body := doRequest("db=test.db&table=users&sortCol=age&sortDir=desc")
		rows := body["rows"].([]interface{})
		firstRow := rows[0].(map[string]interface{})

		if firstRow["name"] != "Eve" {
			t.Errorf("Expected Eve first, got %v", firstRow["name"])
		}
	})

	t.Run("Pagination", func(t *testing.T) {
		// start=1, end=3 -> Should return 2 rows (offset 1, limit 2).
		// Sorted by name implicit/default? No, let's force sort to be deterministic.
		_, body := doRequest("db=test.db&table=users&sortCol=name&sortDir=asc&start=1&end=3")
		rows := body["rows"].([]interface{})

		// Order: Alice, Bob, Charlie, Dave, Eve
		// Offset 1 (Bob), Limit 2 (Bob, Charlie)
		if len(rows) != 2 {
			t.Errorf("Expected 2 rows, got %d", len(rows))
		}

		r0 := rows[0].(map[string]interface{})
		if r0["name"] != "Bob" {
			t.Errorf("Expected Bob at index 0 (was offset 1), got %v", r0["name"])
		}
	})

	t.Run("Filtering (Equals)", func(t *testing.T) {
		// filterModel via AG Grid JSON structure
		// {"name": {"filterType":"text","type":"equals","filter":"Charlie"}}
		filterJSON := `{"name":{"filterType":"text","type":"equals","filter":"Charlie"}}`
		_, body := doRequest(fmt.Sprintf("db=test.db&table=users&filterModel=%s", filterJSON))

		rows := body["rows"].([]interface{})
		if len(rows) != 1 {
			t.Errorf("Expected 1 row for Charlie, got %d", len(rows))
		}
		r0 := rows[0].(map[string]interface{})
		if r0["name"] != "Charlie" {
			t.Errorf("Expected Charlie, got %v", r0["name"])
		}
	})

	t.Run("Filtering (Number GreaterThan)", func(t *testing.T) {
		// {"age": {"filterType":"number","type":"greaterThan","filter":30}}
		// Expect: Charlie(35), Eve(40) -> 2 rows
		filterJSON := `{"age":{"filterType":"number","type":"greaterThan","filter":30}}`
		_, body := doRequest(fmt.Sprintf("db=test.db&table=users&filterModel=%s", filterJSON))

		rows := body["rows"].([]interface{})
		// Note: Charlie is 35, Eve is 40. Alice is 30 (not greater).
		if len(rows) != 2 {
			t.Errorf("Expected 2 rows > 30, got %d", len(rows))
		}
	})
}
