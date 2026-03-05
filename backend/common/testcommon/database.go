package testcommon

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"testing"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/globals"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	enttestpkg "github.com/NicoClack/cryptic-stash/backend/ent/enttest"
	"github.com/NicoClack/cryptic-stash/backend/ent/migrate"
	_ "github.com/NicoClack/cryptic-stash/backend/entps"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

type TestDatabase struct {
	client       *ent.Client
	logger       common.Logger
	startTxHooks []func(tx *ent.Tx) error
}

var (
	dbCounter = int64(0)
)

func CreateDB(t *testing.T) *TestDatabase {
	t.Helper()

	globals.MigrateMu.Lock()
	defer globals.MigrateMu.Unlock()
	dbCounter++
	db, stdErr := sql.Open("sqlite3", fmt.Sprintf(
		"file:temp%v?mode=memory&cache=shared",
		dbCounter,
	))
	require.NoError(t, stdErr)

	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(100)
	db.SetConnMaxLifetime(time.Hour)
	driver := ent.Driver(entsql.OpenDB("sqlite3", db))

	client := enttestpkg.NewClient(t, enttestpkg.WithOptions(driver))

	goose.SetBaseFS(migrate.MigrationsFS)
	stdErr = goose.SetDialect("sqlite3")
	if stdErr != nil {
		_ = client.Close()
		t.Fatalf("couldn't set goose dialect. error: %v", stdErr)
	}

	stdErr = goose.Up(db, "migrations")
	if stdErr != nil {
		_ = client.Close()
		t.Fatalf("migration failed: %v", stdErr)
	}

	// TODO: take logger as argument?
	slog.SetLogLoggerLevel(slog.LevelDebug)

	return &TestDatabase{
		client:       client,
		logger:       common.GetLogger(context.Background(), nil),
		startTxHooks: []func(tx *ent.Tx) error{},
	}
}

func (db *TestDatabase) Start() {
	// TODO: move initialisation logic into here like the real DB service?
}
func (db *TestDatabase) Client() *ent.Client {
	return db.client
}
func (db *TestDatabase) ReadTx(ctx context.Context) (*ent.Tx, error) {
	tx, stdErr := db.client.BeginTx(ctx, &sql.TxOptions{
		ReadOnly: true,
	})
	if stdErr != nil {
		return nil, stdErr
	}
	stdErr = db.runStartTxHooks(tx)
	if stdErr != nil {
		_ = tx.Rollback()
		return nil, stdErr
	}
	return tx, nil
}
func (db *TestDatabase) WriteTx(ctx context.Context) (*ent.Tx, error) {
	tx, stdErr := db.client.Tx(ctx)
	if stdErr != nil {
		return nil, stdErr
	}
	stdErr = db.runStartTxHooks(tx)
	if stdErr != nil {
		_ = tx.Rollback()
		return nil, stdErr
	}
	return tx, nil
}
func (db *TestDatabase) Shutdown() {
	stdErr := db.client.Close()
	if stdErr != nil {
		db.logger.Warn("an error occurred while shutting down a test database", "error", stdErr)
	}
}
func (db *TestDatabase) DefaultLogger() common.Logger {
	return db.logger
}

func (db *TestDatabase) AddStartTxHook(hook func(tx *ent.Tx) error) {
	db.startTxHooks = append(db.startTxHooks, hook)
}
func (db *TestDatabase) runStartTxHooks(tx *ent.Tx) error {
	for _, hook := range db.startTxHooks {
		stdErr := hook(tx)
		if stdErr != nil {
			return stdErr
		}
	}
	return nil
}
