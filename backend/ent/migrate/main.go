//go:build ignore

// Credit: adapted from Ent's docs

package main

import (
	"context"
	"log"
	"os"

	atlas "ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/sqltool"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql/schema"
	"github.com/NicoClack/cryptic-stash/backend/ent/migrate"
	_ "github.com/NicoClack/cryptic-stash/backend/entps"
)

func main() {
    ctx := context.Background()
    dir, stdErr := atlas.NewLocalDir("ent/migrate/migrations")
    if stdErr != nil {
        log.Fatalf("failed creating atlas migration directory: %v", stdErr)
    }
    opts := []schema.MigrateOption{
        schema.WithDir(dir),                         
        schema.WithMigrationMode(schema.ModeReplay), 
        schema.WithDialect(dialect.SQLite),           
        schema.WithFormatter(sqltool.GooseFormatter),
    }
    if len(os.Args) != 2 {
        log.Fatalln("migration name is required. Use: 'go run -mod=mod ent/migrate/main.go <name>'")
    }
    stdErr = migrate.NamedDiff(ctx, "sqlite://file?mode=memory", os.Args[1], opts...)
    if stdErr != nil {
        log.Fatalf("failed generating migration file: %v", stdErr)
    }
}