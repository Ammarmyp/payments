package wallet

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"

	"github.com/Ammarmyp/payments/internal/platform/apperr"
	"github.com/google/uuid"
)

// SettlementAccount is the System Account name that funds deposits for a Currency.
const SettlementAccount = "settlement"

// MaxAmount is the largest single deposit in minor units.
const MaxAmount int64 = 100_000_000

// DefaultPageSize is the Entries page size when limit is omitted.
const DefaultPageSize = 20

// MaxPageSize is the largest Entries page size a client may request.
const MaxPageSize = 100

// ErrWalletNotFound means the requested owned Account does not exist (or is not visible).
var ErrWalletNotFound = fmt.Errorf("wallet: %w", apperr.ErrNotFound)

// Money is an API-facing Amount in minor units with its Currency.
type Money struct {
	Amount   int64
	Currency string
}

// Service applies Wallet product rules on top of Store.
type Service struct {
	store *Store
}

// NewService returns a Service backed by store.
func NewService(store *Store) *Service {
	return &Service{store: store}
}

// CreateWallet opens an owned Account for ownerID in currency.
func (s *Service) CreateWallet(ctx context.Context, ownerID, currency string) (Account, error) {
	var v apperr.ValidationError
	parsedOwner, err := uuid.Parse(ownerID)
	if err != nil {
		v.Add("owner_id", "must be a UUID")
	}
	if !currencyPattern.MatchString(currency) {
		v.Add("currency", "must be ISO 4217 alphabetic code")
	}
	if err := v.Err(); err != nil {
		return Account{}, err
	}

	if _, err := s.store.SystemAccount(ctx, SettlementAccount, currency); err != nil {
		if errors.Is(err, ErrAccountNotFound) {
			v.Add("currency", "unsupported currency")
			return Account{}, v.Err()
		}
		return Account{}, fmt.Errorf("wallet: lookup settlement: %w", err)
	}

	account, err := s.store.CreateAccount(ctx, CreateAccountParams{
		ID:            uuid.New(),
		OwnerID:       &parsedOwner,
		Name:          currency + " wallet",
		Currency:      currency,
		NormalBalance: NormalBalanceCredit,
		Kind:          KindWallet,
	})
	if err != nil {
		if errors.Is(err, ErrAccountExists) {
			return Account{}, err
		}
		return Account{}, fmt.Errorf("wallet: create wallet: %w", err)
	}
	return account, nil
}

// Wallet returns an owned Account by id.
func (s *Service) Wallet(ctx context.Context, id uuid.UUID) (Account, error) {
	account, err := s.ownedAccount(ctx, id)
	if err != nil {
		return Account{}, err
	}
	return account, nil
}

// Balance returns the Account Balance for an owned Account.
func (s *Service) Balance(ctx context.Context, id uuid.UUID) (Money, error) {
	if _, err := s.ownedAccount(ctx, id); err != nil {
		return Money{}, err
	}
	bal, err := s.store.GetAccountBalance(ctx, id)
	if err != nil {
		if errors.Is(err, ErrAccountNotFound) {
			return Money{}, ErrWalletNotFound
		}
		return Money{}, fmt.Errorf("wallet: balance: %w", err)
	}
	return Money{Amount: bal.Balance, Currency: bal.Currency}, nil
}

// DepositResult is a posted deposit: Ledger Transaction id and deposited Money.
type DepositResult struct {
	ID    uuid.UUID
	Money Money
}

// Deposit adds external funds to an owned Account via Settlement.
func (s *Service) Deposit(ctx context.Context, id uuid.UUID, amount int64, currency string) (DepositResult, error) {
	account, err := s.ownedAccount(ctx, id)
	if err != nil {
		return DepositResult{}, err
	}

	var v apperr.ValidationError
	if amount <= 0 {
		v.Add("amount", "must be greater than zero")
	} else if amount > MaxAmount {
		v.Add("amount", "exceeds maximum")
	}
	if currency != account.Currency {
		v.Add("currency", "must match wallet currency")
	}
	if err := v.Err(); err != nil {
		return DepositResult{}, err
	}

	settlement, err := s.store.SystemAccount(ctx, SettlementAccount, account.Currency)
	if err != nil {
		if errors.Is(err, ErrAccountNotFound) {
			return DepositResult{}, fmt.Errorf("wallet: settlement missing for %s", account.Currency)
		}
		return DepositResult{}, fmt.Errorf("wallet: lookup settlement: %w", err)
	}

	txID := uuid.New()
	posted, err := s.store.PostLedgerTransaction(ctx, PostLedgerTransactionParams{
		ID:          txID,
		Description: "deposit",
		Entries: []EntryInput{
			{AccountID: settlement.ID, Direction: EntryDirectionDebit, Amount: amount, Currency: account.Currency},
			{AccountID: account.ID, Direction: EntryDirectionCredit, Amount: amount, Currency: account.Currency},
		},
	})
	if err != nil {
		return DepositResult{}, fmt.Errorf("wallet: deposit: %w", err)
	}
	return DepositResult{
		ID:    posted.ID,
		Money: Money{Amount: amount, Currency: account.Currency},
	}, nil
}

// EntriesPage is one page of owned-Account Entries with an optional next cursor.
type EntriesPage struct {
	Entries    []Entry
	NextCursor string
}

// Entries lists Entries for an owned Account newest-first with keyset paging.
// limit 0 means DefaultPageSize; values outside 1..MaxPageSize are rejected.
func (s *Service) Entries(ctx context.Context, id uuid.UUID, limit int, cursor string) (EntriesPage, error) {
	if _, err := s.ownedAccount(ctx, id); err != nil {
		return EntriesPage{}, err
	}

	if limit == 0 {
		limit = DefaultPageSize
	}
	var v apperr.ValidationError
	if limit < 1 || limit > MaxPageSize {
		v.Add("limit", "out of range")
	}
	var beforeID *int64
	if cursor != "" {
		decoded, err := decodeEntryCursor(cursor)
		if err != nil {
			v.Add("cursor", "invalid cursor")
		} else {
			beforeID = &decoded
		}
	}
	if err := v.Err(); err != nil {
		return EntriesPage{}, err
	}

	rows, err := s.store.ListEntriesByAccountKeyset(ctx, id, beforeID, int32(limit+1))
	if err != nil {
		return EntriesPage{}, fmt.Errorf("wallet: list entries: %w", err)
	}

	nextCursor := ""
	if len(rows) > limit {
		nextCursor = encodeEntryCursor(rows[limit-1].ID)
		rows = rows[:limit]
	}
	if rows == nil {
		rows = []Entry{}
	}
	return EntriesPage{Entries: rows, NextCursor: nextCursor}, nil
}

func encodeEntryCursor(id int64) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(id, 10)))
}

func decodeEntryCursor(cursor string) (int64, error) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return 0, err
	}
	id, err := strconv.ParseInt(string(raw), 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid entry id")
	}
	return id, nil
}

func (s *Service) ownedAccount(ctx context.Context, id uuid.UUID) (Account, error) {
	account, err := s.store.GetAccount(ctx, id)
	if err != nil {
		if errors.Is(err, ErrAccountNotFound) {
			return Account{}, ErrWalletNotFound
		}
		return Account{}, fmt.Errorf("wallet: get wallet: %w", err)
	}
	if account.Kind == KindSystem {
		return Account{}, ErrWalletNotFound
	}
	return account, nil
}
