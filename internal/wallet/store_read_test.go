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

	got, err := store.GetLedgerTransaction(ctx, txID)
	if err != nil {
		t.Fatalf("GetLedgerTransaction: %v", err)
	}
	if got.ID != txID {
		t.Fatalf("ID: got %v want %v", got.ID, txID)
	}
	if got.Description != "deposit" {
		t.Fatalf("Description: got %q want %q", got.Description, "deposit")
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
	settlementEntry, ok := byAccount[settlement.ID]
	if !ok {
		t.Fatal("missing settlement Entry")
	}
	if settlementEntry.Direction != wallet.EntryDirectionDebit || settlementEntry.Amount != 100 || settlementEntry.Currency != "USD" {
		t.Fatalf("settlement Entry: %+v", settlementEntry)
	}
	ownedEntry, ok := byAccount[owned.ID]
	if !ok {
		t.Fatal("missing owned Entry")
	}
	if ownedEntry.Direction != wallet.EntryDirectionCredit || ownedEntry.Amount != 100 || ownedEntry.Currency != "USD" {
		t.Fatalf("owned Entry: %+v", ownedEntry)
	}

	if posted.ID != got.ID || len(posted.Entries) != len(got.Entries) {
		t.Fatalf("retrieved Ledger Transaction diverged from posted result")
	}
}

func TestStoreListEntriesByAccountIncludesPostedLines(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	owned, settlement := seedPostingAccounts(t, store)
	txID := uuid.New()

	_, err := store.PostLedgerTransaction(ctx, wallet.PostLedgerTransactionParams{
		ID:          txID,
		Description: "deposit",
		Entries: []wallet.EntryInput{
			{AccountID: settlement.ID, Direction: wallet.EntryDirectionDebit, Amount: 250, Currency: "USD"},
			{AccountID: owned.ID, Direction: wallet.EntryDirectionCredit, Amount: 250, Currency: "USD"},
		},
	})
	if err != nil {
		t.Fatalf("PostLedgerTransaction: %v", err)
	}

	ownedEntries, err := store.ListEntriesByAccount(ctx, owned.ID)
	if err != nil {
		t.Fatalf("ListEntriesByAccount(owned): %v", err)
	}
	if len(ownedEntries) != 1 {
		t.Fatalf("owned Entries len: got %d want 1", len(ownedEntries))
	}
	if ownedEntries[0].TransactionID != txID ||
		ownedEntries[0].AccountID != owned.ID ||
		ownedEntries[0].Direction != wallet.EntryDirectionCredit ||
		ownedEntries[0].Amount != 250 ||
		ownedEntries[0].Currency != "USD" {
		t.Fatalf("owned Entry: %+v", ownedEntries[0])
	}

	settlementEntries, err := store.ListEntriesByAccount(ctx, settlement.ID)
	if err != nil {
		t.Fatalf("ListEntriesByAccount(settlement): %v", err)
	}
	if len(settlementEntries) != 1 {
		t.Fatalf("settlement Entries len: got %d want 1", len(settlementEntries))
	}
	if settlementEntries[0].TransactionID != txID ||
		settlementEntries[0].AccountID != settlement.ID ||
		settlementEntries[0].Direction != wallet.EntryDirectionDebit ||
		settlementEntries[0].Amount != 250 {
		t.Fatalf("settlement Entry: %+v", settlementEntries[0])
	}
}

func TestStoreGetAccountBalanceMatchesNormalBalanceSemantics(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()
	owned, settlement := seedPostingAccounts(t, store)

	zeroOwned, err := store.GetAccountBalance(ctx, owned.ID)
	if err != nil {
		t.Fatalf("GetAccountBalance(owned before post): %v", err)
	}
	if zeroOwned.AccountID != owned.ID || zeroOwned.Currency != "USD" || zeroOwned.Balance != 0 {
		t.Fatalf("owned balance before post: %+v", zeroOwned)
	}
	zeroSettlement, err := store.GetAccountBalance(ctx, settlement.ID)
	if err != nil {
		t.Fatalf("GetAccountBalance(settlement before post): %v", err)
	}
	if zeroSettlement.Balance != 0 {
		t.Fatalf("settlement balance before post: %+v", zeroSettlement)
	}

	_, err = store.PostLedgerTransaction(ctx, wallet.PostLedgerTransactionParams{
		ID:          uuid.New(),
		Description: "deposit",
		Entries: []wallet.EntryInput{
			{AccountID: settlement.ID, Direction: wallet.EntryDirectionDebit, Amount: 100, Currency: "USD"},
			{AccountID: owned.ID, Direction: wallet.EntryDirectionCredit, Amount: 100, Currency: "USD"},
		},
	})
	if err != nil {
		t.Fatalf("PostLedgerTransaction: %v", err)
	}

	// Debit-normal Settlement: debit increases balance.
	settlementBal, err := store.GetAccountBalance(ctx, settlement.ID)
	if err != nil {
		t.Fatalf("GetAccountBalance(settlement): %v", err)
	}
	if settlementBal.Balance != 100 {
		t.Fatalf("settlement debit-normal balance: got %d want 100", settlementBal.Balance)
	}

	// Credit-normal owned Account: credit increases balance.
	ownedBal, err := store.GetAccountBalance(ctx, owned.ID)
	if err != nil {
		t.Fatalf("GetAccountBalance(owned): %v", err)
	}
	if ownedBal.Balance != 100 {
		t.Fatalf("owned credit-normal balance: got %d want 100", ownedBal.Balance)
	}

	_, err = store.PostLedgerTransaction(ctx, wallet.PostLedgerTransactionParams{
		ID:          uuid.New(),
		Description: "withdraw",
		Entries: []wallet.EntryInput{
			{AccountID: owned.ID, Direction: wallet.EntryDirectionDebit, Amount: 40, Currency: "USD"},
			{AccountID: settlement.ID, Direction: wallet.EntryDirectionCredit, Amount: 40, Currency: "USD"},
		},
	})
	if err != nil {
		t.Fatalf("PostLedgerTransaction(withdraw): %v", err)
	}

	ownedBal, err = store.GetAccountBalance(ctx, owned.ID)
	if err != nil {
		t.Fatalf("GetAccountBalance(owned after withdraw): %v", err)
	}
	if ownedBal.Balance != 60 {
		t.Fatalf("owned after opposing Entry: got %d want 60", ownedBal.Balance)
	}

	settlementBal, err = store.GetAccountBalance(ctx, settlement.ID)
	if err != nil {
		t.Fatalf("GetAccountBalance(settlement after withdraw): %v", err)
	}
	if settlementBal.Balance != 60 {
		t.Fatalf("settlement after opposing Entry: got %d want 60", settlementBal.Balance)
	}
}
