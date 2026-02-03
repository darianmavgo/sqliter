package sqliter

const MaxRowsBuffer = 200

// Config holds the configuration for the sqliter server.
type Config struct {
	// AutoRedirectSingleTable enables or disables automatic redirection when a database has only one table.
	// Defaults to true.
	AutoRedirectSingleTable bool `hcl:"auto_redirect_single_table,optional"`

	// ServeFolder is the folder containing the SQLite files to serve.
	ServeFolder string `hcl:"serve_folder,optional"`

	// Verbose enables detailed logging for the server.
	Verbose bool `hcl:"verbose,optional"`

	LogDir string `hcl:"log_dir,optional"`

	// BaseURL is the prefix where the app is mounted (e.g. "/tools/sqliter").
	// This is used to inject configuration into the React client.
	BaseURL string `hcl:"base_url,optional"`
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		AutoRedirectSingleTable: true,
		ServeFolder:             "sample_data",
		Verbose:                 false,
		LogDir:                  "logs",
		BaseURL:                 "",
	}
}
