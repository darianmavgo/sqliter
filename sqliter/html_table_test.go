package sqliter

import (
	"bytes"
	"strings"
	"testing"
)

func TestStartHTMLTable(t *testing.T) {
	var buf bytes.Buffer
	headers := []string{"Name", "Age"}
	StartHTMLTable(&buf, headers)

	output := buf.String()
	if !strings.Contains(output, "Name") || !strings.Contains(output, "Age") {
		t.Errorf("Expected output to contain headers, got %s", output)
	}
    // Check if it started a table
    if !strings.Contains(output, "<table") {
        t.Errorf("Expected output to contain <table> tag, got %s", output)
    }
}

func TestWriteHTMLRow(t *testing.T) {
	var buf bytes.Buffer
	cells := []string{"Alice", "30"}
	WriteHTMLRow(&buf, cells)

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

func TestStartTableList(t *testing.T) {
    var buf bytes.Buffer
    StartTableList(&buf)

    output := buf.String()
    if !strings.Contains(output, "<html") {
        t.Errorf("Expected output to contain <html> tag, got %s", output)
    }
}
