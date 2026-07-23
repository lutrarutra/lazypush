package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	defaultAPIURL = "https://api.openai.com/v1"
	defaultModel  = "gpt-4o-mini"
)

type Config struct {
	APIURL string `json:"api_url"`
	APIKey string `json:"api_key"`
	Model  string `json:"model"`
}

func ConfigDir() string {
	userDir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(userDir, "lazypush")
}

func ConfigPath() string {
	if p := os.Getenv("LAZYPUSH_CONFIG_PATH"); p != "" {
		return p
	}
	return filepath.Join(ConfigDir(), "config.json")
}

func defaults() *Config {
	return &Config{
		APIURL: defaultAPIURL,
		Model:  defaultModel,
	}
}

func Load() (*Config, error) {
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaults(), nil
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.APIURL == "" {
		cfg.APIURL = defaultAPIURL
	}
	if cfg.Model == "" {
		cfg.Model = defaultModel
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	path := ConfigPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
