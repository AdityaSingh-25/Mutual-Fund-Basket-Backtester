package db

import (
	"fmt"
	"io/fs"
	"log"
	"sort"
)

// RunMigrations applies every *.sql file in files in lexical order, recording
// each applied file in a schema_migrations table so it runs exactly once.
// It is safe to call on every startup.
func RunMigrations(files fs.FS) error {
	if _, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	names, err := fs.Glob(files, "*.sql")
	if err != nil {
		return err
	}
	sort.Strings(names)

	for _, name := range names {
		var applied bool
		if err := DB.QueryRow(
			`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`, name,
		).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %s: %w", name, err)
		}
		if applied {
			continue
		}

		stmt, err := fs.ReadFile(files, name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		// Apply the migration and record it atomically.
		tx, err := DB.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(stmt)); err != nil {
			tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
		if _, err := tx.Exec(
			`INSERT INTO schema_migrations (version) VALUES ($1)`, name,
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %s: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}

		log.Printf("applied migration %s", name)
	}

	return nil
}
