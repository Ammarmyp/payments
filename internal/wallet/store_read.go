package wallet

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// AccountBalance is the Amount implied by an Account's Entries relative to its Normal Balance.
type AccountBalance struct {
	AccountID uuid.UUID
	Currency  string
	Balance   int64
}

// GetLedgerTransaction loads a Ledger Transaction and all of its Entries.
func (s *Store) GetLedgerTransaction(ctx context.Context, id uuid.UUID) (LedgerTransaction, error) {
	row, err := s.q.GetTransaction(ctx, id)
	if err != nil {
		return LedgerTransaction{}, fmt.Errorf("wallet: get ledger transaction: %w", err)
	}

	rows, err := s.q.ListEntriesByTransaction(ctx, id)
	if err != nil {
		return LedgerTransaction{}, fmt.Errorf("wallet: list entries by transaction: %w", err)
	}

	entries, err := entriesFromDB(rows)
	if err != nil {
		return LedgerTransaction{}, err
	}

	return ledgerTransactionFromDB(row, entries)
}

// ListEntriesByAccount returns Entries for an Account ordered by creation time.
func (s *Store) ListEntriesByAccount(ctx context.Context, accountID uuid.UUID) ([]Entry, error) {
	rows, err := s.q.ListEntriesByAccount(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("wallet: list entries by account: %w", err)
	}
	return entriesFromDB(rows)
}

// GetAccountBalance reads the derived Account Balance for an Account.
func (s *Store) GetAccountBalance(ctx context.Context, accountID uuid.UUID) (AccountBalance, error) {
	row, err := s.q.GetAccountBalance(ctx, accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return AccountBalance{}, ErrAccountNotFound
	}
	if err != nil {
		return AccountBalance{}, fmt.Errorf("wallet: get account balance: %w", err)
	}
	return AccountBalance{
		AccountID: row.AccountID,
		Currency:  row.Currency,
		Balance:   row.Balance,
	}, nil
}
