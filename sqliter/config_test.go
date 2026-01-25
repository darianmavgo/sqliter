package sqliter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/darianmavgo/sqliter/internal/testutil"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.DataDir != "sample_data" {
		t.Errorf("Expected DataDir to be 'sample_data', got '%s'", cfg.DataDir)
	}
	if cfg.TemplateDir != "templates" {
		t.Errorf("Expected TemplateDir to be 'templates', got '%s'", cfg.TemplateDir)
	}
}

func TestLoadConfig(t *testing.T) {
	content := `data_dir = "test_data"
template_dir = "test_templates"`
	tmpdir := testutil.GetTestOutputDir(t, "config_test")
	// defer os.RemoveAll(tmpdir)

	tmpfile := filepath.Join(tmpdir, "config.hcl")
	if err := os.WriteFile(tmpfile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(tmpfile)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.DataDir != "test_data" {
		t.Errorf("Expected DataDir to be 'test_data', got '%s'", cfg.DataDir)
	}
	if cfg.TemplateDir != "test_templates" {
		t.Errorf("Expected TemplateDir to be 'test_templates', got '%s'", cfg.TemplateDir)
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	content := ``
	tmpdir := testutil.GetTestOutputDir(t, "config_empty_test")
	// defer os.RemoveAll(tmpdir)

	tmpfile := filepath.Join(tmpdir, "config.hcl")
	if err := os.WriteFile(tmpfile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(tmpfile)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.DataDir != "sample_data" {
		t.Errorf("Expected DataDir to be 'sample_data', got '%s'", cfg.DataDir)
	}
	if cfg.TemplateDir != "templates" {
		t.Errorf("Expected TemplateDir to be 'templates', got '%s'", cfg.TemplateDir)
	}
}
