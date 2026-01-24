package sqliter

import (
	"encoding/json"
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
		// Try to load from "templates" directory first
		defaultTemplates = LoadTemplates("templates")
		// If fails, use embedded templates
		if defaultTemplates == nil {
			defaultTemplates = loadEmbeddedTemplates()
		}
	})
}

// GetDefaultTemplates returns the default (possibly embedded) templates.
func GetDefaultTemplates() *template.Template {
	initTemplates()
	return defaultTemplates
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

	absDir, _ := filepath.Abs(dir)
	// Parse templates with functions
	t, err := template.New("base").Funcs(funcMap).ParseGlob(filepath.Join(dir, "*.html"))
	if err != nil {
		log.Printf("Error loading templates from %s (%s): %v. Falling back to simple output.\n", dir, absDir, err)
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
	StyleSheet   string
	Title        string
}

// ListData is the data passed to the list_head.html template.
type ListData struct {
	StyleSheet string
	Title      string
}

// ListItemData is the data passed to the list_item.html template.
type ListItemData struct {
	Name string
	URL  string
	Type string
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

	data := HeadData{
		Headers:      headers,
		StickyHeader: tw.config.StickyHeader,
		StyleSheet:   tw.config.StyleSheet,
		Title:        title,
	}

	if err := tw.templates.ExecuteTemplate(w, "head.html", data); err != nil {
		log.Printf("Error executing head.html: %v\n", err)
		fmt_StartHTMLTable(w, headers)
		return
	}
	flush(w)
}

// StartTableList writes the initial HTML for a list of tables.
func (tw *TableWriter) StartTableList(w io.Writer, title string) {
	if tw.config.Verbose {
		log.Printf("[SQLITER] Starting table list: %s", title)
	}
	if tw.templates == nil {
		fmt_StartTableList(w)
		return
	}

	data := ListData{
		StyleSheet: tw.config.StyleSheet,
		Title:      title,
	}

	if err := tw.templates.ExecuteTemplate(w, "list_head.html", data); err != nil {
		log.Printf("Error executing list_head.html: %v\n", err)
		fmt_StartTableList(w)
		return
	}
	flush(w)
}

// WriteTableLink writes a link to a table.
func (tw *TableWriter) WriteTableLink(w io.Writer, name, url, kind string) error {
	if tw.templates == nil {
		return fmt_WriteTableLink(w, name, url, kind)
	}

	data := ListItemData{
		Name: name,
		URL:  url,
		Type: kind,
	}

	if err := tw.templates.ExecuteTemplate(w, "list_item.html", data); err != nil {
		log.Printf("Error executing list_item.html: %v\n", err)
		return fmt_WriteTableLink(w, name, url, kind)
	}
	flush(w)
	return nil
}

// EndTableList closes the list view.
func (tw *TableWriter) EndTableList(w io.Writer) {
	if tw.templates == nil {
		fmt_EndTableList(w)
		return
	}

	if err := tw.templates.ExecuteTemplate(w, "list_foot.html", nil); err != nil {
		log.Printf("Error executing list_foot.html: %v\n", err)
		fmt_EndTableList(w)
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
func StartHTMLTable(w io.Writer, headers []string, title string) {
	initTemplates()
	initTemplates()
	tw := NewTableWriter(defaultTemplates, DefaultConfig())
	tw.StartHTMLTable(w, headers, title)
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

// --- List View Implementation (Wrapped) ---

func StartTableList(w io.Writer, title string) {
	initTemplates()
	tw := NewTableWriter(defaultTemplates, DefaultConfig())
	tw.StartTableList(w, title)
}

func WriteTableLink(w io.Writer, name, url, kind string) error {
	initTemplates()
	tw := NewTableWriter(defaultTemplates, DefaultConfig())
	return tw.WriteTableLink(w, name, url, kind)
}

func EndTableList(w io.Writer) {
	initTemplates()
	tw := NewTableWriter(defaultTemplates, DefaultConfig())
	tw.EndTableList(w)
}

func flush(w io.Writer) {
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}
