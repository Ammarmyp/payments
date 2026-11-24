package config_test

import (
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/Ammarmyp/payments/internal/platform/config"
	"github.com/Ammarmyp/payments/internal/platform/logging"
)

func loadLogConfig(getenv func(string) (string, bool)) (config.LogConfig, error) {
	c := config.New(getenv)
	cfg := config.LogConfig{
		Level:  config.Parse(c, "LOG_LEVEL", slog.LevelInfo, logging.ParseLevel),
		Format: config.Parse(c, "LOG_FORMAT", logging.FormatText, logging.ParseFormat),
	}
	return cfg, c.Err()
}

func TestLogConfigDefaultsWhenUnset(t *testing.T) {
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("LOG_FORMAT", "")

	cfg, err := loadLogConfig(os.LookupEnv)
	if err != nil {
		t.Fatalf("loadLogConfig: %v", err)
	}
	if cfg.Level != slog.LevelInfo {
		t.Fatalf("Level = %v, want info", cfg.Level)
	}
	if cfg.Format != logging.FormatText {
		t.Fatalf("Format = %q, want %q", cfg.Format, logging.FormatText)
	}
}

func TestLogConfigCollectsAllInvalidSettings(t *testing.T) {
	t.Setenv("LOG_LEVEL", "verbose")
	t.Setenv("LOG_FORMAT", "yaml")

	_, err := loadLogConfig(os.LookupEnv)
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "LOG_LEVEL") {
		t.Fatalf("error missing LOG_LEVEL: %v", err)
	}
	if !strings.Contains(msg, "LOG_FORMAT") {
		t.Fatalf("error missing LOG_FORMAT: %v", err)
	}
}

func TestLogConfigAcceptsValidOverrides(t *testing.T) {
	t.Setenv("LOG_LEVEL", "DEBUG")
	t.Setenv("LOG_FORMAT", "json")

	cfg, err := loadLogConfig(os.LookupEnv)
	if err != nil {
		t.Fatalf("loadLogConfig: %v", err)
	}
	if cfg.Level != slog.LevelDebug {
		t.Fatalf("Level = %v, want debug", cfg.Level)
	}
	if cfg.Format != logging.FormatJSON {
		t.Fatalf("Format = %q, want %q", cfg.Format, logging.FormatJSON)
	}
}
