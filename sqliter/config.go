package sqliter

import (
	"encoding/json"
	"os"
)

// Config holds the configuration for the sqliter server.
type Config struct {
	// DataDir is the directory containing the SQLite files to serve.
	// Defaults to "sample_data".
	DataDir string `json:"data_dir"`

	// TemplateDir is the directory containing the HTML templates.
	// Defaults to "templates".
	TemplateDir string `json:"template_dir"`

	// StickyHeader enables or disables the sticky header feature for the HTML table.
	// Defaults to true.
	StickyHeader bool `json:"sticky_header"`

	// AutoRedirectSingleTable enables or disables automatic redirection when a database has only one table.
	// Defaults to true.
	AutoRedirectSingleTable bool `json:"auto_redirect_single_table"`

	// StyleSheet is the URL path to the CSS stylesheet.
	// Defaults to "/cssjs/default.css".
	StyleSheet string `json:"stylesheet"`

	// Verbose enables detailed logging for the server.
	Verbose bool `json:"verbose"`
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		DataDir:                 "sample_data",
		TemplateDir:             "templates",
		StickyHeader:            true,
		AutoRedirectSingleTable: true,
		StyleSheet:              "/cssjs/default.css",
		Verbose:                 false,
	}
}

// LoadConfig loads the configuration from a JSON file.
func LoadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cfg := DefaultConfig()
	if err := json.NewDecoder(f).Decode(cfg); err != nil {
		return nil, err
	}

	// Apply defaults if fields are empty
	if cfg.DataDir == "" {
		cfg.DataDir = "sample_data"
	}
	if cfg.TemplateDir == "" {
		cfg.TemplateDir = "templates"
	}
	if cfg.StyleSheet == "" {
		cfg.StyleSheet = "/cssjs/default.css"
	}

	return cfg, nil
}

// ExportCurrentConfig returns the JSON representation of the current configuration.
func (c *Config) ExportCurrentConfig() (string, error) {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ExportDefaultConfig returns the JSON representation of the default configuration.
func ExportDefaultConfig() (string, error) {
	return DefaultConfig().ExportCurrentConfig()
}
