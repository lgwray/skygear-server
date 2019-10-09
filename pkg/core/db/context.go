package db

import (
	"context"
	"errors"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"github.com/skygeario/skygear-server/pkg/core/config"
)

type contextKey string

var (
	keyContainer = contextKey("container")
)

// Context provides db with the interface for retrieving an interface to execute sql
type Context interface {
	DB() ExtContext
}

// TxContext provides the interface for managing transaction
type TxContext interface {
	SafeTxContext

	BeginTx() error
	CommitTx() error
	RollbackTx() error
}

// SafeTxContext only provides interface to check existence of transaction
type SafeTxContext interface {
	HasTx() bool
	EnsureTx()
}

// EndTx implements a common pattern that commit a transaction if no error is
// presented, otherwise rollback the transaction.
func EndTx(tx TxContext, err error) error {
	if err != nil {
		if rbErr := tx.RollbackTx(); rbErr != nil {
			logrus.Errorf("Failed to rollback: %v", rbErr)
		}
		return err
	}

	return tx.CommitTx()
}

// WithTx provides a convenient way to wrap a function within a transaction
func WithTx(tx TxContext, do func() error) (err error) {
	if err = tx.BeginTx(); err != nil {
		return
	}

	err = do()
	return EndTx(tx, err)
}

// TODO: handle thread safety
type contextContainer struct {
	pool Pool
	db   *sqlx.DB
	tx   *sqlx.Tx
}

type dbContext struct {
	context.Context
	tConfig config.TenantConfiguration
}

func InitDBContext(ctx context.Context, pool Pool) context.Context {
	container := &contextContainer{pool: pool}
	return context.WithValue(ctx, keyContainer, container)
}

// InitRequestDBContext initialize db context for the request
func InitRequestDBContext(req *http.Request, pool Pool) *http.Request {
	return req.WithContext(InitDBContext(req.Context(), pool))
}

func newDBContext(ctx context.Context, tConfig config.TenantConfiguration) *dbContext {
	return &dbContext{Context: ctx, tConfig: tConfig}
}

// NewContextWithContext creates a new context.DB from context
func NewContextWithContext(ctx context.Context, tConfig config.TenantConfiguration) Context {
	return newDBContext(ctx, tConfig)
}

// NewTxContextWithContext creates a new context.Tx from context
func NewTxContextWithContext(ctx context.Context, tConfig config.TenantConfiguration) TxContext {
	return newDBContext(ctx, tConfig)
}

// NewSafeTxContextWithContext creates a new context.Tx from context
func NewSafeTxContextWithContext(ctx context.Context, tConfig config.TenantConfiguration) SafeTxContext {
	return newDBContext(ctx, tConfig)
}

func (d *dbContext) DB() ExtContext {
	if d.tx() != nil {
		return d.tx()
	}

	return d.lazydb()
}

func (d *dbContext) HasTx() bool {
	return d.tx() != nil
}

func (d *dbContext) EnsureTx() {
	if d.tx() == nil {
		panic(errors.New("unexpected transaction not began"))
	}
}

func (d *dbContext) BeginTx() (err error) {
	if d.tx() != nil {
		err = ErrDatabaseTxDidBegin
		return err
	}

	var tx *sqlx.Tx
	tx, err = d.lazydb().BeginTxx(d, nil)
	if err != nil {
		return err
	}

	container := d.container()
	container.tx = tx

	return
}

func (d *dbContext) CommitTx() (err error) {
	if d.tx() == nil {
		err = ErrDatabaseTxDidNotBegin
		return err
	}

	if err = d.tx().Commit(); err == nil {
		container := d.container()
		container.tx = nil
	}

	return
}

func (d *dbContext) RollbackTx() (err error) {
	if d.tx() == nil {
		err = ErrDatabaseTxDidNotBegin
		return
	}

	if err = d.tx().Rollback(); err == nil {
		container := d.container()
		container.tx = nil
	}

	return
}

func (d *dbContext) db() *sqlx.DB {
	return d.container().db
}

func (d *dbContext) tx() *sqlx.Tx {
	return d.container().tx
}

func (d *dbContext) lazydb() *sqlx.DB {
	db := d.db()
	if db == nil {
		container := d.container()

		var err error
		if db, err = container.pool.Open(d.tConfig); err != nil {
			panic(err)
		}

		container.db = db
	}

	return db
}

func (d *dbContext) container() *contextContainer {
	return d.Value(keyContainer).(*contextContainer)
}