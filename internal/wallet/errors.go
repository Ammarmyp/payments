package wallet

import "errors"

var (
	// ErrEmptyLedgerTransaction is returned when a post has no Entries.
	ErrEmptyLedgerTransaction = errors.New("wallet: ledger transaction has no entries")

	// ErrUnbalancedLedgerTransaction is returned when debits and credits differ
	// for any Currency, either from Go pre-validation or a Postgres check_violation.
	ErrUnbalancedLedgerTransaction = errors.New("wallet: ledger transaction is not balanced")

	// ErrLedgerInvariant is returned when Postgres rejects a post for a ledger
	// constraint other than the balance/non-empty check (for example FK failures).
	ErrLedgerInvariant = errors.New("wallet: ledger invariant violated")
)
