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

func TestEntriesOnNewWalletAreEmpty(t *testing.T) {
	h := newAPIHandler(t)
	created := createWallet(t, h, uuid.New().String(), "USD")

	rec := apiRequest(t, h, http.MethodGet, "/v1/wallets/"+created.ID+"/entries", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d\nbody=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	page := decodeEntriesPage(t, rec)
	if len(page.Entries) != 0 {
		t.Fatalf("entries = %#v, want empty", page.Entries)
	}
	if page.NextCursor != "" {
		t.Fatalf("next_cursor = %q, want empty", page.NextCursor)
	}
}

func TestEntriesNewestFirstAfterDeposits(t *testing.T) {
	h := newAPIHandler(t)
	created := createWallet(t, h, uuid.New().String(), "USD")
	deposit(t, h, created.ID, 100, "USD")
	deposit(t, h, created.ID, 200, "USD")
	deposit(t, h, created.ID, 300, "USD")

	rec := apiRequest(t, h, http.MethodGet, "/v1/wallets/"+created.ID+"/entries", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d\nbody=%s", rec.Code, rec.Body.String())
	}
	page := decodeEntriesPage(t, rec)
	if len(page.Entries) != 3 {
		t.Fatalf("len(entries) = %d, want 3\n%#v", len(page.Entries), page.Entries)
	}
	if page.NextCursor != "" {
		t.Fatalf("next_cursor = %q, want empty", page.NextCursor)
	}
	amounts := []int64{page.Entries[0].Amount, page.Entries[1].Amount, page.Entries[2].Amount}
	if amounts[0] != 300 || amounts[1] != 200 || amounts[2] != 100 {
		t.Fatalf("amounts newest-first = %v, want [300 200 100]", amounts)
	}
	for _, e := range page.Entries {
		if e.Direction != "credit" || e.Currency != "USD" || e.TransactionID == "" || e.CreatedAt == "" {
			t.Fatalf("entry shape: %#v", e)
		}
		if e.AccountID != "" {
			t.Fatalf("account_id leaked: %#v", e)
		}
		if e.ID == "" {
			t.Fatal("entry id empty")
		}
	}
	firstID, err := page.Entries[0].ID.Int64()
	if err != nil {
		t.Fatalf("id: %v", err)
	}
	secondID, err := page.Entries[1].ID.Int64()
	if err != nil {
		t.Fatalf("id: %v", err)
	}
	if firstID <= secondID {
		t.Fatalf("ids not descending: %d then %d", firstID, secondID)
	}
}

func TestEntriesPagesWithCursor(t *testing.T) {
	h := newAPIHandler(t)
	created := createWallet(t, h, uuid.New().String(), "ETB")
	for amount := int64(1); amount <= 5; amount++ {
		deposit(t, h, created.ID, amount, "ETB")
	}

	first := apiRequest(t, h, http.MethodGet, "/v1/wallets/"+created.ID+"/entries?limit=2", nil)
	if first.Code != http.StatusOK {
		t.Fatalf("first status = %d\nbody=%s", first.Code, first.Body.String())
	}
	page1 := decodeEntriesPage(t, first)
	if len(page1.Entries) != 2 || page1.NextCursor == "" {
		t.Fatalf("page1 = %#v, want 2 entries and next_cursor", page1)
	}
	if page1.Entries[0].Amount != 5 || page1.Entries[1].Amount != 4 {
		t.Fatalf("page1 amounts = [%d %d], want [5 4]", page1.Entries[0].Amount, page1.Entries[1].Amount)
	}

	second := apiRequest(t, h, http.MethodGet, "/v1/wallets/"+created.ID+"/entries?limit=2&cursor="+page1.NextCursor, nil)
	if second.Code != http.StatusOK {
		t.Fatalf("second status = %d\nbody=%s", second.Code, second.Body.String())
	}
	page2 := decodeEntriesPage(t, second)
	if len(page2.Entries) != 2 || page2.NextCursor == "" {
		t.Fatalf("page2 = %#v, want 2 entries and next_cursor", page2)
	}
	if page2.Entries[0].Amount != 3 || page2.Entries[1].Amount != 2 {
		t.Fatalf("page2 amounts = [%d %d], want [3 2]", page2.Entries[0].Amount, page2.Entries[1].Amount)
	}

	third := apiRequest(t, h, http.MethodGet, "/v1/wallets/"+created.ID+"/entries?limit=2&cursor="+page2.NextCursor, nil)
	if third.Code != http.StatusOK {
		t.Fatalf("third status = %d\nbody=%s", third.Code, third.Body.String())
	}
	page3 := decodeEntriesPage(t, third)
	if len(page3.Entries) != 1 || page3.NextCursor != "" {
		t.Fatalf("page3 = %#v, want 1 entry and empty next_cursor", page3)
	}
	if page3.Entries[0].Amount != 1 {
		t.Fatalf("page3 amount = %d, want 1", page3.Entries[0].Amount)
	}
}

func TestEntriesRejectsInvalidLimit(t *testing.T) {
	h := newAPIHandler(t)
	created := createWallet(t, h, uuid.New().String(), "USD")

	rec := apiRequest(t, h, http.MethodGet, "/v1/wallets/"+created.ID+"/entries?limit=abc", nil)
	assertAPIError(t, rec, http.StatusUnprocessableEntity, "invalid_input", "limit")

	rec = apiRequest(t, h, http.MethodGet, "/v1/wallets/"+created.ID+"/entries?limit=0", nil)
	assertAPIError(t, rec, http.StatusUnprocessableEntity, "invalid_input", "limit")

	rec = apiRequest(t, h, http.MethodGet, "/v1/wallets/"+created.ID+"/entries?limit=101", nil)
	assertAPIError(t, rec, http.StatusUnprocessableEntity, "invalid_input", "limit")
}

func TestEntriesRejectsInvalidCursor(t *testing.T) {
	h := newAPIHandler(t)
	created := createWallet(t, h, uuid.New().String(), "USD")

	rec := apiRequest(t, h, http.MethodGet, "/v1/wallets/"+created.ID+"/entries?cursor=not-a-cursor", nil)
	assertAPIError(t, rec, http.StatusUnprocessableEntity, "invalid_input", "cursor")
}

func TestEntriesMissingWalletIs404(t *testing.T) {
	h := newAPIHandler(t)
	rec := apiRequest(t, h, http.MethodGet, "/v1/wallets/"+uuid.New().String()+"/entries", nil)
	assertAPIError(t, rec, http.StatusNotFound, "not_found", "")
}

func TestEntriesSystemAccountIs404(t *testing.T) {
	h := newAPIHandler(t)
	store := testStore(t)
	settlement, err := store.SystemAccount(t.Context(), wallet.SettlementAccount, "USD")
	if err != nil {
		t.Fatalf("SystemAccount: %v", err)
	}

	rec := apiRequest(t, h, http.MethodGet, "/v1/wallets/"+settlement.ID.String()+"/entries", nil)
	assertAPIError(t, rec, http.StatusNotFound, "not_found", "")
}

func TestDepositReturns201(t *testing.T) {
	h := newAPIHandler(t)
	created := createWallet(t, h, uuid.New().String(), "USD")

	rec := apiRequest(t, h, http.MethodPost, "/v1/wallets/"+created.ID+"/deposits", map[string]any{
		"amount":   1500,
		"currency": "USD",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d\nbody=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var body struct {
		ID         string `json:"id"`
		Amount     int64  `json:"amount"`
		Currency   string `json:"currency"`
		Entries    any    `json:"entries"`
		Settlement any    `json:"settlement"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v\nbody=%s", err, rec.Body.String())
	}
	if _, err := uuid.Parse(body.ID); err != nil {
		t.Fatalf("id: %v", err)
	}
	if body.Amount != 1500 || body.Currency != "USD" {
		t.Fatalf("money = {%d %q}, want {1500 USD}", body.Amount, body.Currency)
	}
	if body.Entries != nil || body.Settlement != nil {
		t.Fatalf("response leaked ledger internals: %#v", body)
	}
}

func TestDepositIncreasesBalance(t *testing.T) {
	h := newAPIHandler(t)
	created := createWallet(t, h, uuid.New().String(), "ETB")

	dep := apiRequest(t, h, http.MethodPost, "/v1/wallets/"+created.ID+"/deposits", map[string]any{
		"amount":   250,
		"currency": "ETB",
	})
	if dep.Code != http.StatusCreated {
		t.Fatalf("deposit status = %d\nbody=%s", dep.Code, dep.Body.String())
	}

	rec := apiRequest(t, h, http.MethodGet, "/v1/wallets/"+created.ID+"/balance", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("balance status = %d\nbody=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Amount   int64  `json:"amount"`
		Currency string `json:"currency"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v\nbody=%s", err, rec.Body.String())
	}
	if body.Amount != 250 || body.Currency != "ETB" {
		t.Fatalf("balance = %#v, want amount=250 currency=ETB", body)
	}
}

func TestDepositRejectsNonPositiveAmount(t *testing.T) {
	h := newAPIHandler(t)
	created := createWallet(t, h, uuid.New().String(), "USD")

	rec := apiRequest(t, h, http.MethodPost, "/v1/wallets/"+created.ID+"/deposits", map[string]any{
		"amount":   0,
		"currency": "USD",
	})
	assertAPIError(t, rec, http.StatusUnprocessableEntity, "invalid_input", "amount")
}

func TestDepositRejectsAmountAboveMax(t *testing.T) {
	h := newAPIHandler(t)
	created := createWallet(t, h, uuid.New().String(), "USD")

	rec := apiRequest(t, h, http.MethodPost, "/v1/wallets/"+created.ID+"/deposits", map[string]any{
		"amount":   wallet.MaxAmount + 1,
		"currency": "USD",
	})
	assertAPIError(t, rec, http.StatusUnprocessableEntity, "invalid_input", "amount")
}

func TestDepositRejectsCurrencyMismatch(t *testing.T) {
	h := newAPIHandler(t)
	created := createWallet(t, h, uuid.New().String(), "USD")

	rec := apiRequest(t, h, http.MethodPost, "/v1/wallets/"+created.ID+"/deposits", map[string]any{
		"amount":   100,
		"currency": "ETB",
	})
	assertAPIError(t, rec, http.StatusUnprocessableEntity, "invalid_input", "currency")
}

func TestDepositRepeatedCreatesDistinctTransactions(t *testing.T) {
	h := newAPIHandler(t)
	created := createWallet(t, h, uuid.New().String(), "USD")
	body := map[string]any{"amount": 10, "currency": "USD"}

	first := apiRequest(t, h, http.MethodPost, "/v1/wallets/"+created.ID+"/deposits", body)
	second := apiRequest(t, h, http.MethodPost, "/v1/wallets/"+created.ID+"/deposits", body)
	if first.Code != http.StatusCreated || second.Code != http.StatusCreated {
		t.Fatalf("statuses = %d, %d\nbodies=%s\n%s", first.Code, second.Code, first.Body.String(), second.Body.String())
	}

	var a, b struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &a); err != nil {
		t.Fatalf("decode first: %v", err)
	}
	if err := json.Unmarshal(second.Body.Bytes(), &b); err != nil {
		t.Fatalf("decode second: %v", err)
	}
	if a.ID == "" || a.ID == b.ID {
		t.Fatalf("expected distinct transaction ids, got %q and %q", a.ID, b.ID)
	}
}

func TestDepositMissingWalletIs404(t *testing.T) {
	h := newAPIHandler(t)
	rec := apiRequest(t, h, http.MethodPost, "/v1/wallets/"+uuid.New().String()+"/deposits", map[string]any{
		"amount":   100,
		"currency": "USD",
	})
	assertAPIError(t, rec, http.StatusNotFound, "not_found", "")
}

func TestDepositSystemAccountIs404(t *testing.T) {
	h := newAPIHandler(t)
	store := testStore(t)
	settlement, err := store.SystemAccount(t.Context(), wallet.SettlementAccount, "USD")
	if err != nil {
		t.Fatalf("SystemAccount: %v", err)
	}

	rec := apiRequest(t, h, http.MethodPost, "/v1/wallets/"+settlement.ID.String()+"/deposits", map[string]any{
		"amount":   100,
		"currency": "USD",
	})
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

type entriesPageBody struct {
	Entries    []entryBody `json:"entries"`
	NextCursor string      `json:"next_cursor"`
}

type entryBody struct {
	ID            json.Number `json:"id"`
	Direction     string      `json:"direction"`
	Amount        int64       `json:"amount"`
	Currency      string      `json:"currency"`
	TransactionID string      `json:"transaction_id"`
	CreatedAt     string      `json:"created_at"`
	AccountID     string      `json:"account_id"`
}

func decodeEntriesPage(t *testing.T, rec *httptest.ResponseRecorder) entriesPageBody {
	t.Helper()
	var page entriesPageBody
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode entries: %v\nbody=%s", err, rec.Body.String())
	}
	if page.Entries == nil {
		t.Fatal("entries: want [] not null")
	}
	return page
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

func deposit(t *testing.T, h http.Handler, walletID string, amount int64, currency string) {
	t.Helper()
	rec := apiRequest(t, h, http.MethodPost, "/v1/wallets/"+walletID+"/deposits", map[string]any{
		"amount":   amount,
		"currency": currency,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("deposit status = %d\nbody=%s", rec.Code, rec.Body.String())
	}
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
