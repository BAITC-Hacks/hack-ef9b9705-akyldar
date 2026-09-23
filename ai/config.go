package ai

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	APIKey        string
	Model         string
	Timeout       time.Duration
	MaxAttempts   int
	ForceFallback bool
}

func DefaultConfig() Config {
	return Config{Timeout: 15 * time.Second, MaxAttempts: 2}
}

// LoadConfigFromEnv reads process variables; it never reads .env files.
func LoadConfigFromEnv() (Config, error) {
	cfg := DefaultConfig()
	cfg.APIKey = strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	cfg.Model = strings.TrimSpace(os.Getenv("OPENAI_MODEL"))
	if value := os.Getenv("AI_TIMEOUT_SECONDS"); value != "" {
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 || n > 60 {
			return Config{}, fmt.Errorf("AI_TIMEOUT_SECONDS must be an integer from 1 to 60")
		}
		cfg.Timeout = time.Duration(n) * time.Second
	}
	if value := os.Getenv("AI_MAX_ATTEMPTS"); value != "" {
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 || n > 2 {
			return Config{}, fmt.Errorf("AI_MAX_ATTEMPTS must be 1 or 2")
		}
		cfg.MaxAttempts = n
	}
	if value := os.Getenv("AI_FORCE_FALLBACK"); value != "" {
		if value != "true" && value != "false" {
			return Config{}, fmt.Errorf("AI_FORCE_FALLBACK must be true or false")
		}
		cfg.ForceFallback = value == "true"
	}
	return cfg, cfg.validate()
}

func (cfg Config) validate() error {
	if cfg.Timeout <= 0 || cfg.Timeout > 60*time.Second {
		return fmt.Errorf("AI timeout must be positive and at most 60 seconds")
	}
	if cfg.MaxAttempts < 1 || cfg.MaxAttempts > 2 {
		return fmt.Errorf("AI maximum attempts must be 1 or 2")
	}
	return nil
}
