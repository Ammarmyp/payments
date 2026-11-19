package postgres_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/Ammarmyp/payments/internal/platform/postgres"
	"github.com/jackc/pgx/v5"
)

func testConfig(t *testing.T) postgres.Config {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres InTx tests")
	}
	return postgres.DefaultConfig(url)
}

func TestInTxCommitsOnSuccess(t *testing.T) {
	cfg := testConfig(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := postgres.Open(ctx, cfg)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer pool.Close()

	var saw bool
	err = postgres.InTx(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		saw = true
		_, err := tx.Exec(ctx, `SELECT 1`)
		return err
	})
	if err != nil {
		t.Fatalf("InTx: %v", err)
	}
	if !saw {
		t.Fatal("expected fn to run")
	}
}

func TestInTxRollsBackOnError(t *testing.T) {
	cfg := testConfig(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := postgres.Open(ctx, cfg)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer pool.Close()

	sentinel := errors.New("boom")
	err = postgres.InTx(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT 1`); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("error: got %v want %v", err, sentinel)
	}
}

func TestInTxRespectsCanceledContext(t *testing.T) {
	cfg := testConfig(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := postgres.Open(ctx, cfg)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer pool.Close()

	canceled, cancelFn := context.WithCancel(ctx)
	cancelFn()

	err = postgres.InTx(canceled, pool, func(ctx context.Context, tx pgx.Tx) error {
		t.Fatal("fn should not run when begin context is canceled")
		return nil
	})
	if err == nil {
		t.Fatal("expected begin to fail on canceled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error: got %v want context.Canceled", err)
	}
}

func TestInTxRollsBackAndRepanics(t *testing.T) {
	cfg := testConfig(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := postgres.Open(ctx, cfg)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer pool.Close()

	defer func() {
		if r := recover(); r != "explode" {
			t.Fatalf("panic: got %v want explode", r)
		}
	}()

	_ = postgres.InTx(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT 1`); err != nil {
			return err
		}
		panic("explode")
	})
	t.Fatal("expected panic")
}
