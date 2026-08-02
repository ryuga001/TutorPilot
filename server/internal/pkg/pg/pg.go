package pg

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Beginner interface {
	Querier
	Begin(ctx context.Context) (pgx.Tx, error)
}

func InTx[T any](ctx context.Context, db Beginner, fn func(q Querier) (T, error)) (T, error) {
	var zero T

	tx, err := db.Begin(ctx)
	if err != nil {
		return zero, err
	}

	committed := false
	defer func() {
		if committed {
			return
		}

		_ = tx.Rollback(context.WithoutCancel(ctx))
	}()

	out, err := fn(tx)
	if err != nil {
		return zero, err
	}
	if err := tx.Commit(ctx); err != nil {
		return zero, err
	}
	committed = true
	return out, nil
}
