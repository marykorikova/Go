package trip

import (
	"context"

	"github.com/marykorikova/Go/internal/txmanager"
)

type Repository struct {
	pool txmanager.Executor
}

func New(pool txmanager.Executor) *Repository {
	return &Repository{
		pool: pool,
	}
}

func (r *Repository) getExecutor(ctx context.Context) txmanager.Executor {
	if tx := txmanager.TxFromContext(ctx); tx != nil {
		return tx
	}

	return r.pool
}
