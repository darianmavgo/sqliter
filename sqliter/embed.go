package sqliter

import (
	"html/template"
)

// Templates are no longer embedded as they have been moved to the repository root.
// Applications should load templates from the filesystem.

func loadEmbeddedTemplates() *template.Template {
	return nil
}
