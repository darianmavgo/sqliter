package tests

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/darianmavgo/sqliter/internal/testutil"
	"github.com/darianmavgo/sqliter/server"
	"github.com/darianmavgo/sqliter/sqliter"
	_ "modernc.org/sqlite"
)

func TestRowCRUD(t *testing.T) {
	// Setup temporary directory
	tempDir := testutil.GetTestOutputDir(t, "row_crud")
	// defer os.RemoveAll(tempDir)

	// Setup database
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	_, err = db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, email TEXT)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Setup Server
	cfg := sqliter.DefaultConfig()
	cfg.DataDir = tempDir
	cfg.RowCRUD = true
	srv := server.NewServer(cfg)
	ts := httptest.NewServer(srv)
	defer ts.Close()

	// 1. Test CREATE
	t.Log("Testing CREATE...")
	createPayload := map[string]interface{}{
		"action": "create",
		"data": map[string]interface{}{
			"id":    1,
			"name":  "Alice",
			"email": "alice@example.com",
		},
	}
	if err := sendCRUD(ts.URL+"/test.db/users", createPayload); err != nil {
		t.Fatalf("CREATE failed: %v", err)
	}

	// Verify creation
	var name string
	err = db.QueryRow("SELECT name FROM users WHERE id = 1").Scan(&name)
	if err != nil {
		t.Fatalf("Failed to query created row: %v", err)
	}
	if name != "Alice" {
		t.Errorf("Expected name Alice, got %s", name)
	}

	// 2. Test UPDATE
	t.Log("Testing UPDATE...")
	updatePayload := map[string]interface{}{
		"action": "update",
		"data": map[string]interface{}{
			"email": "alice_new@example.com",
		},
		"where": map[string]interface{}{
			"id": 1,
		},
	}
	if err := sendCRUD(ts.URL+"/test.db/users", updatePayload); err != nil {
		t.Fatalf("UPDATE failed: %v", err)
	}

	// Verify update
	var email string
	err = db.QueryRow("SELECT email FROM users WHERE id = 1").Scan(&email)
	if err != nil {
		t.Fatalf("Failed to query updated row: %v", err)
	}
	if email != "alice_new@example.com" {
		t.Errorf("Expected email alice_new@example.com, got %s", email)
	}

	// 3. Test DELETE
	t.Log("Testing DELETE...")
	deletePayload := map[string]interface{}{
		"action": "delete",
		"where": map[string]interface{}{
			"id": 1,
		},
	}
	if err := sendCRUD(ts.URL+"/test.db/users", deletePayload); err != nil {
		t.Fatalf("DELETE failed: %v", err)
	}

	// Verify deletion
	err = db.QueryRow("SELECT name FROM users WHERE id = 1").Scan(&name)
	if err != sql.ErrNoRows {
		t.Errorf("Expected ErrNoRows, got %v", err)
	}
}

func sendCRUD(url string, payload interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// body, _ := ioutil.ReadAll(resp.Body) // skipping body for now to fix import cycle/complexity if I add fmt
		return http.ErrNotSupported
	}
	return nil
}
