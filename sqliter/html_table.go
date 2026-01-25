package sqliter

import (
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
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
	})
}

// GetDefaultTemplates returns the default (possibly embedded) templates.
func GetDefaultTemplates() *template.Template {
	initTemplates()
	return defaultTemplates
}

// LoadTemplates loads templates from the specified directory.
// If the directory isn't found, it tries walking up the tree to find it.
func LoadTemplates(dir string) *template.Template {
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

	searchDir := dir
	// If path doesn't exist, try walking up to find a directory with that name
	if _, err := os.Stat(searchDir); os.IsNotExist(err) {
		current := "."
		for i := 0; i < 5; i++ {
			candidate := filepath.Join(current, dir)
			if _, err := os.Stat(candidate); err == nil {
				searchDir = candidate
				break
			}
			current = filepath.Join("..", current)
		}
	}

	absDir, _ := filepath.Abs(searchDir)
	// Parse templates with functions
	t, err := template.New("base").Funcs(funcMap).ParseGlob(filepath.Join(searchDir, "*.html"))
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
type HeadData struct {
	Headers      []string
	StickyHeader bool
	StyleSheet   string
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

	data := HeadData{
		Headers:      headers,
		StickyHeader: tw.config.StickyHeader,
		StyleSheet:   tw.config.StyleSheet,
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
