package sqliter

import (
	"embed"
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"sync"
)

var (
	defaultTemplates *template.Template
	once             sync.Once
)

//go:embed templates/*
var embeddedFS embed.FS

func initTemplates() {
	once.Do(func() {
		var err error
		defaultTemplates, err = GetEmbeddedTemplates()
		if err != nil {
			log.Printf("Error loading embedded templates: %v\n", err)
		}
	})
}

// GetEmbeddedTemplates returns the embedded templates.
func GetEmbeddedTemplates() (*template.Template, error) {
	// Define template functions
	funcMap := template.FuncMap{
		"json": func(v interface{}) template.JS {
			a, _ := json.Marshal(v)
			return template.JS(a)
		},
		"safe": func(s string) template.HTML {
			return template.HTML(s)
		},
	}
	// Parse templates from the embedded filesystem
	return template.New("base").Funcs(funcMap).ParseFS(embeddedFS, "templates/*.html")
}

// GetEmbeddedAssets returns the embedded assets filesystem.
func GetEmbeddedAssets() embed.FS {
	return embeddedFS
}

// GetDefaultTemplates returns the default (possibly embedded) templates.
func GetDefaultTemplates() *template.Template {
	initTemplates()
	return defaultTemplates
}

// TableWriter handles writing HTML tables with configurable templates.
type TableWriter struct {
	templates *template.Template
	config    *Config
	editable  bool
}

// NewTableWriter creates a new TableWriter with the given templates.
// If templates is nil, it will use fallback simple HTML.
func NewTableWriter(t *template.Template, cfg *Config) *TableWriter {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return &TableWriter{templates: t, config: cfg}
}

// EnableEditable sets the editable flag for the table.
func (tw *TableWriter) EnableEditable(editable bool) {
	tw.editable = editable
}

// HeadData is the data passed to the head.html template.
// HeadData is the data passed to the head.html template.
type HeadData struct {
	Headers      []string
	StickyHeader bool
	StyleSheet   string // Kept for backward compat or external refs
	CSS          template.CSS
	Title        string
	Editable     bool
}

// StartHTMLTable writes the initial HTML for a page with a table style and the table header.
func (tw *TableWriter) StartHTMLTable(w io.Writer, headers []string, title string) {
	if tw.config.Verbose {
		log.Printf("[SQLITER] Starting HTML table: %s with %d headers", title, len(headers))
	}
	if tw.templates == nil {
		if tw.config.Verbose {
			log.Printf("[SQLITER] No templates found, using fallback simple table.")
		}
		fmt_StartHTMLTable(w, headers)
		return
	}

	// Signal to the client that this table is editable via header if writing to HTTP
	if tw.editable {
		if rw, ok := w.(http.ResponseWriter); ok {
			rw.Header().Set("X-SQLiter-Editable", "true")
		}
	}

	// Read embedded CSS
	var cssContent template.CSS
	cssData, err := embeddedFS.ReadFile("templates/default.css")
	if err == nil {
		cssContent = template.CSS(cssData)
	} else {
		log.Printf("Error reading default.css: %v", err)
	}

	data := HeadData{
		Headers:      headers,
		StickyHeader: tw.config.StickyHeader,
		StyleSheet:   tw.config.StyleSheet,
		CSS:          cssContent,
		Title:        title,
		Editable:     tw.editable,
	}

	if err := tw.templates.ExecuteTemplate(w, "head.html", data); err != nil {
		log.Printf("Error executing head.html: %v\n", err)
		fmt_StartHTMLTable(w, headers)
		return
	}
	flush(w)
}

// WriteHTMLRow writes a single row to the HTML table.
func (tw *TableWriter) WriteHTMLRow(w io.Writer, index int, cells []string) error {
	if tw.templates == nil {
		fmt_WriteHTMLRow(w, index, cells)
		return nil
	}

	data := struct {
		Index int
		Cells []string
	}{
		Index: index,
		Cells: cells,
	}

	if err := tw.templates.ExecuteTemplate(w, "row.html", data); err != nil {
		log.Printf("Error executing row.html: %v\n", err)
		fmt_WriteHTMLRow(w, index, cells)
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

	// Read embedded JS
	var jsContent template.JS
	jsData, err := embeddedFS.ReadFile("templates/default.js")
	if err == nil {
		jsContent = template.JS(jsData)
	} else {
		log.Printf("Error reading default.js: %v", err)
	}

	data := struct {
		JS template.JS
	}{
		JS: jsContent,
	}

	if err := tw.templates.ExecuteTemplate(w, "foot.html", data); err != nil {
		log.Printf("Error executing foot.html: %v\n", err)
		fmt_EndHTMLTable(w)
		return
	}
	flush(w)
}

// --- Global Functions (Backward Compatibility) ---

// StartHTMLTable writes the initial HTML using default templates.
func StartHTMLTable(w io.Writer, headers []string, title string) {
	initTemplates()
	tw := NewTableWriter(defaultTemplates, DefaultConfig())
	tw.StartHTMLTable(w, headers, title)
}

// WriteHTMLRow writes a single row using default templates.
func WriteHTMLRow(w io.Writer, index int, cells []string) error {
	initTemplates()
	tw := NewTableWriter(defaultTemplates, DefaultConfig())
	return tw.WriteHTMLRow(w, index, cells)
}

// EndHTMLTable closes the table using default templates.
func EndHTMLTable(w io.Writer) {
	initTemplates()
	tw := NewTableWriter(defaultTemplates, DefaultConfig())
	tw.EndHTMLTable(w)
}

func flush(w io.Writer) {
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}
