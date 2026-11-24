package logging

import (
	"context"
	"log/slog"
)

type attrsKey struct{}

// With returns a child context that carries attrs for InfoContext and friends.
func With(ctx context.Context, attrs ...slog.Attr) context.Context {
	existing := Attrs(ctx)
	merged := make([]slog.Attr, 0, len(existing)+len(attrs))
	merged = append(merged, existing...)
	merged = append(merged, attrs...)
	return context.WithValue(ctx, attrsKey{}, merged)
}

// Attrs returns log attributes previously stored on ctx.
func Attrs(ctx context.Context) []slog.Attr {
	v, _ := ctx.Value(attrsKey{}).([]slog.Attr)
	return v
}

type contextHandler struct {
	inner slog.Handler
}

func (h *contextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *contextHandler) Handle(ctx context.Context, r slog.Record) error {
	attrs := Attrs(ctx)
	if len(attrs) > 0 {
		r = r.Clone()
		r.AddAttrs(attrs...)
	}
	return h.inner.Handle(ctx, r)
}

func (h *contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &contextHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *contextHandler) WithGroup(name string) slog.Handler {
	return &contextHandler{inner: h.inner.WithGroup(name)}
}
