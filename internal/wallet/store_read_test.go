package wallet_test

import (
	"context"
	"testing"

	"github.com/Ammarmyp/payments/internal/wallet"
	"github.com/google/uuid"
)

func TestStoreGetLedgerTransactionIncludesEntries(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	debit, credit := seedPostingAccounts(t, store)
	txID := uuid.New()

	posted, err := store.PostLedgerTransaction(ctx, wallet.PostLedgerTransactionParams{
		ID:          txID,
		Description: "fund cash",
		Entries: []wallet.EntryInput{
			{AccountID: debit.ID, Direction: wallet.EntryDirectionDebit, Amount: 100, Currency: "USD"},
			{AccountID: credit.ID, Direction: wallet.EntryDirectionCredit, Amount: 100, Currency: "USD"},
		},
	})
	if err != nil {
		t.Fatalf("PostLedgerTransaction: %v", err)
	}

	got, err := store.GetLedgerTransaction(ctx, txID)
	if err != nil {
		t.Fatalf("GetLedgerTransaction: %v", err)
	}
	if got.ID != txID {
		t.Fatalf("ID: got %v want %v", got.ID, txID)
	}
	if got.Description != "fund cash" {
		t.Fatalf("Description: got %q want %q", got.Description, "fund cash")
	}
	if got.CreatedAt.IsZero() {
		t.Fatal("CreatedAt: expected non-zero")
	}
	if len(got.Entries) != 2 {
		t.Fatalf("Entries len: got %d want 2", len(got.Entries))
	}

	byAccount := map[uuid.UUID]wallet.Entry{}
	for _, e := range got.Entries {
		if e.TransactionID != txID {
			t.Fatalf("Entry.TransactionID: got %v want %v", e.TransactionID, txID)
		}
		byAccount[e.AccountID] = e
	}
	debitEntry, ok := byAccount[debit.ID]
	if !ok {
		t.Fatal("missing debit Entry")
	}
	if debitEntry.Direction != wallet.EntryDirectionDebit || debitEntry.Amount != 100 || debitEntry.Currency != "USD" {
		t.Fatalf("debit Entry: %+v", debitEntry)
	}
	creditEntry, ok := byAccount[credit.ID]
	if !ok {
		t.Fatal("missing credit Entry")
	}
	if creditEntry.Direction != wallet.EntryDirectionCredit || creditEntry.Amount != 100 || creditEntry.Currency != "USD" {
		t.Fatalf("credit Entry: %+v", creditEntry)
	}

	if posted.ID != got.ID || len(posted.Entries) != len(got.Entries) {
		t.Fatalf("retrieved Ledger Transaction diverged from posted result")
	}
}

func TestStoreListEntriesByAccountIncludesPostedLines(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	debit, credit := seedPostingAccounts(t, store)
	txID := uuid.New()

	_, err := store.PostLedgerTransaction(ctx, wallet.PostLedgerTransactionParams{
		ID:          txID,
		Description: "fund cash",
		Entries: []wallet.EntryInput{
			{AccountID: debit.ID, Direction: wallet.EntryDirectionDebit, Amount: 250, Currency: "USD"},
			{AccountID: credit.ID, Direction: wallet.EntryDirectionCredit, Amount: 250, Currency: "USD"},
		},
	})
	if err != nil {
		t.Fatalf("PostLedgerTransaction: %v", err)
	}

	debitEntries, err := store.ListEntriesByAccount(ctx, debit.ID)
	if err != nil {
		t.Fatalf("ListEntriesByAccount(debit): %v", err)
	}
	if len(debitEntries) != 1 {
		t.Fatalf("debit Entries len: got %d want 1", len(debitEntries))
	}
	if debitEntries[0].TransactionID != txID ||
		debitEntries[0].AccountID != debit.ID ||
		debitEntries[0].Direction != wallet.EntryDirectionDebit ||
		debitEntries[0].Amount != 250 ||
		debitEntries[0].Currency != "USD" {
		t.Fatalf("debit Entry: %+v", debitEntries[0])
	}

	creditEntries, err := store.ListEntriesByAccount(ctx, credit.ID)
	if err != nil {
		t.Fatalf("ListEntriesByAccount(credit): %v", err)
	}
	if len(creditEntries) != 1 {
		t.Fatalf("credit Entries len: got %d want 1", len(creditEntries))
	}
	if creditEntries[0].TransactionID != txID ||
		creditEntries[0].AccountID != credit.ID ||
		creditEntries[0].Direction != wallet.EntryDirectionCredit ||
		creditEntries[0].Amount != 250 {
		t.Fatalf("credit Entry: %+v", creditEntries[0])
	}
}

