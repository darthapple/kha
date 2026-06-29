package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	APIKey    string
	UserEmail string
	Pipeline  []string // ordered status names, low→high
}

var defaultPipeline = []string{
	"triage",
	"backlog",
	"scoping",
	"in design",
	"ready for development",
	"in development",
	"in review",
	"testing",
	"shipped",
}

type fileConfig struct {
	APIKey    string   `json:"CLICKUP_API_KEY"`
	UserEmail string   `json:"user_email"`
	Pipeline  []string `json:"pipeline"`
}

func Load() (*Config, error) {
	cfg := &Config{
		Pipeline: defaultPipeline,
	}

	// 1. env var
	cfg.APIKey = os.Getenv("CLICKUP_API_KEY")

	// 2. .env.local in current directory
	if cfg.APIKey == "" {
		if v, err := readEnvFile(".env.local", "CLICKUP_API_KEY"); err == nil && v != "" {
			cfg.APIKey = v
		}
	}

	// 3. ~/.kha/config.json — api_key, user_email, pipeline
	cfgPath := filepath.Join(homeDir(), ".kha", "config.json")
	if data, err := os.ReadFile(cfgPath); err == nil {
		var fc fileConfig
		if json.Unmarshal(data, &fc) == nil {
			if cfg.APIKey == "" && fc.APIKey != "" {
				cfg.APIKey = fc.APIKey
			}
			if fc.UserEmail != "" {
				cfg.UserEmail = fc.UserEmail
			}
			if len(fc.Pipeline) > 0 {
				cfg.Pipeline = fc.Pipeline
			}
		}
	}

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("CLICKUP_API_KEY not found in environment, .env.local, or ~/.kha/config.json")
	}
	return cfg, nil
}

func readEnvFile(path, key string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 && strings.TrimSpace(parts[0]) == key {
			return strings.Trim(strings.TrimSpace(parts[1]), `"`), nil
		}
	}
	return "", nil
}

func homeDir() string {
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	return "."
}
