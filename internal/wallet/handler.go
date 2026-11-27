package wallet

import (
	"net/http"
	"time"

	"github.com/Ammarmyp/payments/internal/platform/httpx"
	"github.com/google/uuid"
)

// Handler exposes Wallet HTTP routes.
type Handler struct {
	svc *Service
}

// NewHandler returns an HTTP Handler backed by svc.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Register mounts wallet routes on mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.Handle("POST /v1/wallets", httpx.HandlerFunc(h.create))
	mux.Handle("GET /v1/wallets/{id}", httpx.HandlerFunc(h.get))
	mux.Handle("GET /v1/wallets/{id}/balance", httpx.HandlerFunc(h.balance))
	mux.Handle("POST /v1/wallets/{id}/deposits", httpx.HandlerFunc(h.deposit))
	mux.Handle("GET /v1/wallets/{id}/entries", httpx.HandlerFunc(h.entries))
}

type createWalletRequest struct {
	OwnerID  string `json:"owner_id"`
	Currency string `json:"currency"`
}

type walletResponse struct {
	ID        string    `json:"id"`
	OwnerID   string    `json:"owner_id"`
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"created_at"`
}

type moneyResponse struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

type depositRequest struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

type depositResponse struct {
	ID       string `json:"id"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

type entryResponse struct {
	ID            int64     `json:"id"`
	Direction     string    `json:"direction"`
	Amount        int64     `json:"amount"`
	Currency      string    `json:"currency"`
	TransactionID string    `json:"transaction_id"`
	CreatedAt     time.Time `json:"created_at"`
}

type entriesResponse struct {
	Entries    []entryResponse `json:"entries"`
	NextCursor string          `json:"next_cursor"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) error {
	var req createWalletRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	account, err := h.svc.CreateWallet(r.Context(), req.OwnerID, req.Currency)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, r, http.StatusCreated, walletJSON(account))
	return nil
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) error {
	id, err := parseWalletID(r)
	if err != nil {
		return err
	}
	account, err := h.svc.Wallet(r.Context(), id)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, r, http.StatusOK, walletJSON(account))
	return nil
}

func (h *Handler) balance(w http.ResponseWriter, r *http.Request) error {
	id, err := parseWalletID(r)
	if err != nil {
		return err
	}
	bal, err := h.svc.Balance(r.Context(), id)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, r, http.StatusOK, moneyResponse{Amount: bal.Amount, Currency: bal.Currency})
	return nil
}

func (h *Handler) deposit(w http.ResponseWriter, r *http.Request) error {
	id, err := parseWalletID(r)
	if err != nil {
		return err
	}
	var req depositRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	result, err := h.svc.Deposit(r.Context(), id, req.Amount, req.Currency)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, r, http.StatusCreated, depositResponse{
		ID:       result.ID.String(),
		Amount:   result.Money.Amount,
		Currency: result.Money.Currency,
	})
	return nil
}

func (h *Handler) entries(w http.ResponseWriter, r *http.Request) error {
	id, err := parseWalletID(r)
	if err != nil {
		return err
	}
	limit, err := httpx.QueryInt(r, "limit", DefaultPageSize, 1, MaxPageSize)
	if err != nil {
		return err
	}
	cursor := r.URL.Query().Get("cursor")
	page, err := h.svc.Entries(r.Context(), id, limit, cursor)
	if err != nil {
		return err
	}
	out := entriesResponse{
		Entries:    make([]entryResponse, 0, len(page.Entries)),
		NextCursor: page.NextCursor,
	}
	for _, e := range page.Entries {
		out.Entries = append(out.Entries, entryResponse{
			ID:            e.ID,
			Direction:     string(e.Direction),
			Amount:        e.Amount,
			Currency:      e.Currency,
			TransactionID: e.TransactionID.String(),
			CreatedAt:     e.CreatedAt.UTC(),
		})
	}
	httpx.WriteJSON(w, r, http.StatusOK, out)
	return nil
}

func parseWalletID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return uuid.Nil, ErrWalletNotFound
	}
	return id, nil
}

func walletJSON(a Account) walletResponse {
	ownerID := ""
	if a.OwnerID != nil {
		ownerID = a.OwnerID.String()
	}
	return walletResponse{
		ID:        a.ID.String(),
		OwnerID:   ownerID,
		Currency:  a.Currency,
		CreatedAt: a.CreatedAt.UTC(),
	}
}
