package wallet_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Ammarmyp/payments/internal/platform/postgres"
	"github.com/Ammarmyp/payments/internal/wallet"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func testDatabaseURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping Wallet store integration tests")
	}
	return url
}

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	pool, err := postgres.Open(ctx, postgres.DefaultConfig(testDatabaseURL(t)))
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func testStore(t *testing.T) *wallet.Store {
	t.Helper()
	return wallet.NewStore(testPool(t))
}

func TestStoreCreateOwnedAccount(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	ownerID := uuid.New()
	accountID := uuid.New()

	created, err := store.CreateAccount(ctx, wallet.CreateAccountParams{
		ID:            accountID,
		OwnerID:       &ownerID,
		Name:          "USD wallet",
		Currency:      "USD",
		NormalBalance: wallet.NormalBalanceCredit,
		Kind:          wallet.KindWallet,
	})
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}

	if created.ID != accountID {
		t.Fatalf("ID: got %v want %v", created.ID, accountID)
	}
	if created.OwnerID == nil || *created.OwnerID != ownerID {
		t.Fatalf("OwnerID: got %v want %v", created.OwnerID, ownerID)
	}
	if created.Name != "USD wallet" {
		t.Fatalf("Name: got %q want %q", created.Name, "USD wallet")
	}
	if created.Currency != "USD" {
		t.Fatalf("Currency: got %q want %q", created.Currency, "USD")
	}
	if created.NormalBalance != wallet.NormalBalanceCredit {
		t.Fatalf("NormalBalance: got %q want %q", created.NormalBalance, wallet.NormalBalanceCredit)
	}
	if created.Kind != wallet.KindWallet {
		t.Fatalf("Kind: got %q want %q", created.Kind, wallet.KindWallet)
	}
	if created.CreatedAt.IsZero() {
		t.Fatal("CreatedAt: expected non-zero")
	}

	got, err := store.GetAccount(ctx, accountID)
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if got.ID != created.ID || got.Name != created.Name || got.Kind != created.Kind {
		t.Fatalf("GetAccount mismatch: got %+v want %+v", got, created)
	}
}

func TestStoreCreateSystemAccountNullOwner(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	accountID := uuid.New()
	name := fmt.Sprintf("sys-%s", accountID)

	created, err := store.CreateAccount(ctx, wallet.CreateAccountParams{
		ID:            accountID,
		OwnerID:       nil,
		Name:          name,
		Currency:      "EUR",
		NormalBalance: wallet.NormalBalanceDebit,
		Kind:          wallet.KindSystem,
	})
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	if created.OwnerID != nil {
		t.Fatalf("OwnerID: got %v want nil (System Account)", created.OwnerID)
	}
	if created.Kind != wallet.KindSystem {
		t.Fatalf("Kind: got %q want %q", created.Kind, wallet.KindSystem)
	}
	if created.NormalBalance != wallet.NormalBalanceDebit {
		t.Fatalf("NormalBalance: got %q want %q", created.NormalBalance, wallet.NormalBalanceDebit)
	}

	got, err := store.GetAccount(ctx, accountID)
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if got.OwnerID != nil {
		t.Fatalf("GetAccount OwnerID: got %v want nil", got.OwnerID)
	}
	if got.Kind != wallet.KindSystem {
		t.Fatalf("GetAccount Kind: got %q want %q", got.Kind, wallet.KindSystem)
	}
}

