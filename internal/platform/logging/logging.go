// Package logging builds a process slog logger with service identity and
// context-scoped attributes (attrs live on context, not a logger in context).
package logging

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// Format selects the log encoding.
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

// Options configures New.
type Options struct {
	Writer  io.Writer
	Service string
	Version string
	Level   slog.Level
	Format  Format
}

// New returns a slog logger that always includes service and version.
func New(opts Options) (*slog.Logger, error) {
	if opts.Writer == nil {
		return nil, fmt.Errorf("logging: writer is required")
	}
	if opts.Service == "" {
		return nil, fmt.Errorf("logging: service is required")
	}
	if opts.Version == "" {
		return nil, fmt.Errorf("logging: version is required")
	}

	handlerOpts := &slog.HandlerOptions{Level: opts.Level}

	var base slog.Handler
	switch opts.Format {
	case FormatJSON:
		base = slog.NewJSONHandler(opts.Writer, handlerOpts)
	case FormatText, "":
		handlerOpts.ReplaceAttr = shortTime
		base = slog.NewTextHandler(opts.Writer, handlerOpts)
	default:
		return nil, fmt.Errorf("logging: unknown format %q", opts.Format)
	}

	h := &contextHandler{inner: base}
	return slog.New(h).With(
		slog.String("service", opts.Service),
		slog.String("version", opts.Version),
	), nil
}

// shortTime keeps terminal text logs as clock time only (HH:MM:SS).
func shortTime(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey && len(groups) == 0 && a.Value.Kind() == slog.KindTime {
		return slog.String(slog.TimeKey, a.Value.Time().Format("15:04:05"))
	}
	return a
}

// ParseLevel accepts debug/info/warn/error in any letter case.
func ParseLevel(s string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("logging: unknown level %q", s)
	}
}

// ParseFormat accepts text or json in any letter case.
func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "text", "":
		return FormatText, nil
	case "json":
		return FormatJSON, nil
	default:
		return "", fmt.Errorf("logging: unknown format %q", s)
	}
}
