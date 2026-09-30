package txmanager

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TxManager управляет транзакциями.
type TxManager interface {
	// Do выполняет fn в транзакции.
	// Если fn возвращает ошибку — ROLLBACK.
	// Если fn паникует — ROLLBACK и проброс паники.
	// Если транзакция уже открыта в ctx — переиспользует её.
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type manager struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) TxManager {
	return &manager{pool: pool}
}

func (m *manager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	// Если транзакция уже открыта — переиспользуем её.
	if tx := TxFromContext(ctx); tx != nil {
		return fn(ctx)
	}

	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	// Страховка: если Commit не вызван — откат.
	// После успешного Commit повторный Rollback безопасен (no-op).
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	ctx = WithTx(ctx, tx)

	// Обработка паники: откат и проброс наружу.
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	if err := fn(ctx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
