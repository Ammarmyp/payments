package wallet_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/Ammarmyp/payments/internal/wallet"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// seedPostingAccounts returns an owned credit-normal Account and a debit-normal
// System Account (ADR-0002). A deposit-shaped post debits settlement and credits owned.
func seedPostingAccounts(t *testing.T, store *wallet.Store) (owned, settlement wallet.Account) {
	t.Helper()
	ctx := context.Background()
	owner := uuid.New()

	owned, err := store.CreateAccount(ctx, wallet.CreateAccountParams{
		ID:            uuid.New(),
		OwnerID:       &owner,
		Name:          "USD wallet",
		Currency:      "USD",
		NormalBalance: wallet.NormalBalanceCredit,
		Kind:          wallet.KindWallet,
	})
	if err != nil {
		t.Fatalf("create owned Account: %v", err)
	}
	settlementID := uuid.New()
	settlement, err = store.CreateAccount(ctx, wallet.CreateAccountParams{
		ID:            settlementID,
		OwnerID:       nil,
		Name:          fmt.Sprintf("sys-%s", settlementID),
		Currency:      "USD",
		NormalBalance: wallet.NormalBalanceDebit,
		Kind:          wallet.KindSystem,
	})
	if err != nil {
		t.Fatalf("create settlement Account: %v", err)
	}
	return owned, settlement
}

func countLedgerRows(t *testing.T, pool *pgxpool.Pool, txID uuid.UUID) (txs, entries int64) {
	t.Helper()
	ctx := context.Background()
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM transactions WHERE id = $1`, txID).Scan(&txs); err != nil {
		t.Fatalf("count transactions: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM entries WHERE transaction_id = $1`, txID).Scan(&entries); err != nil {
		t.Fatalf("count entries: %v", err)
	}
	return txs, entries
}

func TestStorePostBalancedLedgerTransaction(t *testing.T) {
	pool := testPool(t)
	store := wallet.NewStore(pool)
	ctx := context.Background()
	owned, settlement := seedPostingAccounts(t, store)
	txID := uuid.New()

	posted, err := store.PostLedgerTransaction(ctx, wallet.PostLedgerTransactionParams{
		ID:          txID,
		Description: "deposit",
		Entries: []wallet.EntryInput{
			{AccountID: settlement.ID, Direction: wallet.EntryDirectionDebit, Amount: 100, Currency: "USD"},
			{AccountID: owned.ID, Direction: wallet.EntryDirectionCredit, Amount: 100, Currency: "USD"},
		},
	})
	if err != nil {
		t.Fatalf("PostLedgerTransaction: %v", err)
	}
	if posted.ID != txID {
		t.Fatalf("ID: got %v want %v", posted.ID, txID)
	}
	if posted.Description != "deposit" {
		t.Fatalf("Description: got %q", posted.Description)
	}
	if len(posted.Entries) != 2 {
		t.Fatalf("Entries len: got %d want 2", len(posted.Entries))
	}
	if posted.CreatedAt.IsZero() {
		t.Fatal("CreatedAt: expected non-zero")
	}

	txs, entries := countLedgerRows(t, pool, txID)
	if txs != 1 || entries != 2 {
		t.Fatalf("persisted rows: transactions=%d entries=%d", txs, entries)
	}
}

func TestStorePostRejectsUnbalancedLedgerTransaction(t *testing.T) {
	pool := testPool(t)
	store := wallet.NewStore(pool)
	ctx := context.Background()
	owned, settlement := seedPostingAccounts(t, store)
	txID := uuid.New()

	_, err := store.PostLedgerTransaction(ctx, wallet.PostLedgerTransactionParams{
		ID:          txID,
		Description: "unbalanced",
		Entries: []wallet.EntryInput{
			{AccountID: settlement.ID, Direction: wallet.EntryDirectionDebit, Amount: 100, Currency: "USD"},
			{AccountID: owned.ID, Direction: wallet.EntryDirectionCredit, Amount: 50, Currency: "USD"},
		},
	})
	if !errors.Is(err, wallet.ErrUnbalancedLedgerTransaction) {
		t.Fatalf("error: got %v want %v", err, wallet.ErrUnbalancedLedgerTransaction)
	}

	txs, entries := countLedgerRows(t, pool, txID)
	if txs != 0 || entries != 0 {
		t.Fatalf("expected no leftover rows, got transactions=%d entries=%d", txs, entries)
	}
}

