package logging_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/Ammarmyp/payments/internal/platform/logging"
)

func TestNewIncludesServiceAndVersionOnEveryLine(t *testing.T) {
	var buf bytes.Buffer
	log, err := logging.New(logging.Options{
		Writer:  &buf,
		Service: "wallet",
		Version: "dev",
		Level:   slog.LevelInfo,
		Format:  logging.FormatJSON,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	log.Info("boot")

	var line map[string]any
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
		t.Fatalf("decode log: %v\n%s", err, buf.String())
	}
	if line["service"] != "wallet" {
		t.Fatalf("service = %v, want wallet", line["service"])
	}
	if line["version"] != "dev" {
		t.Fatalf("version = %v, want dev", line["version"])
	}
	if line["msg"] != "boot" {
		t.Fatalf("msg = %v, want boot", line["msg"])
	}
}

func TestParseLevelAcceptsCommonLetterCase(t *testing.T) {
	tests := []struct {
		in   string
		want slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"INFO", slog.LevelInfo},
		{"Warn", slog.LevelWarn},
		{"ERROR", slog.LevelError},
	}
	for _, tt := range tests {
		got, err := logging.ParseLevel(tt.in)
		if err != nil {
			t.Fatalf("ParseLevel(%q): %v", tt.in, err)
		}
		if got != tt.want {
			t.Fatalf("ParseLevel(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestTextFormatUsesShortTimeOfDay(t *testing.T) {
	var buf bytes.Buffer
	log, err := logging.New(logging.Options{
		Writer:  &buf,
		Service: "wallet",
		Version: "dev",
		Level:   slog.LevelInfo,
		Format:  logging.FormatText,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	log.Info("hello")
	out := buf.String()
	// slog text default uses RFC3339-ish dates; short clock is HH:MM:SS without a date prefix.
	if strings.Contains(out, "T") && strings.Contains(out, "Z") {
		t.Fatalf("expected short time-of-day, got %q", out)
	}
	if !strings.Contains(out, "hello") {
		t.Fatalf("missing message: %q", out)
	}
}

func TestContextAttributesMergeIntoRecords(t *testing.T) {
	var buf bytes.Buffer
	log, err := logging.New(logging.Options{
		Writer:  &buf,
		Service: "wallet",
		Version: "dev",
		Level:   slog.LevelInfo,
		Format:  logging.FormatJSON,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := logging.With(context.Background(), slog.String("request_id", "req-1"))
	log.InfoContext(ctx, "handled")

	var line map[string]any
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
		t.Fatalf("decode log: %v\n%s", err, buf.String())
	}
	if line["request_id"] != "req-1" {
		t.Fatalf("request_id = %v, want req-1", line["request_id"])
	}
}
