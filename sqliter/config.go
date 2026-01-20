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
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		DataDir:     "sample_data",
		TemplateDir: "templates",
	}
}

// LoadConfig loads the configuration from a JSON file.
func LoadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cfg := &Config{}
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

	return cfg, nil
}
