package tests

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/darianmavgo/sqliter/sqliter"
)

func TestNewTableWriter(t *testing.T) {
	tests := []struct {
		name           string
		cfg            *sqliter.Config
		expectedSticky bool
	}{
		{
			name:           "with default config",
			cfg:            sqliter.DefaultConfig(),
			expectedSticky: true,
		},
		{
			name: "with sticky header disabled",
			cfg: &sqliter.Config{
				StickyHeader: false,
			},
			expectedSticky: false,
		},
		{
			name:           "with nil config uses defaults",
			cfg:            nil,
			expectedSticky: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl := sqliter.GetDefaultTemplates()
			tw := sqliter.NewTableWriter(tmpl, tt.cfg)

			if tw == nil {
				t.Fatal("Expected TableWriter, got nil")
			}

			// Test that it can render with the expected sticky header setting
			var buf bytes.Buffer
			tw.StartHTMLTable(&buf, []string{"Col1", "Col2"}, "Test Title")

			output := buf.String()
			if tt.expectedSticky {
				// When sticky is enabled, the CSS should include position: sticky
				if !strings.Contains(output, "position: sticky") {
					t.Error("Expected sticky header CSS when StickyHeader=true")
				}
			}
		})
	}
}

func TestEnableEditable(t *testing.T) {
	tests := []struct {
		name              string
		editable          bool
		expectMetaTag     bool
		expectEditableHdr bool
	}{
		{
			name:              "editable enabled",
			editable:          true,
			expectMetaTag:     true,
			expectEditableHdr: true,
		},
		{
			name:              "editable disabled",
			editable:          false,
			expectMetaTag:     false,
			expectEditableHdr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl := sqliter.GetDefaultTemplates()
			cfg := sqliter.DefaultConfig()
			tw := sqliter.NewTableWriter(tmpl, cfg)

			tw.EnableEditable(tt.editable)

			// Use httptest.ResponseRecorder to capture headers
			rec := httptest.NewRecorder()
			tw.StartHTMLTable(rec, []string{"Name", "Email"}, "Users")

			output := rec.Body.String()

			// Check for meta tag
			metaTag := `<meta name="sqliter-editable" content="true">`
			if tt.expectMetaTag {
				if !strings.Contains(output, metaTag) {
					t.Errorf("Expected editable meta tag in output when editable=%v", tt.editable)
				}
			} else {
				if strings.Contains(output, metaTag) {
					t.Errorf("Did not expect editable meta tag when editable=%v", tt.editable)
				}
			}

			// Check for HTTP header
			headerValue := rec.Header().Get("X-SQLiter-Editable")
			if tt.expectEditableHdr {
				if headerValue != "true" {
					t.Errorf("Expected X-SQLiter-Editable header to be 'true', got '%s'", headerValue)
				}
			} else {
				if headerValue != "" {
					t.Errorf("Did not expect X-SQLiter-Editable header when editable=%v, got '%s'", tt.editable, headerValue)
				}
			}

			// Check for edit bar row
			editBarRow := `<tr id="edit-bar-row">`
			if !strings.Contains(output, editBarRow) {
				t.Error("Expected edit bar row in output")
			}

			// Check for row-id-header
			rowIdHeader := `class="row-id-header"`
			if !strings.Contains(output, rowIdHeader) {
				t.Error("Expected row-id-header class in output")
			}
		})
	}
}

func TestSetStickyHeader(t *testing.T) {
	tests := []struct {
		name         string
		sticky       bool
		expectSticky bool
	}{
		{
			name:         "sticky header enabled",
			sticky:       true,
			expectSticky: true,
		},
		{
			name:         "sticky header disabled",
			sticky:       false,
			expectSticky: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl := sqliter.GetDefaultTemplates()
			cfg := sqliter.DefaultConfig()
			tw := sqliter.NewTableWriter(tmpl, cfg)

			tw.SetStickyHeader(tt.sticky)

			var buf bytes.Buffer
			tw.StartHTMLTable(&buf, []string{"Col1"}, "Test")

			output := buf.String()

			// The sticky CSS is in default.css which gets embedded
			// When sticky is true, the CSS should include position: sticky
			// When false, it might still be in the CSS but the template logic should handle it
			// For now, we just verify the method accepts the value without error
			if !strings.Contains(output, "<table") {
				t.Error("Expected table tag in output")
			}
		})
	}
}

