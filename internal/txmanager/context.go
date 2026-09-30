package txmanager

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type contextKey struct{}

var txKey = contextKey{}

// WithTx кладёт транзакцию в контекст.
func WithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txKey, tx)
}

// TxFromContext достаёт транзакцию из контекста.
// Возвращает nil, если транзакции нет.
func TxFromContext(ctx context.Context) pgx.Tx {
	tx, _ := ctx.Value(txKey).(pgx.Tx)
	return tx
}
