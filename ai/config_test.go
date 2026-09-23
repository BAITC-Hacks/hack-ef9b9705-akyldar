package ai

import (
	"strings"
	"testing"
	"time"
)

func clearConfigEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{"OPENAI_API_KEY", "OPENAI_MODEL", "AI_TIMEOUT_SECONDS", "AI_MAX_ATTEMPTS", "AI_FORCE_FALLBACK"} {
		t.Setenv(name, "")
	}
}

func TestConfigDefaultsAndOverrides(t *testing.T) {
	clearConfigEnv(t)
	cfg, err := LoadConfigFromEnv()
	if err != nil || cfg.Timeout != 15*time.Second || cfg.MaxAttempts != 2 || cfg.APIKey != "" || cfg.Model != "" || cfg.ForceFallback {
		t.Fatal("invalid defaults", err)
	}
	t.Setenv("OPENAI_API_KEY", " synthetic-key ")
	t.Setenv("OPENAI_MODEL", " configured-model ")
	t.Setenv("AI_TIMEOUT_SECONDS", "5")
	t.Setenv("AI_MAX_ATTEMPTS", "1")
	t.Setenv("AI_FORCE_FALLBACK", "true")
	cfg, err = LoadConfigFromEnv()
	if err != nil || cfg.APIKey != "synthetic-key" || cfg.Model != "configured-model" || cfg.Timeout != 5*time.Second || cfg.MaxAttempts != 1 || !cfg.ForceFallback {
		t.Fatal("invalid override", err)
	}
}

func TestConfigRejectsUnsafeValues(t *testing.T) {
	for _, tc := range []struct{ name, value string }{
		{"AI_TIMEOUT_SECONDS", "0"}, {"AI_TIMEOUT_SECONDS", "61"}, {"AI_TIMEOUT_SECONDS", "999999999999999999999"},
		{"AI_TIMEOUT_SECONDS", "not-a-number"}, {"AI_MAX_ATTEMPTS", "3"}, {"AI_MAX_ATTEMPTS", "0"}, {"AI_MAX_ATTEMPTS", "-1"},
		{"AI_FORCE_FALLBACK", "yes"},
	} {
		t.Run(tc.name+tc.value, func(t *testing.T) {
			clearConfigEnv(t)
			t.Setenv(tc.name, tc.value)
			t.Setenv("OPENAI_API_KEY", "synthetic-secret")
			_, err := LoadConfigFromEnv()
			if err == nil || strings.Contains(err.Error(), "synthetic-secret") {
				t.Fatal("invalid configuration handling")
			}
		})
	}
	for _, cfg := range []Config{{Timeout: time.Second, MaxAttempts: 3}, {Timeout: 0, MaxAttempts: 1}, {Timeout: 61 * time.Second, MaxAttempts: 1}} {
		if _, err := NewService(cfg, nil); err == nil {
			t.Fatal("invalid direct service config accepted")
		}
	}
}
