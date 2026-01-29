package sqliter

import (
	"fmt"
	"os"

	"github.com/hashicorp/hcl/v2/hclsimple"
)

// Config holds the configuration for the sqliter server.
type Config struct {
	// DataDir is the directory containing the SQLite files to serve.
	// Defaults to "sample_data".
	DataDir string `hcl:"data_dir,optional"`

	// TemplateDir is the directory containing the HTML templates.
	// Defaults to "templates".
	TemplateDir string `hcl:"template_dir,optional"`

	// StickyHeader enables or disables the sticky header feature for the HTML table.
	// Defaults to true.
	StickyHeader bool `hcl:"sticky_header,optional"`

	// AutoRedirectSingleTable enables or disables automatic redirection when a database has only one table.
	// Defaults to true.
	AutoRedirectSingleTable bool `hcl:"auto_redirect_single_table,optional"`

	// StyleSheet is the URL path to the CSS stylesheet.
	// Defaults to "/cssjs/default.css".
	StyleSheet string `hcl:"stylesheet,optional"`

	// Port is the port the server listens on.
	// Defaults to "8080".
	Port string `hcl:"port,optional"`

	// ServeFolder is the folder containing the SQLite files to serve.
	ServeFolder string `hcl:"serve_folder,optional"`

	// SecretsDB is the path to the secrets database.
	SecretsDB string `hcl:"secrets_db,optional"`

	// SecretKey is the path to the secret key file.
	SecretKey string `hcl:"secret_key,optional"`

	// Verbose enables detailed logging for the server.
	Verbose bool `hcl:"verbose,optional"`

	// AutoSelectTb0 enables automatic selection of the first table if none is specified.
	AutoSelectTb0 bool `hcl:"auto_select_tb0,optional"`

	// RowCRUD enables row-level Create, Read, Update, Delete operations.
	// Defaults to false.
	RowCRUD bool `hcl:"row_crud,optional"`

	// LogDir is the directory where logs will be stored.
	// Defaults to "logs".
	LogDir string `hcl:"log_dir,optional"`

	// EnableWASM enables WebAssembly-based client-side rendering
	EnableWASM bool `hcl:"enable_wasm,optional"`

	// WASMBinaryPath is the path to the compiled sqliter.wasm file
	WASMBinaryPath string `hcl:"wasm_binary_path,optional"`
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		DataDir:                 "sample_data",
		TemplateDir:             "templates",
		StickyHeader:            true,
		AutoRedirectSingleTable: true,
		StyleSheet:              "/cssjs/default.css",
		Port:                    "8080",
		ServeFolder:             "sample_data",
		SecretsDB:               "secrets.db",
		SecretKey:               ".secret.key",
		Verbose:                 false,
		AutoSelectTb0:           true,
		RowCRUD:                 false,
		LogDir:                  "logs",
		EnableWASM:              false,
		WASMBinaryPath:          "bin/sqliter.wasm",
	}
}

// LoadConfig loads the configuration from an HCL file.
func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}

	err := hclsimple.DecodeFile(path, nil, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Apply defaults if fields are empty post-load (for fields that might be empty in HCL)
	if cfg.DataDir == "" {
		cfg.DataDir = "sample_data"
	}
	if cfg.TemplateDir == "" {
		cfg.TemplateDir = "templates"
	}
	if cfg.StyleSheet == "" {
		cfg.StyleSheet = "/cssjs/default.css"
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	return cfg, nil
}
