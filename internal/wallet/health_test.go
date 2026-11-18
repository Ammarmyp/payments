package wallet_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Ammarmyp/payments/internal/wallet"
)

func TestHealthzSucceedsWithoutDatabase(t *testing.T) {
	rec := serve(t, nil, wallet.NewReadyGate(), "/healthz")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	assertBody(t, rec, `{"status":"ok"}`)
}

type stubPinger struct {
	err error
}

func (s stubPinger) Ping(context.Context) error {
	return s.err
}

func TestReadyzSucceedsWhenDatabaseReachableAndReady(t *testing.T) {
	gate := wallet.NewReadyGate()
	gate.SetReady(true)
	rec := serve(t, stubPinger{}, gate, "/readyz")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	assertBody(t, rec, `{"status":"ok"}`)
}

func TestReadyzFailsWhenDatabaseUnreachable(t *testing.T) {
	gate := wallet.NewReadyGate()
	gate.SetReady(true)
	rec := serve(t, stubPinger{err: errors.New("db down")}, gate, "/readyz")

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	assertBody(t, rec, `{"status":"unavailable"}`)
}

func TestReadyzFailsWhenShuttingDown(t *testing.T) {
	gate := wallet.NewReadyGate()
	gate.SetReady(false)
	rec := serve(t, stubPinger{}, gate, "/readyz")

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	assertBody(t, rec, `{"status":"unavailable"}`)
}

type blockingPinger struct{}

func (blockingPinger) Ping(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

func TestReadyzFailsWhenPingTimesOut(t *testing.T) {
	mux := http.NewServeMux()
	gate := wallet.NewReadyGate()
	gate.SetReady(true)
	wallet.NewHealthHandler(blockingPinger{}, gate).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		defer close(done)
		mux.ServeHTTP(rec, req)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("readyz did not return within bounded ping timeout")
	}

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	assertBody(t, rec, `{"status":"unavailable"}`)
}

func serve(t *testing.T, pinger wallet.Pinger, gate *wallet.ReadyGate, path string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	wallet.NewHealthHandler(pinger, gate).Register(mux)

	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func assertBody(t *testing.T, rec *httptest.ResponseRecorder, want string) {
	t.Helper()
	got := strings.TrimSpace(rec.Body.String())
	if got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}