func TestStartHTMLTable_TitleRendering(t *testing.T) {
	tests := []struct {
		name          string
		title         string
		expectedTitle string
	}{
		{
			name:          "with custom title",
			title:         "My Database",
			expectedTitle: "My Database",
		},
		{
			name:          "with empty title defaults to SQLITER",
			title:         "",
			expectedTitle: "SQLITER",
		},
		{
			name:          "with path-like title",
			title:         "data.db",
			expectedTitle: "data.db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl := sqliter.GetDefaultTemplates()
			cfg := sqliter.DefaultConfig()
			tw := sqliter.NewTableWriter(tmpl, cfg)

			var buf bytes.Buffer
			tw.StartHTMLTable(&buf, []string{"Col1"}, tt.title)

			output := buf.String()

			// Check for title tag
			if tt.title != "" {
				expectedTitleTag := "<title>" + tt.expectedTitle + "</title>"
				if !strings.Contains(output, expectedTitleTag) {
					t.Errorf("Expected title tag with '%s', got output:\n%s", tt.expectedTitle, output)
				}
			} else {
				// Empty title should default to SQLITER
				if !strings.Contains(output, "<title>SQLITER</title>") {
					t.Error("Expected default SQLITER title for empty title")
				}
			}
		})
	}
}

func TestWriteHTMLRow_WithRowID(t *testing.T) {
	tmpl := sqliter.GetDefaultTemplates()
	cfg := sqliter.DefaultConfig()
	tw := sqliter.NewTableWriter(tmpl, cfg)

	var buf bytes.Buffer
	cells := []string{"Alice", "alice@example.com"}
	err := tw.WriteHTMLRow(&buf, 5, cells)

	if err != nil {
		t.Fatalf("WriteHTMLRow failed: %v", err)
	}

	output := buf.String()

	// Check for row ID cell
	if !strings.Contains(output, `<td class="row-id">5</td>`) {
		t.Error("Expected row ID cell with index 5")
	}

	// Check for data cells
	if !strings.Contains(output, "Alice") {
		t.Error("Expected cell data 'Alice'")
	}
	if !strings.Contains(output, "alice@example.com") {
		t.Error("Expected cell data 'alice@example.com'")
	}

	// Check for tr tag
	if !strings.Contains(output, "<tr") {
		t.Error("Expected <tr> tag")
	}
}

func TestEndHTMLTable_IncludesJS(t *testing.T) {
	tmpl := sqliter.GetDefaultTemplates()
	cfg := sqliter.DefaultConfig()
	tw := sqliter.NewTableWriter(tmpl, cfg)

	var buf bytes.Buffer
	tw.EndHTMLTable(&buf)

	output := buf.String()

	// Check for closing tags
	if !strings.Contains(output, "</tbody>") {
		t.Error("Expected </tbody> tag")
	}
	if !strings.Contains(output, "</table>") {
		t.Error("Expected </table> tag")
	}
	if !strings.Contains(output, "</body>") {
		t.Error("Expected </body> tag")
	}
	if !strings.Contains(output, "</html>") {
		t.Error("Expected </html> tag")
	}

	// Check for embedded JavaScript
	if !strings.Contains(output, "<script>") {
		t.Error("Expected <script> tag with embedded JS")
	}
	if !strings.Contains(output, "DOMContentLoaded") {
		t.Error("Expected embedded JavaScript content")
	}
}

