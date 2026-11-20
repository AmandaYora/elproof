package infrastructure

import (
	"context"
	"database/sql"
)

// dbtx is what a repository needs from either the pool or a transaction.
type dbtx interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type txKey struct{}

// conn is the executor a quotation repository method must use: the
// transaction the caller started with TxManager.InTx when ctx carries one,
// otherwise the pool. The transaction never leaves the offers module (a module
// owns its tables; cross-module work is orchestrated above it).
func conn(ctx context.Context, db *sql.DB) dbtx {
	if tx, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
		return tx
	}
	return db
}

// TxManager runs a function inside one database transaction of offers.
type TxManager struct {
	db *sql.DB
}

func NewTxManager(db *sql.DB) *TxManager { return &TxManager{db: db} }

// InTx begins a transaction, hands fn a context that carries it, commits when
// fn returns nil and rolls back on an error or a panic. A call made while a
// transaction is already open joins it (no nesting): the outermost caller
// decides the outcome.
func (m *TxManager) InTx(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	if _, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
		return fn(ctx)
	}
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	if err = fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		return err
	}
	return tx.Commit()
}

// runInTx runs fn on the ambient transaction when there is one (the caller
// commits), otherwise in a transaction of its own.
func runInTx(ctx context.Context, db *sql.DB, fn func(q dbtx) error) error {
	if tx, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
		return fn(tx)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}
