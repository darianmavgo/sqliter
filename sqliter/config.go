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

	// Port is the port the server listens on.
	// Defaults to "8080".
	Port string `hcl:"port,optional"`

	// Verbose enables detailed logging for the server.
	Verbose bool `hcl:"verbose,optional"`

	// LogDir is the directory where logs will be stored.
	// Defaults to "logs".
	LogDir string `hcl:"log_dir,optional"`
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		DataDir: "sample_data",
		Port:    "8080",
		Verbose: false,
		LogDir:  "logs",
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
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	return cfg, nil
}
