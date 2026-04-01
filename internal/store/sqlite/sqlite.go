package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	store := &Store{db: db}
	if err := store.configure(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) configure() error {
	if _, err := s.db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return fmt.Errorf("enable foreign_keys: %w", err)
	}
	if _, err := s.db.Exec("PRAGMA journal_mode = WAL;"); err != nil {
		return fmt.Errorf("set journal_mode wal: %w", err)
	}
	return nil
}

func (s *Store) Migrate(ctx context.Context) error {
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}

	ordered := make([]fs.DirEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		ordered = append(ordered, entry)
	}

	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].Name() < ordered[j].Name()
	})

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	for _, migration := range ordered {
		version, err := parseMigrationVersion(migration.Name())
		if err != nil {
			return err
		}

		var exists int
		err = tx.QueryRowContext(ctx, "SELECT 1 FROM schema_migrations WHERE version = ?", version).Scan(&exists)
		if err == nil {
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("check migration version %d: %w", version, err)
		}

		sqlBytes, err := migrationFiles.ReadFile("migrations/" + migration.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", migration.Name(), err)
		}

		if _, err := tx.ExecContext(ctx, string(sqlBytes)); err != nil {
			return fmt.Errorf("apply migration %s: %w", migration.Name(), err)
		}

		if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations(version, name) VALUES (?, ?)", version, migration.Name()); err != nil {
			return fmt.Errorf("insert migration %s: %w", migration.Name(), err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration tx: %w", err)
	}

	return nil
}

func parseMigrationVersion(name string) (int, error) {
	parts := strings.SplitN(name, "_", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid migration filename: %s", name)
	}

	version, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid migration version in %s: %w", name, err)
	}

	return version, nil
}
