package db

import (
	"context"
	"embed"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

// migrationsFS embeds the SQL migrations into the server binary so Render does
// not need a separate migration step or filesystem path at runtime.
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	files, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read embedded migrations: %w", err)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		sqlBytes, err := migrationsFS.ReadFile("migrations/" + file.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", file.Name(), err)
		}
		if _, err := pool.Exec(ctx, string(sqlBytes)); err != nil {
			return fmt.Errorf("run migration %s: %w", file.Name(), err)
		}
	}
	return nil
}
