//go:build integration

package txmanager

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"testing"
)

func TestTxManager_Commit_Integration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping database: %v", err)
	}

	txm := New(pool)

	err = txm.Do(ctx, func(txCtx context.Context) error {
		tx := TxFromContext(txCtx)
		if tx == nil {
			t.Fatal("transaction not found in context")
		}

		_, err := tx.Exec(
			txCtx,
			"CREATE TEMP TABLE txmanager_test (value integer)",
		)
		if err != nil {
			return err
		}

		_, err = tx.Exec(
			txCtx,
			"INSERT INTO txmanager_test (value) VALUES (42)",
		)
		return err
	})

	if err != nil {
		t.Fatalf("transaction failed: %v", err)
	}
}

func TestTxManager_Rollback_Integration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping database: %v", err)
	}

	// Выделяем одно соединение, чтобы временная таблица
	// была доступна до и после транзакции.
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire connection: %v", err)
	}
	defer conn.Release()

	_, err = conn.Exec(ctx, `
		CREATE TEMP TABLE txmanager_rollback_test (
			value integer
		)
	`)
	if err != nil {
		t.Fatalf("create temp table: %v", err)
	}

	// Используем отдельный пул с одним соединением,
	// чтобы txmanager работал с этой же временной таблицей.
	testPool, err := pgxpool.NewWithConfig(ctx, func() *pgxpool.Config {
		cfg, parseErr := pgxpool.ParseConfig(databaseURL)
		if parseErr != nil {
			t.Fatalf("parse config: %v", parseErr)
		}
		cfg.MaxConns = 1
		return cfg
	}())
	if err != nil {
		t.Fatalf("create test pool: %v", err)
	}
	t.Cleanup(testPool.Close)

	// Создаём таблицу непосредственно в соединении testPool.
	_, err = testPool.Exec(ctx, `
		CREATE TEMP TABLE txmanager_rollback_check (
			value integer
		)
	`)
	if err != nil {
		t.Fatalf("create rollback check table: %v", err)
	}

	txm := New(testPool)
	expectedErr := errors.New("test rollback")

	err = txm.Do(ctx, func(txCtx context.Context) error {
		tx := TxFromContext(txCtx)
		if tx == nil {
			return errors.New("transaction not found in context")
		}

		_, execErr := tx.Exec(
			txCtx,
			"INSERT INTO txmanager_rollback_check (value) VALUES (42)",
		)
		if execErr != nil {
			return execErr
		}

		return expectedErr
	})

	if !errors.Is(err, expectedErr) {
		t.Fatalf("got error %v, want %v", err, expectedErr)
	}

	var count int

	err = testPool.QueryRow(
		ctx,
		"SELECT COUNT(*) FROM txmanager_rollback_check",
	).Scan(&count)
	if err != nil {
		t.Fatalf("count rows: %v", err)
	}

	if count != 0 {
		t.Errorf("rows after rollback = %d, want 0", count)
	}
}

func TestTxManager_Nested_Integration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping database: %v", err)
	}

	txm := New(pool)

	err = txm.Do(ctx, func(outerCtx context.Context) error {
		outerTx := TxFromContext(outerCtx)
		if outerTx == nil {
			return errors.New("outer transaction not found")
		}

		return txm.Do(outerCtx, func(innerCtx context.Context) error {
			innerTx := TxFromContext(innerCtx)
			if innerTx == nil {
				return errors.New("inner transaction not found")
			}

			if innerTx != outerTx {
				return errors.New("nested Do opened a different transaction")
			}

			return nil
		})
	})

	if err != nil {
		t.Fatalf("nested transaction failed: %v", err)
	}
}
