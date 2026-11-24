package config_test

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/Ammarmyp/payments/internal/platform/config"
	"github.com/Ammarmyp/payments/internal/platform/logging"
)

func TestEnvDefaultsAndCollectAll(t *testing.T) {
	t.Run("defaults_when_unset", func(t *testing.T) {
		env := config.New(func(string) (string, bool) { return "", false })
		level := config.Parse(env, "LOG_LEVEL", slog.LevelInfo, logging.ParseLevel)
		format := config.Parse(env, "LOG_FORMAT", logging.FormatText, logging.ParseFormat)
		if err := env.Err(); err != nil {
			t.Fatalf("Err: %v", err)
		}
		if level != slog.LevelInfo {
			t.Fatalf("Level = %v, want info", level)
		}
		if format != logging.FormatText {
			t.Fatalf("Format = %q, want %q", format, logging.FormatText)
		}
	})

	t.Run("collects_all_invalid", func(t *testing.T) {
		values := map[string]string{
			"LOG_LEVEL":  "verbose",
			"LOG_FORMAT": "yaml",
		}
		env := config.New(func(key string) (string, bool) {
			v, ok := values[key]
			return v, ok
		})
		_ = config.Parse(env, "LOG_LEVEL", slog.LevelInfo, logging.ParseLevel)
		_ = config.Parse(env, "LOG_FORMAT", logging.FormatText, logging.ParseFormat)
		err := env.Err()
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
	})

	t.Run("required_missing", func(t *testing.T) {
		env := config.New(func(string) (string, bool) { return "", false })
		_ = env.Required("DATABASE_URL")
		err := env.Err()
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "DATABASE_URL") {
			t.Fatalf("error missing DATABASE_URL: %v", err)
		}
	})
}