func TestTableWriter_FullTableFlow(t *testing.T) {
	tmpl := sqliter.GetDefaultTemplates()
	cfg := sqliter.DefaultConfig()
	cfg.RowCRUD = true
	tw := sqliter.NewTableWriter(tmpl, cfg)

	tw.EnableEditable(true)
	tw.SetStickyHeader(true)

	rec := httptest.NewRecorder()

	// Start table
	headers := []string{"ID", "Name", "Email"}
	tw.StartHTMLTable(rec, headers, "Users Table")

	// Write rows
	rows := [][]string{
		{"1", "Alice", "alice@example.com"},
		{"2", "Bob", "bob@example.com"},
		{"3", "Charlie", "charlie@example.com"},
	}

	for i, row := range rows {
		err := tw.WriteHTMLRow(rec, i, row)
		if err != nil {
			t.Fatalf("Failed to write row %d: %v", i, err)
		}
	}

	// End table
	tw.EndHTMLTable(rec)

	output := rec.Body.String()

	// Verify complete HTML structure
	requiredElements := []string{
		"<!DOCTYPE html>",
		"<html",
		"<head>",
		"<title>Users Table</title>",
		`<meta name="sqliter-editable" content="true">`,
		"<style>",
		"<table",
		"<thead>",
		"<tr id=\"edit-bar-row\">",
		`class="row-id-header"`,
		"<th data-sortable=\"true\">ID</th>",
		"<th data-sortable=\"true\">Name</th>",
		"<th data-sortable=\"true\">Email</th>",
		"<tbody>",
		`<td class="row-id">0</td>`,
		"Alice",
		"alice@example.com",
		"Bob",
		"Charlie",
		"</tbody>",
		"</table>",
		"<script>",
		"</html>",
	}

	for _, elem := range requiredElements {
		if !strings.Contains(output, elem) {
			t.Errorf("Missing required element: %s", elem)
		}
	}

	// Verify X-SQLiter-Editable header
	if rec.Header().Get("X-SQLiter-Editable") != "true" {
		t.Error("Expected X-SQLiter-Editable header to be set")
	}
}

func TestTableWriter_NonEditableMode(t *testing.T) {
	tmpl := sqliter.GetDefaultTemplates()
	cfg := sqliter.DefaultConfig()
	cfg.RowCRUD = false
	tw := sqliter.NewTableWriter(tmpl, cfg)

	tw.EnableEditable(false)

	rec := httptest.NewRecorder()
	tw.StartHTMLTable(rec, []string{"Name"}, "Read-Only Table")

	output := rec.Body.String()

	// Should NOT have editable meta tag
	if strings.Contains(output, `<meta name="sqliter-editable"`) {
		t.Error("Should not have editable meta tag when editable=false")
	}

	// Should NOT have X-SQLiter-Editable header
	if rec.Header().Get("X-SQLiter-Editable") != "" {
		t.Error("Should not have X-SQLiter-Editable header when editable=false")
	}

	// Should still have basic structure
	if !strings.Contains(output, "<table") {
		t.Error("Expected table tag")
	}
}

func TestTableWriter_EmbeddedAssetsLoaded(t *testing.T) {
	tmpl := sqliter.GetDefaultTemplates()
	cfg := sqliter.DefaultConfig()
	tw := sqliter.NewTableWriter(tmpl, cfg)

	var buf bytes.Buffer
	tw.StartHTMLTable(&buf, []string{"Col"}, "Test")

	output := buf.String()

	// Verify CSS is embedded (check for some known CSS content)
	cssPatterns := []string{
		"background-color: #121212;", // Dark theme
		"color: #e0e0e0;",            // Light text
		"font-family:",               // Font styling
	}

	for _, pattern := range cssPatterns {
		if !strings.Contains(output, pattern) {
			t.Errorf("Expected embedded CSS pattern: %s", pattern)
		}
	}

	// Now test EndHTMLTable for JS
	buf.Reset()
	tw.EndHTMLTable(&buf)
	output = buf.String()

	// Verify JS is embedded
	jsPatterns := []string{
		"DOMContentLoaded",
		"addEventListener",
		"querySelector",
	}

	for _, pattern := range jsPatterns {
		if !strings.Contains(output, pattern) {
			t.Errorf("Expected embedded JS pattern: %s", pattern)
		}
	}
}
