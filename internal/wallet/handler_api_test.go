package wallet_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Ammarmyp/payments/internal/platform/httpx"
	"github.com/Ammarmyp/payments/internal/platform/logging"
	"github.com/Ammarmyp/payments/internal/wallet"
	"github.com/google/uuid"
)

func TestCreateWalletReturns201(t *testing.T) {
	h := newAPIHandler(t)
	ownerID := uuid.New()

	rec := apiRequest(t, h, http.MethodPost, "/v1/wallets", map[string]string{
		"owner_id": ownerID.String(),
		"currency": "USD",
	})

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d\nbody=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	body := decodeWallet(t, rec)
	if _, err := uuid.Parse(body.ID); err != nil {
		t.Fatalf("id: %v", err)
	}
	if body.OwnerID != ownerID.String() {
		t.Fatalf("owner_id = %q, want %q", body.OwnerID, ownerID.String())
	}
	if body.Currency != "USD" {
		t.Fatalf("currency = %q, want USD", body.Currency)
	}
	if body.CreatedAt == "" {
		t.Fatal("created_at: expected non-empty")
	}
	if body.Name != "" || body.Kind != "" || body.NormalBalance != "" {
		t.Fatalf("response leaked internal fields: %#v", body)
	}
}

func TestCreateWalletRejectsInvalidOwnerID(t *testing.T) {
	h := newAPIHandler(t)
	rec := apiRequest(t, h, http.MethodPost, "/v1/wallets", map[string]string{
		"owner_id": "not-a-uuid",
		"currency": "USD",
	})
	assertAPIError(t, rec, http.StatusUnprocessableEntity, "invalid_input", "owner_id")
}

func TestCreateWalletRejectsInvalidCurrency(t *testing.T) {
	h := newAPIHandler(t)
	rec := apiRequest(t, h, http.MethodPost, "/v1/wallets", map[string]string{
		"owner_id": uuid.New().String(),
		"currency": "usd",
	})
	assertAPIError(t, rec, http.StatusUnprocessableEntity, "invalid_input", "currency")
}

func TestCreateWalletRejectsUnsupportedCurrency(t *testing.T) {
	h := newAPIHandler(t)
	rec := apiRequest(t, h, http.MethodPost, "/v1/wallets", map[string]string{
		"owner_id": uuid.New().String(),
		"currency": "ZZZ",
	})
	assertAPIError(t, rec, http.StatusUnprocessableEntity, "invalid_input", "currency")
}

func TestCreateWalletConflictOnDuplicateOwnerCurrency(t *testing.T) {
	h := newAPIHandler(t)
	ownerID := uuid.New().String()
	body := map[string]string{"owner_id": ownerID, "currency": "ETB"}

	first := apiRequest(t, h, http.MethodPost, "/v1/wallets", body)
	if first.Code != http.StatusCreated {
		t.Fatalf("first create status = %d\nbody=%s", first.Code, first.Body.String())
	}
	second := apiRequest(t, h, http.MethodPost, "/v1/wallets", body)
	assertAPIError(t, second, http.StatusConflict, "conflict", "")
}

