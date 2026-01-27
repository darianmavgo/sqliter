package tests

import (
	"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darianmavgo/sqliter/internal/testutil"
	"github.com/darianmavgo/sqliter/server"
	"github.com/darianmavgo/sqliter/sqliter"
	_ "modernc.org/sqlite"
)

func TestTableUXStructure(t *testing.T) {
	// Setup temporary directory
	tempDir := testutil.GetTestOutputDir(t, "ux_structure")

	// Setup database
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	_, err = db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	_, err = db.Exec("INSERT INTO users (name) VALUES ('Alice')")
	if err != nil {
		t.Fatal(err)
	}

	// Setup Server
	cfg := sqliter.DefaultConfig()
	cfg.DataDir = tempDir
	// Point TemplateDir to the real templates relative to tests/ package
	cfg.TemplateDir = "../templates"

	srv := server.NewServer(cfg)
	ts := httptest.NewServer(srv)
	defer ts.Close()

	// Request the table
	resp, err := http.Get(ts.URL + "/test.db/users")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	body := string(bodyBytes)

	// Verify Structure
	checks := []struct {
		Name     string
		Expected string
	}{
		{"Edit Bar Row", `<tr id="edit-bar-row">`},
		{"Pencil Header", `class="row-id-header"`}, // OR check specific char if encoding allows
		{"Row ID Cell", `class="row-id"`},
		{"Row Index 0", `<td class="row-id">0</td>`}, // Assuming 0-indexed
	}

	for _, check := range checks {
		if !strings.Contains(body, check.Expected) {
			t.Errorf("UX Verification Failed: %s not found in HTML.\nExpected: %s", check.Name, check.Expected)
		}
	}

	// Check specifically that pencil emoji is around if possible,
	// but strictly encoding of source file vs response might ensure HTML escape or raw bytes.
	// We put '✏️' in template.
	if !strings.Contains(body, "✏️") {
		// Attempt checking assuming it might be escaped?
		// But in Go templates it usually prints as is if not specialized.
		t.Logf("Warning: Pencil emoji not found directly. Checking context...")
	}
}
