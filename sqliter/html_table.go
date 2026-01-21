package sqliter

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"sync"
)

var (
	defaultTemplates *template.Template
	once             sync.Once
)

func initTemplates() {
	once.Do(func() {
		defaultTemplates = LoadTemplates("templates")
	})
}

// LoadTemplates loads templates from the specified directory.
func LoadTemplates(dir string) *template.Template {
	// Define template functions
	funcMap := template.FuncMap{
		"json": func(v interface{}) template.JS {
			a, _ := json.Marshal(v)
			return template.JS(a)
		},
	}

	// Parse templates with functions
	t, err := template.New("base").Funcs(funcMap).ParseGlob(filepath.Join(dir, "*.html"))
	if err != nil {
		log.Printf("Error loading templates from %s: %v. Falling back to simple output.\n", dir, err)
		return nil
	}
	return t
}

// TableWriter handles writing HTML tables with configurable templates.
type TableWriter struct {
	templates *template.Template
	config    *Config
}

// NewTableWriter creates a new TableWriter with the given templates.
// If templates is nil, it will use fallback simple HTML.
func NewTableWriter(t *template.Template, cfg *Config) *TableWriter {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return &TableWriter{templates: t, config: cfg}
}

// HeadData is the data passed to the head.html template.
type HeadData struct {
	Headers      []string
	StickyHeader bool
}

// StartHTMLTable writes the initial HTML for a page with a table style and the table header.
func (tw *TableWriter) StartHTMLTable(w io.Writer, headers []string) {
	if tw.templates == nil {
		fmt_StartHTMLTable(w, headers)
		return
	}

	data := HeadData{
		Headers:      headers,
		StickyHeader: tw.config.StickyHeader,
	}

	if err := tw.templates.ExecuteTemplate(w, "head.html", data); err != nil {
		log.Printf("Error executing head.html: %v\n", err)
		fmt_StartHTMLTable(w, headers)
		return
	}
	flush(w)
}

// WriteHTMLRow writes a single row to the HTML table.
func (tw *TableWriter) WriteHTMLRow(w io.Writer, cells []string) error {
	if tw.templates == nil {
		fmt_WriteHTMLRow(w, cells)
		return nil
	}

	if err := tw.templates.ExecuteTemplate(w, "row.html", cells); err != nil {
		log.Printf("Error executing row.html: %v\n", err)
		fmt_WriteHTMLRow(w, cells)
		return err
	}
	flush(w)
	return nil
}

// EndHTMLTable closes the table and HTML tags.
func (tw *TableWriter) EndHTMLTable(w io.Writer) {
	if tw.templates == nil {
		fmt_EndHTMLTable(w)
		return
	}

	if err := tw.templates.ExecuteTemplate(w, "foot.html", nil); err != nil {
		log.Printf("Error executing foot.html: %v\n", err)
		fmt_EndHTMLTable(w)
		return
	}
	flush(w)
}

// --- Global Functions (Backward Compatibility) ---

// StartHTMLTable writes the initial HTML using default templates.
func StartHTMLTable(w io.Writer, headers []string) {
	initTemplates()
	initTemplates()
	tw := NewTableWriter(defaultTemplates, DefaultConfig())
	tw.StartHTMLTable(w, headers)
}

// WriteHTMLRow writes a single row using default templates.
func WriteHTMLRow(w io.Writer, cells []string) error {
	initTemplates()
	initTemplates()
	tw := NewTableWriter(defaultTemplates, DefaultConfig())
	return tw.WriteHTMLRow(w, cells)
}

// EndHTMLTable closes the table using default templates.
func EndHTMLTable(w io.Writer) {
	initTemplates()
	initTemplates()
	tw := NewTableWriter(defaultTemplates, DefaultConfig())
	tw.EndHTMLTable(w)
}

// --- List View Implementation (Currently not templated, shared) ---

func StartTableList(w io.Writer) {
	io.WriteString(w, `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.1.3/dist/css/bootstrap.min.css" rel="stylesheet">
  <style>
    body { padding: 20px; background-color: #212529; color: #f8f9fa; }
    h3 { margin-bottom: 20px; border-bottom: 1px solid #495057; padding-bottom: 10px; }
    a { text-decoration: none; color: #6ea8fe; font-family: monospace; font-size: 1.1em; }
    a:hover { color: #fff; }
    .list-group-item-dark { background-color: #2c3034; border-color: #373b3e; color: #dee2e6; }
    .list-group-item-action:hover { background-color: #343a40; color: #fff; }
  </style>
</head>
<body>
<div class="container" style="max-width: 800px;">
  <div class="list-group">
`)
	flush(w)
}

func WriteTableLink(w io.Writer, name, url string) error {
	_, err := fmt.Fprintf(w, `<a href="%s" class="list-group-item list-group-item-action list-group-item-dark">%s</a>`, url, name)
	if err != nil {
		return err
	}
	flush(w)
	return nil
}

func EndTableList(w io.Writer) {
	io.WriteString(w, `</div></div></body></html>`)
	flush(w)
}

func flush(w io.Writer) {
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// --- Fallback implementations ---

func fmt_StartHTMLTable(w io.Writer, headers []string) {
	io.WriteString(w, "<!DOCTYPE html><html><head><title>Data</title></head><body><table border='1'><thead><tr>")
	for _, h := range headers {
		io.WriteString(w, "<th>"+h+"</th>")
	}
	io.WriteString(w, "</tr></thead><tbody>")
	flush(w)
}

func fmt_WriteHTMLRow(w io.Writer, cells []string) {
	io.WriteString(w, "<tr>")
	for _, c := range cells {
		io.WriteString(w, "<td>"+c+"</td>")
	}
	io.WriteString(w, "</tr>")
	flush(w)
}

func fmt_EndHTMLTable(w io.Writer) {
	io.WriteString(w, "</tbody></table></body></html>")
	flush(w)
}