func TestStorePostRejectsEmptyLedgerTransaction(t *testing.T) {
	pool := testPool(t)
	store := wallet.NewStore(pool)
	ctx := context.Background()
	txID := uuid.New()

	_, err := store.PostLedgerTransaction(ctx, wallet.PostLedgerTransactionParams{
		ID:          txID,
		Description: "empty",
		Entries:     nil,
	})
	if !errors.Is(err, wallet.ErrEmptyLedgerTransaction) {
		t.Fatalf("error: got %v want %v", err, wallet.ErrEmptyLedgerTransaction)
	}

	txs, entries := countLedgerRows(t, pool, txID)
	if txs != 0 || entries != 0 {
		t.Fatalf("expected no leftover rows, got transactions=%d entries=%d", txs, entries)
	}
}

func TestStorePostRollsBackOnMidPostFailure(t *testing.T) {
	pool := testPool(t)
	store := wallet.NewStore(pool)
	ctx := context.Background()
	owned, settlement := seedPostingAccounts(t, store)
	txID := uuid.New()
	missingAccount := uuid.New()

	_, err := store.PostLedgerTransaction(ctx, wallet.PostLedgerTransactionParams{
		ID:          txID,
		Description: "fk failure",
		Entries: []wallet.EntryInput{
			{AccountID: settlement.ID, Direction: wallet.EntryDirectionDebit, Amount: 25, Currency: "USD"},
			{AccountID: owned.ID, Direction: wallet.EntryDirectionCredit, Amount: 25, Currency: "USD"},
			// Balanced overall, but unknown Account triggers FK failure after earlier inserts.
			{AccountID: missingAccount, Direction: wallet.EntryDirectionDebit, Amount: 10, Currency: "USD"},
			{AccountID: missingAccount, Direction: wallet.EntryDirectionCredit, Amount: 10, Currency: "USD"},
		},
	})
	if !errors.Is(err, wallet.ErrLedgerInvariant) {
		t.Fatalf("error: got %v want %v", err, wallet.ErrLedgerInvariant)
	}

	txs, entries := countLedgerRows(t, pool, txID)
	if txs != 0 || entries != 0 {
		t.Fatalf("expected full rollback, got transactions=%d entries=%d", txs, entries)
	}
}

func TestStorePostConcurrentBalancedLedgerTransactions(t *testing.T) {
	pool := testPool(t)
	store := wallet.NewStore(pool)
	ctx := context.Background()
	owned, settlement := seedPostingAccounts(t, store)

	const workers = 8
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	ids := make([]uuid.UUID, workers)

	for i := 0; i < workers; i++ {
		ids[i] = uuid.New()
		wg.Add(1)
		go func(txID uuid.UUID) {
			defer wg.Done()
			_, err := store.PostLedgerTransaction(ctx, wallet.PostLedgerTransactionParams{
				ID:          txID,
				Description: "concurrent",
				Entries: []wallet.EntryInput{
					{AccountID: settlement.ID, Direction: wallet.EntryDirectionDebit, Amount: 1, Currency: "USD"},
					{AccountID: owned.ID, Direction: wallet.EntryDirectionCredit, Amount: 1, Currency: "USD"},
				},
			})
			errs <- err
		}(ids[i])
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent post: %v", err)
		}
	}
	for _, id := range ids {
		txs, entries := countLedgerRows(t, pool, id)
		if txs != 1 || entries != 2 {
			t.Fatalf("tx %s: transactions=%d entries=%d", id, txs, entries)
		}
	}

	var ownedBal, settlementBal int64
	if err := pool.QueryRow(ctx, `SELECT balance FROM account_balances WHERE account_id = $1`, owned.ID).Scan(&ownedBal); err != nil {
		t.Fatalf("owned balance: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT balance FROM account_balances WHERE account_id = $1`, settlement.ID).Scan(&settlementBal); err != nil {
		t.Fatalf("settlement balance: %v", err)
	}
	if ownedBal != workers || settlementBal != workers {
		t.Fatalf("balances: owned=%d settlement=%d want %d each", ownedBal, settlementBal, workers)
	}
}
