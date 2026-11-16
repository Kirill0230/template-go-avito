package transaction

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type txKey struct{}

type TxManagerImpl struct {
	pool    *pgxpool.Pool
	timeout time.Duration
}

func NewTxManagerImpl(pool *pgxpool.Pool, timeout time.Duration) *TxManagerImpl {
	return &TxManagerImpl{pool: pool, timeout: timeout}
}

func (t *TxManagerImpl) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	_, ok := ctx.Value(txKey{}).(pgx.Tx)
	if ok {
		return fn(ctx)
	}

	beginCtx, cancel := context.WithTimeout(ctx, t.timeout)
	tx, err := t.pool.BeginTx(beginCtx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	cancel()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), t.timeout)
		defer cancel()
		_ = tx.Rollback(rollbackCtx)
	}()

	txCtx := context.WithValue(ctx, txKey{}, tx)
	err = fn(txCtx)
	if err != nil {
		return err
	}

	commitCtx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()
	err = tx.Commit(commitCtx)
	if err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}
