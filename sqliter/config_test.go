package sqliter

import (
	"os"
	"testing"
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
	content := `{"data_dir": "test_data", "template_dir": "test_templates"}`
	tmpfile, err := os.CreateTemp("", "config.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(tmpfile.Name())
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
	content := `{}`
	tmpfile, err := os.CreateTemp("", "config_empty.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(tmpfile.Name())
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
