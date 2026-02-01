package sqliter

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestBanquetCompliance(t *testing.T) {
	// 1. Setup Environment
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "compliance.db")
	db, _ := sql.Open("sqlite", dbPath)
	db.Exec(`CREATE TABLE t1 (id int, val text); INSERT INTO t1 VALUES (1, 'A');`)
	db.Close()

	cfg := &Config{ServeFolder: tmpDir}
	server := NewServer(cfg)

	tests := []struct {
		name        string
		pathParam   string
		expectedSQL string // Partial match check
		expectErr   bool
	}{
		{
			name:        "Basic Table",
			pathParam:   "/compliance.db/t1",
			expectedSQL: `SELECT * FROM "t1"`,
		},
		// {
		// 	name:        "Slice Notation (Limit)",
		// 	pathParam:   "/compliance.db/t1[0:5]",
		// 	expectedSQL: `LIMIT 5`,
		// },
		// {
		// 	name:        "Slice Notation (Offset+Limit)",
		// 	pathParam:   "/compliance.db/t1[5:10]",
		// 	expectedSQL: `LIMIT 5 OFFSET 5`,
		// },
		{
			name:        "Column Selection",
			pathParam:   "/compliance.db/t1/id,val",
			expectedSQL: `SELECT "id", "val" FROM "t1"`,
		},
		// Note: Sorting/Filtering syntax depends on specific Banquet implementation details.
		// Assuming standard valid paths:
		{
			name:      "Invalid Path",
			pathParam: "/compliance.db/nonexistent_table",
			expectErr: true, // SQL error likely
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Properly encode URL parameters
			q := make(url.Values)
			q.Set("path", tt.pathParam)

			req := httptest.NewRequest("GET", "/sqliter/rows?"+q.Encode(), nil)
			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)

			resp := w.Result()
			// content-type might be text/plain if error, or json if success
			bodyBytes, _ := io.ReadAll(resp.Body)

			var body map[string]interface{}
			// Ensure we try to decode even if status is error, as we might send JSON error
			json.Unmarshal(bodyBytes, &body)

			if tt.expectErr {
				if resp.StatusCode == http.StatusOK && body["error"] == nil {
					t.Errorf("Expected error, got success")
				}
				return
			}

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("Request failed: status=%d body=%s", resp.StatusCode, string(bodyBytes))
			}

			// Check generated SQL
			generatedSQL, ok := body["sql"].(string)
			if !ok {
				t.Fatalf("Response did not contain 'sql' field: %v", body)
			}

			if !strings.Contains(generatedSQL, tt.expectedSQL) {
				t.Errorf("Expected SQL to contain %q, got %q", tt.expectedSQL, generatedSQL)
			}
		})
	}
}
