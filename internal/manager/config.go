package manager

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// PipelineStep maps one ClickUp status to a skill name.
// Skill is empty for statuses that have no automation (backlog, shipped, etc.).
type PipelineStep struct {
	Status string `json:"status"`
	Skill  string `json:"skill"`
}

// Config is the single source of truth for the manager and all spawned containers.
type Config struct {
	ClickUpAPIKey   string
	ClickUpListID   string
	AnthropicAPIKey    string // API key auth (sk-ant-api03-...)
	ClaudeOAuthToken   string // OAuth token from `claude setup-token` → CLAUDE_CODE_OAUTH_TOKEN
	RepoURL         string
	GitToken        string
	SkillImage      string
	NATSUrl         string
	PollInterval    time.Duration
	SlotTTL         time.Duration
	Steps           []PipelineStep
	// Pipeline is the comma-separated ordered status list, derived from Steps.
	// Passed to `kha next --pipeline` for feature advancement.
	Pipeline string
}

var defaultSteps = []PipelineStep{
	{Status: "triage", Skill: "triage"},
	{Status: "backlog"},
	{Status: "scoping", Skill: "scoping"},
	{Status: "in design", Skill: "design"},
	{Status: "ready for development", Skill: "develop"},
	{Status: "in development"},
	{Status: "in review", Skill: "review"},
	{Status: "testing", Skill: "qa"},
	{Status: "shipped"},
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		PollInterval: 60 * time.Second,
		NATSUrl:      "nats://nats:4222",
		SlotTTL:      30 * time.Minute,
		SkillImage:   "ghcr.io/darthapple/kha:latest",
	}

	cfg.ClickUpAPIKey = mustEnv("CLICKUP_API_KEY")
	cfg.ClickUpListID = mustEnv("CLICKUP_LIST_ID")
	cfg.RepoURL = mustEnv("PROJECT_REPO_URL")

	cfg.AnthropicAPIKey = os.Getenv("ANTHROPIC_API_KEY")
	cfg.ClaudeOAuthToken = os.Getenv("CLAUDE_CODE_OAUTH_TOKEN")
	cfg.GitToken = os.Getenv("GIT_TOKEN")
	cfg.NATSUrl = envOr("NATS_URL", cfg.NATSUrl)
	cfg.SkillImage = envOr("SKILL_IMAGE", cfg.SkillImage)

	if v := os.Getenv("POLL_INTERVAL"); v != "" {
		secs, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("POLL_INTERVAL: %w", err)
		}
		cfg.PollInterval = time.Duration(secs) * time.Second
	}

	if v := os.Getenv("SLOT_TTL"); v != "" {
		secs, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("SLOT_TTL: %w", err)
		}
		cfg.SlotTTL = time.Duration(secs) * time.Second
	}

	if v := os.Getenv("PIPELINE_STEPS"); v != "" {
		if err := json.Unmarshal([]byte(v), &cfg.Steps); err != nil {
			return nil, fmt.Errorf("PIPELINE_STEPS: %w", err)
		}
	} else {
		cfg.Steps = defaultSteps
	}

	statuses := make([]string, len(cfg.Steps))
	for i, s := range cfg.Steps {
		statuses[i] = s.Status
	}
	cfg.Pipeline = strings.Join(statuses, ",")

	return cfg, nil
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		fmt.Fprintf(os.Stderr, "error: %s env var required\n", key)
		os.Exit(1)
	}
	return v
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
