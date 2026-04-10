package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ctxKey string

const txKey ctxKey = "db_transaction"

type PTransactor struct {
	pool *pgxpool.Pool
}

func NewPtransactor(pool *pgxpool.Pool) *PTransactor {
	return &PTransactor{
		pool: pool,
	}
}
func (t *PTransactor) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := t.pool.Begin(ctx)
	if err != nil {
		return err
	}

	txCtx := context.WithValue(ctx, txKey, tx)

	err = fn(txCtx)
	if err != nil {
		tx.Rollback(ctx)
		return err
	}

	return tx.Commit(ctx)
}

func GetTx(ctx context.Context) pgx.Tx {
	tx, ok := ctx.Value(txKey).(pgx.Tx)
	if ok {
		return tx
	}
	return nil
}
