package wallet

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Ammarmyp/payments/internal/platform/postgres"
	"github.com/Ammarmyp/payments/internal/wallet/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// EntryDirection is debit or credit on an Entry.
type EntryDirection string

const (
	EntryDirectionDebit  EntryDirection = "debit"
	EntryDirectionCredit EntryDirection = "credit"
)

// EntryInput is one line of a Ledger Transaction to post.
type EntryInput struct {
	AccountID uuid.UUID
	Direction EntryDirection
	Amount    int64
	Currency  string
}

// PostLedgerTransactionParams posts one atomic Ledger Transaction with its Entries.
// ID must be application-generated (non-nil).
type PostLedgerTransactionParams struct {
	ID          uuid.UUID
	Description string
	Entries     []EntryInput
}

// Entry is one persisted line of a Ledger Transaction.
type Entry struct {
	ID            int64
	TransactionID uuid.UUID
	AccountID     uuid.UUID
	Direction     EntryDirection
	Amount        int64
	Currency      string
	CreatedAt     time.Time
}

// LedgerTransaction is a persisted movement of value and its Entries.
type LedgerTransaction struct {
	ID          uuid.UUID
	Description string
	CreatedAt   time.Time
	Entries     []Entry
}

// PostLedgerTransaction inserts a Ledger Transaction and its Entries atomically.
func (s *Store) PostLedgerTransaction(ctx context.Context, params PostLedgerTransactionParams) (LedgerTransaction, error) {
	if err := validatePost(params); err != nil {
		return LedgerTransaction{}, err
	}

	var posted LedgerTransaction
	err := postgres.InTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		q := s.q.WithTx(tx)

		row, err := q.CreateTransaction(ctx, db.CreateTransactionParams{
			ID:          params.ID,
			Description: params.Description,
		})
		if err != nil {
			return fmt.Errorf("wallet: create ledger transaction: %w", err)
		}

		entries := make([]Entry, 0, len(params.Entries))
		for _, in := range params.Entries {
			erow, err := q.CreateEntry(ctx, db.CreateEntryParams{
				TransactionID: params.ID,
				AccountID:     in.AccountID,
				Direction:     db.EntryDirection(in.Direction),
				Amount:        in.Amount,
				Currency:      in.Currency,
			})
			if err != nil {
				return fmt.Errorf("wallet: create entry: %w", err)
			}
			entry, err := entryFromDB(erow)
			if err != nil {
				return err
			}
			entries = append(entries, entry)
		}

		posted, err = ledgerTransactionFromDB(row, entries)
		return err
	})
	if err != nil {
		return LedgerTransaction{}, mapPostError(err)
	}
	return posted, nil
}

func validatePost(params PostLedgerTransactionParams) error {
	if params.ID == uuid.Nil {
		return fmt.Errorf("wallet: ledger transaction id must be application-generated")
	}
	if len(params.Entries) == 0 {
		return ErrEmptyLedgerTransaction
	}

	type totals struct {
		debit  int64
		credit int64
	}
	byCurrency := map[string]*totals{}

	for _, e := range params.Entries {
		if e.AccountID == uuid.Nil {
			return fmt.Errorf("wallet: entry account id is required")
		}
		if e.Amount <= 0 {
			return fmt.Errorf("wallet: entry amount must be a positive minor-unit integer")
		}
		if !currencyPattern.MatchString(e.Currency) {
			return fmt.Errorf("wallet: entry currency must be ISO 4217 alphabetic code")
		}
		switch e.Direction {
		case EntryDirectionDebit, EntryDirectionCredit:
		default:
			return fmt.Errorf("wallet: entry direction must be debit or credit")
		}

		t := byCurrency[e.Currency]
		if t == nil {
			t = &totals{}
			byCurrency[e.Currency] = t
		}
		switch e.Direction {
		case EntryDirectionDebit:
			t.debit += e.Amount
		case EntryDirectionCredit:
			t.credit += e.Amount
		}
	}

	for _, t := range byCurrency {
		if t.debit != t.credit {
			return ErrUnbalancedLedgerTransaction
		}
	}
	return nil
}

func mapPostError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23514": // check_violation from deferred empty/balance triggers
			if strings.Contains(pgErr.Message, "has no entries") {
				return fmt.Errorf("%w: %v", ErrEmptyLedgerTransaction, err)
			}
			return fmt.Errorf("%w: %v", ErrUnbalancedLedgerTransaction, err)
		case "23503": // foreign_key_violation
			return fmt.Errorf("%w: %v", ErrLedgerInvariant, err)
		}
	}
	return err
}

func ledgerTransactionFromDB(row db.Transaction, entries []Entry) (LedgerTransaction, error) {
	if !row.CreatedAt.Valid {
		return LedgerTransaction{}, fmt.Errorf("wallet: ledger transaction %s missing created_at", row.ID)
	}
	return LedgerTransaction{
		ID:          row.ID,
		Description: row.Description,
		CreatedAt:   row.CreatedAt.Time,
		Entries:     entries,
	}, nil
}

func entryFromDB(row db.Entry) (Entry, error) {
	if !row.CreatedAt.Valid {
		return Entry{}, fmt.Errorf("wallet: entry %d missing created_at", row.ID)
	}
	return Entry{
		ID:            row.ID,
		TransactionID: row.TransactionID,
		AccountID:     row.AccountID,
		Direction:     EntryDirection(row.Direction),
		Amount:        row.Amount,
		Currency:      row.Currency,
		CreatedAt:     row.CreatedAt.Time,
	}, nil
}

func entriesFromDB(rows []db.Entry) ([]Entry, error) {
	out := make([]Entry, 0, len(rows))
	for _, row := range rows {
		entry, err := entryFromDB(row)
		if err != nil {
			return nil, err
		}
		out = append(out, entry)
	}
	return out, nil
}
