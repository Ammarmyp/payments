package main

import (
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/Ammarmyp/payments/internal/platform/config"
	"github.com/Ammarmyp/payments/internal/platform/logging"
)

func TestLoadConfigDefaultsWhenMinimal(t *testing.T) {
	env := config.New(func(key string) (string, bool) {
		if key == "DATABASE_URL" {
			return "postgres://localhost/wallet", true
		}
		return "", false
	})

	cfg, err := loadConfig(env)
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.Env != "development" {
		t.Fatalf("Env = %q, want development", cfg.Env)
	}
	if cfg.HTTPAddr != ":8081" {
		t.Fatalf("HTTPAddr = %q, want :8081", cfg.HTTPAddr)
	}
	if cfg.ShutdownTimeout != 10*time.Second {
		t.Fatalf("ShutdownTimeout = %v, want 10s", cfg.ShutdownTimeout)
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Fatalf("LogLevel = %v, want info", cfg.LogLevel)
	}
	if cfg.LogFormat != logging.FormatText {
		t.Fatalf("LogFormat = %q, want text", cfg.LogFormat)
	}
	if cfg.DatabaseURL != "postgres://localhost/wallet" {
		t.Fatalf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.DBMaxConns != 10 {
		t.Fatalf("DBMaxConns = %d, want 10", cfg.DBMaxConns)
	}
}

func TestLoadConfigProductionDefaultsJSONLogs(t *testing.T) {
	env := config.New(func(key string) (string, bool) {
		switch key {
		case "APP_ENV":
			return "production", true
		case "DATABASE_URL":
			return "postgres://localhost/wallet", true
		default:
			return "", false
		}
	})

	cfg, err := loadConfig(env)
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.LogFormat != logging.FormatJSON {
		t.Fatalf("LogFormat = %q, want json", cfg.LogFormat)
	}
}

func TestLoadConfigCollectsAllProblems(t *testing.T) {
	env := config.New(func(key string) (string, bool) {
		switch key {
		case "APP_ENV":
			return "staging", true
		case "LOG_LEVEL":
			return "verbose", true
		case "LOG_FORMAT":
			return "yaml", true
		case "DATABASE_MAX_CONNS":
			return "0", true
		default:
			return "", false
		}
	})

	_, err := loadConfig(env)
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	for _, want := range []string{"APP_ENV", "LOG_LEVEL", "LOG_FORMAT", "DATABASE_URL", "DATABASE_MAX_CONNS"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error missing %s: %v", want, err)
		}
	}
}

func TestLoadConfigAcceptsValidOverrides(t *testing.T) {
	env := config.New(func(key string) (string, bool) {
		values := map[string]string{
			"APP_ENV":                 "production",
			"WALLET_HTTP_ADDR":        ":9090",
			"WALLET_SHUTDOWN_TIMEOUT": "5s",
			"LOG_LEVEL":               "DEBUG",
			"LOG_FORMAT":              "text",
			"DATABASE_URL":            "postgres://db/wallet",
			"DATABASE_MAX_CONNS":      "25",
		}
		v, ok := values[key]
		return v, ok
	})

	cfg, err := loadConfig(env)
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.Env != "production" || cfg.HTTPAddr != ":9090" || cfg.ShutdownTimeout != 5*time.Second {
		t.Fatalf("cfg = %#v", cfg)
	}
	if cfg.LogLevel != slog.LevelDebug || cfg.LogFormat != logging.FormatText {
		t.Fatalf("log = %v %q", cfg.LogLevel, cfg.LogFormat)
	}
	if cfg.DatabaseURL != "postgres://db/wallet" || cfg.DBMaxConns != 25 {
		t.Fatalf("db = %q %d", cfg.DatabaseURL, cfg.DBMaxConns)
	}
}