func TestStoreGetAccountBalanceMatchesNormalBalanceSemantics(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	debit, credit := seedPostingAccounts(t, store)

	zeroDebit, err := store.GetAccountBalance(ctx, debit.ID)
	if err != nil {
		t.Fatalf("GetAccountBalance(debit before post): %v", err)
	}
	if zeroDebit.AccountID != debit.ID || zeroDebit.Currency != "USD" || zeroDebit.Balance != 0 {
		t.Fatalf("debit balance before post: %+v", zeroDebit)
	}
	zeroCredit, err := store.GetAccountBalance(ctx, credit.ID)
	if err != nil {
		t.Fatalf("GetAccountBalance(credit before post): %v", err)
	}
	if zeroCredit.Balance != 0 {
		t.Fatalf("credit balance before post: %+v", zeroCredit)
	}

	_, err = store.PostLedgerTransaction(ctx, wallet.PostLedgerTransactionParams{
		ID:          uuid.New(),
		Description: "fund cash",
		Entries: []wallet.EntryInput{
			{AccountID: debit.ID, Direction: wallet.EntryDirectionDebit, Amount: 100, Currency: "USD"},
			{AccountID: credit.ID, Direction: wallet.EntryDirectionCredit, Amount: 100, Currency: "USD"},
		},
	})
	if err != nil {
		t.Fatalf("PostLedgerTransaction: %v", err)
	}

	// Debit-normal Account: debit increases balance.
	debitBal, err := store.GetAccountBalance(ctx, debit.ID)
	if err != nil {
		t.Fatalf("GetAccountBalance(debit): %v", err)
	}
	if debitBal.Balance != 100 {
		t.Fatalf("debit-normal balance: got %d want 100", debitBal.Balance)
	}

	// Credit-normal Account: credit increases balance.
	creditBal, err := store.GetAccountBalance(ctx, credit.ID)
	if err != nil {
		t.Fatalf("GetAccountBalance(credit): %v", err)
	}
	if creditBal.Balance != 100 {
		t.Fatalf("credit-normal balance: got %d want 100", creditBal.Balance)
	}

	_, err = store.PostLedgerTransaction(ctx, wallet.PostLedgerTransactionParams{
		ID:          uuid.New(),
		Description: "spend cash",
		Entries: []wallet.EntryInput{
			{AccountID: debit.ID, Direction: wallet.EntryDirectionCredit, Amount: 40, Currency: "USD"},
			{AccountID: credit.ID, Direction: wallet.EntryDirectionDebit, Amount: 40, Currency: "USD"},
		},
	})
	if err != nil {
		t.Fatalf("PostLedgerTransaction(spend): %v", err)
	}

	debitBal, err = store.GetAccountBalance(ctx, debit.ID)
	if err != nil {
		t.Fatalf("GetAccountBalance(debit after spend): %v", err)
	}
	if debitBal.Balance != 60 {
		t.Fatalf("debit-normal after opposing Entry: got %d want 60", debitBal.Balance)
	}

	creditBal, err = store.GetAccountBalance(ctx, credit.ID)
	if err != nil {
		t.Fatalf("GetAccountBalance(credit after spend): %v", err)
	}
	if creditBal.Balance != 60 {
		t.Fatalf("credit-normal after opposing Entry: got %d want 60", creditBal.Balance)
	}
}
