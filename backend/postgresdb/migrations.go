package postgresdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	migrate "github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"io/fs"
	"log/slog"
	"os"
)

var ErrSchemaOutOfDate = errors.New("schema is out of date")

type Migrator struct {
	db     *sql.DB
	logger *slog.Logger
	cfg    MigrationConfig
}

func NewMigrator(
	ctx context.Context,
	db *sql.DB,
	logger *slog.Logger,
	cfg MigrationConfig,
) (*Migrator, error) {
	// Ensure live connection
	err := db.PingContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Migrator{
		db:     db,
		logger: logger,
		cfg:    cfg,
	}, nil
}

type MigrationConfig struct {
	// ApplyMigrations determines whether to apply migrations.
	// If false, the migrator will return an error if the database schema is out of date.
	ApplyMigrations bool
	MigrationsFS    fs.FS
	MigrationsPath  string
}

func (m *Migrator) Run(ctx context.Context) error {
	migrationsSchema := "internal_state"

	m.logger.InfoContext(ctx, "ensure internal_state schema")
	_, err := m.db.ExecContext(ctx, fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", migrationsSchema))
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	m.logger.InfoContext(ctx, "checking database schema")
	driver, err := postgres.WithInstance(m.db, &postgres.Config{
		SchemaName: migrationsSchema,
	})
	if err != nil {
		return fmt.Errorf("failed to create driver: %w", err)
	}
	defer func(driver database.Driver) {
		err := driver.Close()
		if err != nil {
			if errors.Is(err, sql.ErrConnDone) {
				return
			}
			//m.logger.ErrorContext(ctx, "failed to close migration driver", "error", err)
		}
	}(driver)

	src, err := iofs.New(m.cfg.MigrationsFS, m.cfg.MigrationsPath)
	if err != nil {
		return fmt.Errorf("failed to open migrations source: %w", err)
	}
	defer func(src source.Driver) {
		err := src.Close()
		if err != nil {
			m.logger.ErrorContext(ctx, "failed to close migration source", "error", err)
		}
	}(src)

	mi, err := migrate.NewWithInstance("embeded", src, "postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer func(mi *migrate.Migrate) {
		err, _ := mi.Close()
		if err != nil {
			m.logger.ErrorContext(ctx, "failed to close migrate instance", "error", err)
		}
	}(mi)

	// Get the current version of the database schema.
	version, dirty, err := mi.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return fmt.Errorf("failed to get version: %w", err)
	}
	m.logger.DebugContext(ctx, "database schema version", "version", version, "dirty", dirty)

	// Get the next version of the database schema.
	_, err = src.Next(version)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to get next migration: %w", err)
	}
	isUpToDate := errors.Is(err, os.ErrNotExist)

	if isUpToDate {
		m.logger.InfoContext(ctx, "database schema is up to date")
		return nil
	}

	if !m.cfg.ApplyMigrations {
		m.logger.WarnContext(ctx, "database schema is out of date, but migrations are disabled")
		return fmt.Errorf("schema at version (%d): %w", version, ErrSchemaOutOfDate)
	}

	m.logger.InfoContext(ctx, "migrating database schema")

	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			m.logger.WarnContext(ctx, "context canceled: stopping database schema migration")
			mi.GracefulStop <- true
		case <-done:
		}
	}()

	err = mi.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to migrate: %w", err)
	}

	m.logger.InfoContext(ctx, "database schema migrated")
	return nil
}