func TestStoreListAccountsByOwner(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	ownerID := uuid.New()
	otherOwner := uuid.New()
	firstID := uuid.New()
	secondID := uuid.New()
	otherID := uuid.New()

	for _, p := range []wallet.CreateAccountParams{
		{ID: firstID, OwnerID: &ownerID, Name: "USD wallet", Currency: "USD", NormalBalance: wallet.NormalBalanceCredit, Kind: wallet.KindWallet},
		{ID: secondID, OwnerID: &ownerID, Name: "ETB wallet", Currency: "ETB", NormalBalance: wallet.NormalBalanceCredit, Kind: wallet.KindWallet},
		{ID: otherID, OwnerID: &otherOwner, Name: "USD wallet", Currency: "USD", NormalBalance: wallet.NormalBalanceCredit, Kind: wallet.KindWallet},
	} {
		if _, err := store.CreateAccount(ctx, p); err != nil {
			t.Fatalf("CreateAccount %s: %v", p.Name, err)
		}
	}

	listed, err := store.ListAccountsByOwner(ctx, ownerID)
	if err != nil {
		t.Fatalf("ListAccountsByOwner: %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("len: got %d want 2", len(listed))
	}

	ids := map[uuid.UUID]bool{}
	for _, a := range listed {
		ids[a.ID] = true
		if a.OwnerID == nil || *a.OwnerID != ownerID {
			t.Fatalf("listed OwnerID: got %v want %v", a.OwnerID, ownerID)
		}
		if a.Kind != wallet.KindWallet {
			t.Fatalf("listed Kind: got %q want %q", a.Kind, wallet.KindWallet)
		}
	}
	if !ids[firstID] || !ids[secondID] {
		t.Fatalf("listed IDs: got %v want %v and %v", ids, firstID, secondID)
	}
	if ids[otherID] {
		t.Fatal("list included Account belonging to another Owner")
	}
}

func TestStoreSystemAccountSettlementSeeded(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	for _, currency := range []string{"USD", "ETB"} {
		got, err := store.SystemAccount(ctx, "settlement", currency)
		if err != nil {
			t.Fatalf("SystemAccount(settlement, %s): %v", currency, err)
		}
		if got.Kind != wallet.KindSystem {
			t.Fatalf("%s Kind: got %q want %q", currency, got.Kind, wallet.KindSystem)
		}
		if got.OwnerID != nil {
			t.Fatalf("%s OwnerID: got %v want nil", currency, got.OwnerID)
		}
		if got.Name != "settlement" {
			t.Fatalf("%s Name: got %q want settlement", currency, got.Name)
		}
		if got.Currency != currency {
			t.Fatalf("Currency: got %q want %q", got.Currency, currency)
		}
		if got.NormalBalance != wallet.NormalBalanceDebit {
			t.Fatalf("%s NormalBalance: got %q want debit", currency, got.NormalBalance)
		}
	}
}

func TestStoreSystemAccountNotFound(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	_, err := store.SystemAccount(ctx, "settlement", "XYZ")
	if !errors.Is(err, wallet.ErrAccountNotFound) {
		t.Fatalf("error: got %v want %v", err, wallet.ErrAccountNotFound)
	}
}

func TestStoreCreateAccountDuplicateWalletReturnsExists(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	ownerID := uuid.New()

	params := wallet.CreateAccountParams{
		ID:            uuid.New(),
		OwnerID:       &ownerID,
		Name:          "USD wallet",
		Currency:      "USD",
		NormalBalance: wallet.NormalBalanceCredit,
		Kind:          wallet.KindWallet,
	}
	if _, err := store.CreateAccount(ctx, params); err != nil {
		t.Fatalf("first CreateAccount: %v", err)
	}

	params.ID = uuid.New()
	_, err := store.CreateAccount(ctx, params)
	if !errors.Is(err, wallet.ErrAccountExists) {
		t.Fatalf("second CreateAccount: got %v want %v", err, wallet.ErrAccountExists)
	}
}

func TestStoreCreateSystemAccountExplicitKind(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	id := uuid.New()
	name := fmt.Sprintf("sys-%s", id)

	created, err := store.CreateAccount(ctx, wallet.CreateAccountParams{
		ID:            id,
		OwnerID:       nil,
		Name:          name,
		Currency:      "USD",
		NormalBalance: wallet.NormalBalanceDebit,
		Kind:          wallet.KindSystem,
	})
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	if created.Kind != wallet.KindSystem || created.OwnerID != nil {
		t.Fatalf("got Kind=%q OwnerID=%v", created.Kind, created.OwnerID)
	}
}

func TestStoreCreateAccountDuplicateSystemReturnsExists(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	name := fmt.Sprintf("sys-dup-%s", uuid.New())

	params := wallet.CreateAccountParams{
		ID:            uuid.New(),
		OwnerID:       nil,
		Name:          name,
		Currency:      "USD",
		NormalBalance: wallet.NormalBalanceDebit,
		Kind:          wallet.KindSystem,
	}
	if _, err := store.CreateAccount(ctx, params); err != nil {
		t.Fatalf("first CreateAccount: %v", err)
	}

	params.ID = uuid.New()
	_, err := store.CreateAccount(ctx, params)
	if !errors.Is(err, wallet.ErrAccountExists) {
		t.Fatalf("second CreateAccount: got %v want %v", err, wallet.ErrAccountExists)
	}
}
