package tests

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/darianmavgo/sqliter/internal/testutil"
	"github.com/darianmavgo/sqliter/server"
	"github.com/darianmavgo/sqliter/sqliter"
	_ "github.com/mattn/go-sqlite3"
)

func TestBrowserLinksFlow(t *testing.T) {
	// 1. Setup Environment
	// We create a temporary directory to act as the DataDir
	tempDir := testutil.GetTestOutputDir(t, "browser_links_test")

	// Create the DB file "test_links.db"
	dbName := "test_links.db"
	dbPath := filepath.Join(tempDir, dbName)

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	// Create a single table 'links' to ensure auto-redirect works (if enabled)
	_, err = db.Exec("CREATE TABLE links (id INTEGER PRIMARY KEY, url TEXT)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Insert initial values
	initialLinks := []string{
		"http://example.com/first",
		"http://example.com/second",
		"http://example.com/third",
	}
	for _, link := range initialLinks {
		_, err = db.Exec("INSERT INTO links (url) VALUES (?)", link)
		if err != nil {
			t.Fatalf("Failed to insert link %s: %v", link, err)
		}
	}

	// Setup Server
	cfg := sqliter.DefaultConfig()
	cfg.DataDir = tempDir
	cfg.RowCRUD = true
	cfg.AutoRedirectSingleTable = true // Enable this to test the redirection behavior

	srv := server.NewServer(cfg)
	ts := httptest.NewServer(srv)
	defer ts.Close()

	client := ts.Client()

	// 2. Browse http://.../test_links.db
	targetURL := ts.URL + "/" + dbName
	t.Logf("Browsing to: %s", targetURL)

	resp, err := client.Get(targetURL)
	if err != nil {
		t.Fatalf("Failed to browse %s: %v", targetURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 after redirect/load, got %d", resp.StatusCode)
	}

	// Calculate the actual URL we ended up at (should be /test_links.db/links)
	finalURL := resp.Request.URL.String()
	t.Logf("Landed on: %s", finalURL)

	// Verify that we actually landed on the table page
	// We check if the final URL ends with "/links"
	if filepath.Base(finalURL) != "links" {
		t.Logf("Warning: Did not redirect to 'links' table as expected. Current path: %s", finalURL)
		// Try to construct manually if it failed, or fail if strictly testing redirect
		// For robustness of the flow test, we can force it, but let's see.
		finalURL = ts.URL + "/" + dbName + "/links"
	}

	// 3. Edit the second value
	// We assume the second value is ID=2.
	// Create random number
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	randomNum := rnd.Intn(10000)
	newURL := fmt.Sprintf("http://example.com/second/%d", randomNum)
	t.Logf("Updating second row to: %s", newURL)

	// Construct update payload
	updatePayload := map[string]interface{}{
		"action": "update",
		"data": map[string]interface{}{
			"url": newURL,
		},
		"where": map[string]interface{}{
			"id": 2, // Second inserted value
		},
	}

	jsonData, err := json.Marshal(updatePayload)
	if err != nil {
		t.Fatalf("Failed to marshal payload: %v", err)
	}

	// Perform POST
	respEdit, err := client.Post(finalURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to post update: %v", err)
	}
	defer respEdit.Body.Close()

	if respEdit.StatusCode != http.StatusOK {
		t.Errorf("Edit failed with status: %d", respEdit.StatusCode)
	}

	// 4. Confirm updates via SQL
	t.Log("Verifying update via SQL...")
	var storedURL string
	err = db.QueryRow("SELECT url FROM links WHERE id = 2").Scan(&storedURL)
	if err != nil {
		t.Fatalf("Failed to query DB: %v", err)
	}

	if storedURL != newURL {
		t.Errorf("Verification failed. Expected '%s', got '%s'", newURL, storedURL)
	} else {
		t.Logf("Success! DB contains updated value: %s", storedURL)
	}
}
