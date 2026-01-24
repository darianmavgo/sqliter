package sqliter

import (
	"embed"
	"encoding/json"
	"html/template"
	"io/fs"
	"log"
)

//go:embed templates/*.html
var embeddedTemplatesFS embed.FS

func loadEmbeddedTemplates() *template.Template {
	// Define template functions
	funcMap := template.FuncMap{
		"json": func(v interface{}) template.JS {
			a, _ := json.Marshal(v)
			return template.JS(a)
		},
	}

	subFS, err := fs.Sub(embeddedTemplatesFS, "templates")
	if err != nil {
		log.Printf("Error creating sub FS: %v", err)
		return nil
	}

	t, err := template.New("base").Funcs(funcMap).ParseFS(subFS, "*.html")
	if err != nil {
		log.Printf("Error loading embedded templates: %v", err)
		return nil
	}

	return t
}
