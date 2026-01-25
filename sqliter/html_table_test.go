package sqliter

import (
	"bytes"
	"strings"
	"testing"
)

func TestStartHTMLTable(t *testing.T) {
	var buf bytes.Buffer
	headers := []string{"Name", "Age"}
	StartHTMLTable(&buf, headers, "Test Title")

	output := buf.String()
	if !strings.Contains(output, "Name") || !strings.Contains(output, "Age") {
		t.Errorf("Expected output to contain headers, got %s", output)
	}
	// Check if it started a table
	if !strings.Contains(output, "<table") {
		t.Errorf("Expected output to contain <table> tag, got %s", output)
	}

	// Check if embedded CSS was injected
	if !strings.Contains(output, "background-color: #121212;") {
		t.Errorf("Expected output to contain inlined CSS. Got:\n%s", output)
	}
}

func TestWriteHTMLRow(t *testing.T) {
	var buf bytes.Buffer
	cells := []string{"Alice", "30"}
	WriteHTMLRow(&buf, 0, cells)

	output := buf.String()
	if !strings.Contains(output, "Alice") || !strings.Contains(output, "30") {
		t.Errorf("Expected output to contain cell data, got %s", output)
	}
	if !strings.Contains(output, "<tr") {
		t.Errorf("Expected output to contain <tr> tag, got %s", output)
	}
}

func TestEndHTMLTable(t *testing.T) {
	var buf bytes.Buffer
	EndHTMLTable(&buf)

	output := buf.String()
	if !strings.Contains(output, "</table>") {
		t.Errorf("Expected output to contain </table> tag, got %s", output)
	}
}
