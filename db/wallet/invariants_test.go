package wallet_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func testDatabaseURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping ledger schema tests")
	}
	return url
}

func connect(t *testing.T) *pgx.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	conn, err := pgx.Connect(ctx, testDatabaseURL(t))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	return conn
}

// seedAccounts returns a debit-normal System Account and a credit-normal owned
// Account (ADR-0002). Callers debit the system row and credit the owned row.
func seedAccounts(t *testing.T, conn *pgx.Conn) (systemAccount, ownedAccount uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	systemAccount = uuid.New()
	ownedAccount = uuid.New()

	_, err := conn.Exec(ctx, `
		INSERT INTO accounts (id, owner_id, name, currency, normal_balance, kind)
		VALUES
			($1, NULL, $2, 'USD', 'debit', 'system'),
			($3, $4, 'USD wallet', 'USD', 'credit', 'wallet')
	`, systemAccount, fmt.Sprintf("sys-%s", systemAccount), ownedAccount, uuid.New())
	if err != nil {
		t.Fatalf("seed accounts: %v", err)
	}
	return systemAccount, ownedAccount
}

func isPGCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}

func TestLedgerTransactionRejectsEmptyCommit(t *testing.T) {
	conn := connect(t)
	ctx := context.Background()
	txID := uuid.New()

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO transactions (id, description) VALUES ($1, 'empty')
	`, txID)
	if err != nil {
		t.Fatalf("insert transaction: %v", err)
	}

	err = tx.Commit(ctx)
	if err == nil {
		t.Fatal("expected empty Ledger Transaction commit to fail")
	}
	if !isPGCode(err, "23514") { // check_violation
		t.Fatalf("expected check_violation, got %v", err)
	}
}

func TestLedgerTransactionRejectsUnbalancedCommit(t *testing.T) {
	conn := connect(t)
	ctx := context.Background()
	systemAccount, _ := seedAccounts(t, conn)
	txID := uuid.New()

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO transactions (id, description) VALUES ($1, 'unbalanced')
	`, txID)
	if err != nil {
		t.Fatalf("insert transaction: %v", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO entries (transaction_id, account_id, direction, amount, currency)
		VALUES ($1, $2, 'debit', 100, 'USD')
	`, txID, systemAccount)
	if err != nil {
		t.Fatalf("insert entry: %v", err)
	}

	err = tx.Commit(ctx)
	if err == nil {
		t.Fatal("expected unbalanced Ledger Transaction commit to fail")
	}
	if !isPGCode(err, "23514") {
		t.Fatalf("expected check_violation, got %v", err)
	}
}

func TestLedgerTransactionAcceptsBalancedCommit(t *testing.T) {
	conn := connect(t)
	ctx := context.Background()
	systemAccount, ownedAccount := seedAccounts(t, conn)
	txID := uuid.New()

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO transactions (id, description) VALUES ($1, 'balanced')
	`, txID)
	if err != nil {
		t.Fatalf("insert transaction: %v", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO entries (transaction_id, account_id, direction, amount, currency)
		VALUES
			($1, $2, 'debit', 50, 'USD'),
			($1, $3, 'credit', 50, 'USD')
	`, txID, systemAccount, ownedAccount)
	if err != nil {
		t.Fatalf("insert entries: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("balanced commit: %v", err)
	}
}

func postBalanced(t *testing.T, conn *pgx.Conn, description string, amount int64) (txID uuid.UUID, entryID int64) {
	t.Helper()
	ctx := context.Background()
	systemAccount, ownedAccount := seedAccounts(t, conn)
	txID = uuid.New()

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO transactions (id, description) VALUES ($1, $2)
	`, txID, description)
	if err != nil {
		t.Fatalf("insert transaction: %v", err)
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO entries (transaction_id, account_id, direction, amount, currency)
		VALUES ($1, $2, 'debit', $3, 'USD')
		RETURNING id
	`, txID, systemAccount, amount).Scan(&entryID)
	if err != nil {
		t.Fatalf("insert debit entry: %v", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO entries (transaction_id, account_id, direction, amount, currency)
		VALUES ($1, $2, 'credit', $3, 'USD')
	`, txID, ownedAccount, amount)
	if err != nil {
		t.Fatalf("insert credit entry: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit balanced post: %v", err)
	}
	return txID, entryID
}

func TestLedgerTransactionIsAppendOnly(t *testing.T) {
	conn := connect(t)
	ctx := context.Background()
	txID, _ := postBalanced(t, conn, "locked", 25)

	_, err := conn.Exec(ctx, `UPDATE transactions SET description = 'mutated' WHERE id = $1`, txID)
	if err == nil {
		t.Fatal("expected UPDATE on transactions to fail")
	}
	if !isPGCode(err, "23001") { // restrict_violation
		t.Fatalf("expected restrict_violation on UPDATE transactions, got %v", err)
	}

	_, err = conn.Exec(ctx, `DELETE FROM transactions WHERE id = $1`, txID)
	if err == nil {
		t.Fatal("expected DELETE on transactions to fail")
	}
	if !isPGCode(err, "23001") {
		t.Fatalf("expected restrict_violation on DELETE transactions, got %v", err)
	}
}

func TestEntriesAreAppendOnly(t *testing.T) {
	conn := connect(t)
	ctx := context.Background()
	_, entryID := postBalanced(t, conn, "entries-locked", 10)

	_, err := conn.Exec(ctx, `UPDATE entries SET amount = 99 WHERE id = $1`, entryID)
	if err == nil {
		t.Fatal("expected UPDATE on entries to fail")
	}
	if !isPGCode(err, "23001") {
		t.Fatalf("expected restrict_violation on UPDATE entries, got %v", err)
	}

	_, err = conn.Exec(ctx, `DELETE FROM entries WHERE id = $1`, entryID)
	if err == nil {
		t.Fatal("expected DELETE on entries to fail")
	}
	if !isPGCode(err, "23001") {
		t.Fatalf("expected restrict_violation on DELETE entries, got %v", err)
	}
}

func TestLedgerTransactionCreatedAtDefaults(t *testing.T) {
	conn := connect(t)
	ctx := context.Background()
	systemAccount, ownedAccount := seedAccounts(t, conn)
	txID := uuid.New()

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO transactions (id, description) VALUES ($1, 'default created_at')
	`, txID)
	if err != nil {
		t.Fatalf("insert without created_at: %v", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO entries (transaction_id, account_id, direction, amount, currency)
		VALUES
			($1, $2, 'debit', 5, 'USD'),
			($1, $3, 'credit', 5, 'USD')
	`, txID, systemAccount, ownedAccount)
	if err != nil {
		t.Fatalf("insert entries: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}

	var createdAt time.Time
	err = conn.QueryRow(ctx, `SELECT created_at FROM transactions WHERE id = $1`, txID).Scan(&createdAt)
	if err != nil {
		t.Fatalf("select created_at: %v", err)
	}
	if createdAt.IsZero() {
		t.Fatal("expected created_at default to be set")
	}
}
