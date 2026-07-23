package config_test

import (
	"path/filepath"
	"testing"

	"github.com/lutrarutra/lazypush/internal/config"
)

func TestConfigDir(t *testing.T) {
	dir := config.ConfigDir()
	if dir == "" {
		t.Fatal("ConfigDir() returned empty string")
	}
	if filepath.Base(dir) != "lazypush" {
		t.Errorf("ConfigDir() = %q, expected basename 'lazypush'", dir)
	}
}

func TestConfigPath(t *testing.T) {
	path := config.ConfigPath()
	if path == "" {
		t.Fatal("ConfigPath() returned empty string")
	}
	if filepath.Base(path) != "config.json" {
		t.Errorf("ConfigPath() = %q, expected basename 'config.json'", path)
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LAZYPUSH_CONFIG_PATH", filepath.Join(dir, "config.json"))

	original := &config.Config{
		APIURL: "https://example.com/v1",
		APIKey: "sk-test-key",
		Model:  "gpt-4",
	}
	if err := config.Save(original); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.APIURL != original.APIURL {
		t.Errorf("APIURL = %q, want %q", loaded.APIURL, original.APIURL)
	}
	if loaded.APIKey != original.APIKey {
		t.Errorf("APIKey = %q, want %q", loaded.APIKey, original.APIKey)
	}
	if loaded.Model != original.Model {
		t.Errorf("Model = %q, want %q", loaded.Model, original.Model)
	}
}

func TestLoadReturnsDefaultsWhenFileMissing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LAZYPUSH_CONFIG_PATH", filepath.Join(dir, "config.json"))

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.APIURL != "https://api.openai.com/v1" {
		t.Errorf("default APIURL = %q, want 'https://api.openai.com/v1'", cfg.APIURL)
	}
	if cfg.Model != "gpt-4o-mini" {
		t.Errorf("default Model = %q, want 'gpt-4o-mini'", cfg.Model)
	}
}

func TestLoadEnvOverride(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "custom.json")
	t.Setenv("LAZYPUSH_CONFIG_PATH", cfgPath)

	original := &config.Config{APIURL: "http://localhost:8080/v1", APIKey: "local-key", Model: "local-model"}
	config.Save(original)

	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.APIURL != "http://localhost:8080/v1" {
		t.Errorf("APIURL = %q", loaded.APIURL)
	}
}
