package sqliter

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.ServeFolder != "sample_data" {
		t.Errorf("Expected ServeFolder to be 'sample_data', got '%s'", cfg.ServeFolder)
	}
	if cfg.AutoRedirectSingleTable != true {
		t.Errorf("Expected AutoRedirectSingleTable to be true, got %v", cfg.AutoRedirectSingleTable)
	}
}