func TestGetWalletReturns200(t *testing.T) {
	h := newAPIHandler(t)
	created := createWallet(t, h, uuid.New().String(), "USD")

	rec := apiRequest(t, h, http.MethodGet, "/v1/wallets/"+created.ID, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d\nbody=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	got := decodeWallet(t, rec)
	if got.ID != created.ID || got.OwnerID != created.OwnerID || got.Currency != "USD" {
		t.Fatalf("got %#v, want %#v", got, created)
	}
}

func TestGetWalletMissingIs404(t *testing.T) {
	h := newAPIHandler(t)
	rec := apiRequest(t, h, http.MethodGet, "/v1/wallets/"+uuid.New().String(), nil)
	assertAPIError(t, rec, http.StatusNotFound, "not_found", "")
}

func TestGetWalletSystemAccountIs404(t *testing.T) {
	h := newAPIHandler(t)
	store := testStore(t)
	settlement, err := store.SystemAccount(t.Context(), wallet.SettlementAccount, "USD")
	if err != nil {
		t.Fatalf("SystemAccount: %v", err)
	}

	rec := apiRequest(t, h, http.MethodGet, "/v1/wallets/"+settlement.ID.String(), nil)
	assertAPIError(t, rec, http.StatusNotFound, "not_found", "")
}

func TestGetWalletNonUUIDPathIs404(t *testing.T) {
	h := newAPIHandler(t)
	rec := apiRequest(t, h, http.MethodGet, "/v1/wallets/not-a-uuid", nil)
	assertAPIError(t, rec, http.StatusNotFound, "not_found", "")
}

func TestBalanceOnNewWalletIsZero(t *testing.T) {
	h := newAPIHandler(t)
	created := createWallet(t, h, uuid.New().String(), "USD")

	rec := apiRequest(t, h, http.MethodGet, "/v1/wallets/"+created.ID+"/balance", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d\nbody=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var body struct {
		Amount   int64  `json:"amount"`
		Currency string `json:"currency"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v\nbody=%s", err, rec.Body.String())
	}
	if body.Amount != 0 || body.Currency != "USD" {
		t.Fatalf("balance = %#v, want amount=0 currency=USD", body)
	}
}

func TestBalanceMissingIs404(t *testing.T) {
	h := newAPIHandler(t)
	rec := apiRequest(t, h, http.MethodGet, "/v1/wallets/"+uuid.New().String()+"/balance", nil)
	assertAPIError(t, rec, http.StatusNotFound, "not_found", "")
}

func TestBalanceSystemAccountIs404(t *testing.T) {
	h := newAPIHandler(t)
	store := testStore(t)
	settlement, err := store.SystemAccount(t.Context(), wallet.SettlementAccount, "USD")
	if err != nil {
		t.Fatalf("SystemAccount: %v", err)
	}

	rec := apiRequest(t, h, http.MethodGet, "/v1/wallets/"+settlement.ID.String()+"/balance", nil)
	assertAPIError(t, rec, http.StatusNotFound, "not_found", "")
}

func TestUnmatchedRouteReturnsJSON404(t *testing.T) {
	h := newAPIHandler(t)
	rec := apiRequest(t, h, http.MethodGet, "/v1/nope", nil)
	assertAPIError(t, rec, http.StatusNotFound, "not_found", "")
}

type walletBody struct {
	ID            string `json:"id"`
	OwnerID       string `json:"owner_id"`
	Currency      string `json:"currency"`
	CreatedAt     string `json:"created_at"`
	Name          string `json:"name"`
	Kind          string `json:"kind"`
	NormalBalance string `json:"normal_balance"`
}

func createWallet(t *testing.T, h http.Handler, ownerID, currency string) walletBody {
	t.Helper()
	rec := apiRequest(t, h, http.MethodPost, "/v1/wallets", map[string]string{
		"owner_id": ownerID,
		"currency": currency,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d\nbody=%s", rec.Code, rec.Body.String())
	}
	return decodeWallet(t, rec)
}

func decodeWallet(t *testing.T, rec *httptest.ResponseRecorder) walletBody {
	t.Helper()
	var body walletBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode wallet: %v\nbody=%s", err, rec.Body.String())
	}
	return body
}

func assertAPIError(t *testing.T, rec *httptest.ResponseRecorder, status int, code, field string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d\nbody=%s", rec.Code, status, rec.Body.String())
	}
	var body struct {
		Error struct {
			Code   string            `json:"code"`
			Fields map[string]string `json:"fields"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error: %v\nbody=%s", err, rec.Body.String())
	}
	if body.Error.Code != code {
		t.Fatalf("code = %q, want %q\nbody=%s", body.Error.Code, code, rec.Body.String())
	}
	if field != "" {
		if _, ok := body.Error.Fields[field]; !ok {
			t.Fatalf("fields missing %q: %#v", field, body.Error.Fields)
		}
	}
}

func newAPIHandler(t *testing.T) http.Handler {
	t.Helper()
	store := testStore(t)
	svc := wallet.NewService(store)
	mux := http.NewServeMux()
	wallet.NewHandler(svc).Register(mux)
	mux.HandleFunc("/", httpx.NotFound)

	log, err := logging.New(logging.Options{
		Writer:  io.Discard,
		Service: "wallet",
		Version: "dev",
		Level:   slog.LevelInfo,
		Format:  logging.FormatJSON,
	})
	if err != nil {
		t.Fatalf("logging.New: %v", err)
	}
	return httpx.Chain(mux, httpx.RequestID, httpx.AccessLog(log), httpx.Recover(log))
}

func apiRequest(t *testing.T, h http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		r = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, r)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Header().Get(httpx.RequestIDHeader) == "" {
		t.Fatalf("missing %s on response", httpx.RequestIDHeader)
	}
	return rec
}
