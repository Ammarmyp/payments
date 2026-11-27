package wallet

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/Ammarmyp/payments/internal/platform/apperr"
	"github.com/Ammarmyp/payments/internal/wallet/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NormalBalance is whether an Account increases on debit or on credit.
type NormalBalance string

const (
	NormalBalanceDebit  NormalBalance = "debit"
	NormalBalanceCredit NormalBalance = "credit"
)

// AccountKind is whether an Account is wallet (owned) or system (platform).
type AccountKind string

const (
	KindWallet AccountKind = "wallet"
	KindSystem AccountKind = "system"
)

// ErrAccountNotFound means no account exists for the requested id.
var ErrAccountNotFound = fmt.Errorf("account: %w", apperr.ErrNotFound)

// ErrAccountExists means an Account uniqueness constraint rejected the insert.
var ErrAccountExists = fmt.Errorf("wallet: account already exists: %w", apperr.ErrConflict)

var currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

// Account is a named balance container in a single Currency.
type Account struct {
	ID            uuid.UUID
	OwnerID       *uuid.UUID
	Name          string
	Currency      string
	NormalBalance NormalBalance
	Kind          AccountKind
	CreatedAt     time.Time
}

// CreateAccountParams creates an Owner-linked wallet Account or a System Account.
// ID must be application-generated (non-nil) so callers know the identifier before commit.
type CreateAccountParams struct {
	ID            uuid.UUID
	OwnerID       *uuid.UUID
	Name          string
	Currency      string
	NormalBalance NormalBalance
	Kind          AccountKind
}

// Store persists Wallet ledger data through sqlc-generated queries.
type Store struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

// NewStore returns a Store backed by pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, q: db.New(pool)}
}

// CreateAccount inserts an Account. Kind and OwnerID must agree (wallet has Owner; system has none).
func (s *Store) CreateAccount(ctx context.Context, params CreateAccountParams) (Account, error) {
	if err := validateCreateAccount(params); err != nil {
		return Account{}, err
	}

	row, err := s.q.CreateAccount(ctx, db.CreateAccountParams{
		ID:            params.ID,
		OwnerID:       params.OwnerID,
		Name:          params.Name,
		Currency:      params.Currency,
		NormalBalance: db.EntryDirection(params.NormalBalance),
		Kind:          db.AccountKind(params.Kind),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return Account{}, fmt.Errorf("%w: %v", ErrAccountExists, err)
		}
		return Account{}, fmt.Errorf("wallet: create account: %w", err)
	}
	return accountFromDB(row)
}

// GetAccount loads an Account by ID.
func (s *Store) GetAccount(ctx context.Context, id uuid.UUID) (Account, error) {
	row, err := s.q.GetAccount(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrAccountNotFound
	}
	if err != nil {
		return Account{}, fmt.Errorf("wallet: get account: %w", err)
	}
	return accountFromDB(row)
}

// SystemAccount returns a System Account by name and Currency (for example Settlement).
func (s *Store) SystemAccount(ctx context.Context, name, currency string) (Account, error) {
	if name == "" {
		return Account{}, fmt.Errorf("wallet: system account name must be non-empty")
	}
	if !currencyPattern.MatchString(currency) {
		return Account{}, fmt.Errorf("wallet: currency must be ISO 4217 alphabetic code")
	}
	row, err := s.q.GetSystemAccount(ctx, db.GetSystemAccountParams{
		Name:     name,
		Currency: currency,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrAccountNotFound
	}
	if err != nil {
		return Account{}, fmt.Errorf("wallet: get system account: %w", err)
	}
	return accountFromDB(row)
}

// ListAccountsByOwner returns Accounts owned by ownerID (never System Accounts).
func (s *Store) ListAccountsByOwner(ctx context.Context, ownerID uuid.UUID) ([]Account, error) {
	rows, err := s.q.ListAccountsByOwner(ctx, &ownerID)
	if err != nil {
		return nil, fmt.Errorf("wallet: list accounts by owner: %w", err)
	}
	out := make([]Account, 0, len(rows))
	for _, row := range rows {
		a, err := accountFromDB(row)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

func validateCreateAccount(params CreateAccountParams) error {
	if params.ID == uuid.Nil {
		return fmt.Errorf("wallet: account id must be application-generated")
	}
	if params.Name == "" {
		return fmt.Errorf("wallet: account name must be non-empty")
	}
	if !currencyPattern.MatchString(params.Currency) {
		return fmt.Errorf("wallet: currency must be ISO 4217 alphabetic code")
	}
	switch params.NormalBalance {
	case NormalBalanceDebit, NormalBalanceCredit:
	default:
		return fmt.Errorf("wallet: normal balance must be debit or credit")
	}
	switch params.Kind {
	case KindWallet:
		if params.OwnerID == nil {
			return fmt.Errorf("wallet: wallet account requires an owner")
		}
	case KindSystem:
		if params.OwnerID != nil {
			return fmt.Errorf("wallet: system account must not have an owner")
		}
	default:
		return fmt.Errorf("wallet: account kind must be wallet or system")
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func accountFromDB(row db.Account) (Account, error) {
	if !row.CreatedAt.Valid {
		return Account{}, fmt.Errorf("wallet: account %s missing created_at", row.ID)
	}
	return Account{
		ID:            row.ID,
		OwnerID:       row.OwnerID,
		Name:          row.Name,
		Currency:      row.Currency,
		NormalBalance: NormalBalance(row.NormalBalance),
		Kind:          AccountKind(row.Kind),
		CreatedAt:     row.CreatedAt.Time,
	}, nil
}
