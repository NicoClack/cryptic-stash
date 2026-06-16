package services

import (
	"context"
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/NicoClack/cryptic-stash/backend/common"
	"github.com/NicoClack/cryptic-stash/backend/common/globals"
	"github.com/NicoClack/cryptic-stash/backend/ent"
	"github.com/NicoClack/cryptic-stash/backend/ent/migrate"
	"github.com/NicoClack/cryptic-stash/backend/ent/schema"
	_ "github.com/NicoClack/cryptic-stash/backend/entps"
	"github.com/pressly/goose/v3"
)

func NewDatabase(app *common.App) *Database {
	return &Database{
		app: app,
	}
}

type Database struct {
	app          *common.App
	client       *ent.Client
	startOnce    sync.Once
	shutdownOnce sync.Once
	mu           sync.RWMutex
}

func (service *Database) Start() {
	service.startOnce.Do(func() {
		// When the program normally runs, it shouldn't be possible for there to be parallel calls of this.
		// However, the tests for this file can run in parallel.
		// We could disable that, but there could still be a race condition with testcommon.CreateDB(), so they share a mutex
		globals.MigrateMu.Lock()
		defer globals.MigrateMu.Unlock()

		stdErr := os.MkdirAll(service.app.Env.MOUNT_PATH, 0700)
		if stdErr != nil {
			log.Fatalf("couldn't create storage directory. error:\n%v", stdErr)
		}

		db, stdErr := sql.Open("sqlite3", filepath.Join(service.app.Env.MOUNT_PATH, "database.sqlite3"))
		if stdErr != nil {
			log.Fatalf("couldn't start database. error:\n%v", stdErr)
		}

		db.SetMaxIdleConns(5)
		db.SetMaxOpenConns(100)
		db.SetConnMaxLifetime(time.Hour)

		schema.Init(service.app.Env.BASE_ENCRYPTION_KEY)
		driver := ent.Driver(entsql.OpenDB("sqlite3", db))
		client := ent.NewClient(driver)

		goose.SetBaseFS(migrate.MigrationsFS)
		stdErr = goose.SetDialect("sqlite3")
		if stdErr != nil {
			log.Fatalf("couldn't set goose dialect. error\n%v", stdErr)
		}

		service.app.Logger.Info("running migrations...")
		stdErr = goose.Up(db, "migrations")
		if stdErr != nil {
			log.Fatalf("migration failed: %v", stdErr)
		}
		service.app.Logger.Info("migrations complete")

		service.mu.Lock()
		defer service.mu.Unlock()
		service.client = client
	})
}

func (service *Database) Client() *ent.Client {
	service.mu.RLock()
	defer service.mu.RUnlock()
	if service.client == nil {
		panic("can't get database client, service isn't running")
	}
	return service.client
}
func (service *Database) ReadTx(ctx context.Context) (*ent.Tx, error) {
	return service.Client().BeginTx(ctx, &sql.TxOptions{
		ReadOnly: true,
	})
}
func (service *Database) WriteTx(ctx context.Context) (*ent.Tx, error) {
	return service.Client().Tx(ctx)
}

func (service *Database) Shutdown() {
	service.startOnce.Do(func() {})
	service.shutdownOnce.Do(func() {
		service.mu.Lock()
		client := service.client
		service.client = nil
		service.mu.Unlock()

		if client != nil {
			stdErr := client.Close()
			if stdErr != nil {
				service.app.Logger.Warn("an error occurred while shutting down the database", stdErr)
			}
		}
	})
}
func (service *Database) DefaultLogger() common.Logger {
	return service.app.Logger
}
