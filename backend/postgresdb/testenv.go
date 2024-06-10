package postgresdb

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/require"
	"log/slog"
	"os"
	"testing"
)

func WithPostgres(t *testing.T, f func(t *testing.T, connectionDetails DirectConfig)) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	defer cancel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	pool, err := dockertest.NewPool("")
	require.NoError(t, err)

	var (
		password = "test"
		dbName   = "test"
		user     = "test"
	)

	resource, err := pool.Run("postgres", "latest", []string{
		fmt.Sprintf("POSTGRES_PASSWORD=%s", password),
		fmt.Sprintf("POSTGRES_DB=%s", dbName),
		fmt.Sprintf("POSTGRES_USER=%s", user),
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		err := pool.Purge(resource)
		if err != nil {
			t.Log("docker pool purge: ", err)
		}
	})

	port := resource.GetPort("5432/tcp")
	t.Logf("Postgres container started: port: %s, password: %s, dbName: %s, user: %s", port, password, dbName, user)

	config := DirectConfig{
		Hostname: "localhost",
		Port:     port,
		UserName: user,
		Password: password,
		Database: dbName,
	}

	// Wait for postgres to start.
	err = pool.Retry(func() error {
		db, err := NewDirectConnection(ctx, config, logger)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		return db.Ping()
	})
	require.NoError(t, err)

	t.Log("Database connection ready")
	logger.Info("Database connection ready")

	f(t, config)
}

func WithDatabase(t *testing.T, f func(t *testing.T, db *sql.DB)) {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	WithPostgres(t, func(t *testing.T, connectionDetails DirectConfig) {
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)
		defer cancel()

		db, err := NewDirectConnection(ctx, connectionDetails, logger)
		require.NoError(t, err)

		f(t, db)
	})
}
