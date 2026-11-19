package wallet

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"
)

const readyPingTimeout = 2 * time.Second

// Pinger reports whether PostgreSQL is reachable.
type Pinger interface {
	Ping(ctx context.Context) error
}

// ReadyGate tracks whether the process should accept traffic.
type ReadyGate struct {
	ready atomic.Bool
}

// NewReadyGate returns a gate that starts not-ready.
func NewReadyGate() *ReadyGate {
	return &ReadyGate{}
}

// SetReady marks the process ready or not-ready for traffic.
func (g *ReadyGate) SetReady(ready bool) {
	g.ready.Store(ready)
}

// Ready reports whether the process should accept traffic.
func (g *ReadyGate) Ready() bool {
	return g.ready.Load()
}

// HealthHandler serves liveness and readiness endpoints.
type HealthHandler struct {
	pinger Pinger
	gate   *ReadyGate
}

// NewHealthHandler wires /healthz and /readyz. pinger may be nil for health-only tests;
// /readyz requires a non-nil pinger.
func NewHealthHandler(pinger Pinger, gate *ReadyGate) *HealthHandler {
	return &HealthHandler{pinger: pinger, gate: gate}
}

// Register mounts health endpoints on mux.
func (h *HealthHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("GET /readyz", h.readyz)
}

func (h *HealthHandler) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, `{"status":"ok"}`+"\n")
}

func (h *HealthHandler) readyz(w http.ResponseWriter, r *http.Request) {
	if h.gate == nil || !h.gate.Ready() || h.pinger == nil {
		writeJSON(w, http.StatusServiceUnavailable, `{"status":"unavailable"}`+"\n")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), readyPingTimeout)
	defer cancel()

	if err := h.pinger.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, `{"status":"unavailable"}`+"\n")
		return
	}

	writeJSON(w, http.StatusOK, `{"status":"ok"}`+"\n")
}

func writeJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}
