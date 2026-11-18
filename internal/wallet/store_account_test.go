package wallet_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Ammarmyp/payments/internal/platform/postgres"
	"github.com/Ammarmyp/payments/internal/wallet"
	"github.com/google/uuid"
)

func testStore(t *testing.T) *wallet.Store {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping Wallet store integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	pool, err := postgres.Open(ctx, postgres.DefaultConfig(url))
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(pool.Close)

	return wallet.NewStore(pool)
}

func TestStoreCreateOwnedAccount(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	ownerID := uuid.New()
	accountID := uuid.New()

	created, err := store.CreateAccount(ctx, wallet.CreateAccountParams{
		ID:            accountID,
		OwnerID:       &ownerID,
		Name:          "user-cash",
		Currency:      "USD",
		NormalBalance: wallet.NormalBalanceDebit,
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
	if created.Name != "user-cash" {
		t.Fatalf("Name: got %q want %q", created.Name, "user-cash")
	}
	if created.Currency != "USD" {
		t.Fatalf("Currency: got %q want %q", created.Currency, "USD")
	}
	if created.NormalBalance != wallet.NormalBalanceDebit {
		t.Fatalf("NormalBalance: got %q want %q", created.NormalBalance, wallet.NormalBalanceDebit)
	}
	if created.CreatedAt.IsZero() {
		t.Fatal("CreatedAt: expected non-zero")
	}

	got, err := store.GetAccount(ctx, accountID)
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if got.ID != created.ID || got.Name != created.Name {
		t.Fatalf("GetAccount mismatch: got %+v want %+v", got, created)
	}
}

func TestStoreCreateSystemAccountNullOwner(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	accountID := uuid.New()

	created, err := store.CreateAccount(ctx, wallet.CreateAccountParams{
		ID:            accountID,
		OwnerID:       nil,
		Name:          "platform-clearing",
		Currency:      "EUR",
		NormalBalance: wallet.NormalBalanceCredit,
	})
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	if created.OwnerID != nil {
		t.Fatalf("OwnerID: got %v want nil (System Account)", created.OwnerID)
	}
	if created.NormalBalance != wallet.NormalBalanceCredit {
		t.Fatalf("NormalBalance: got %q want %q", created.NormalBalance, wallet.NormalBalanceCredit)
	}

	got, err := store.GetAccount(ctx, accountID)
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if got.OwnerID != nil {
		t.Fatalf("GetAccount OwnerID: got %v want nil", got.OwnerID)
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
		{ID: firstID, OwnerID: &ownerID, Name: "a", Currency: "USD", NormalBalance: wallet.NormalBalanceDebit},
		{ID: secondID, OwnerID: &ownerID, Name: "b", Currency: "USD", NormalBalance: wallet.NormalBalanceCredit},
		{ID: otherID, OwnerID: &otherOwner, Name: "other", Currency: "USD", NormalBalance: wallet.NormalBalanceDebit},
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
	}
	if !ids[firstID] || !ids[secondID] {
		t.Fatalf("listed IDs: got %v want %v and %v", ids, firstID, secondID)
	}
	if ids[otherID] {
		t.Fatal("list included Account belonging to another Owner")
	}
}
