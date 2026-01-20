package main

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

    _ "github.com/mattn/go-sqlite3"
)

func TestHandler(t *testing.T) {
	// Setup: Create a temporary sample_data directory
	err := os.MkdirAll("sample_data", 0755)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
        // Cleanup if we want, but since it's integration with existing folder structure,
        // we might interfere with other tests if running concurrently.
        // Assuming we are running in a sandbox where we can modify sample_data or we should rely on existing ones.
        // The instruction said "use sample_data", so I should probably rely on existing ones or creating a dedicated test one.
        // Let's create a dedicated test db file.
	}()

    // Create a dummy db
    dbPath := filepath.Join("sample_data", "test.db")
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        t.Fatal(err)
    }
    _, err = db.Exec("CREATE TABLE test_table (id INTEGER, val TEXT); INSERT INTO test_table VALUES (1, 'testval');")
    if err != nil {
        t.Fatal(err)
    }
    db.Close()
    defer os.Remove(dbPath)

	// Test 1: List files (Root)
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
    if !strings.Contains(rr.Body.String(), "test.db") {
        t.Errorf("handler root did not list test.db")
    }

    // Test 2: Query DB
    req, err = http.NewRequest("GET", "/test.db/test_table", nil)
    if err != nil {
        t.Fatal(err)
    }
    rr = httptest.NewRecorder()
    handler.ServeHTTP(rr, req)

    if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
    if !strings.Contains(rr.Body.String(), "testval") {
        t.Errorf("handler did not return expected data")
    }
}
