package api

import (
	"context"
	"fmt"
	"github.com/ory/dockertest/v3"
	"log/slog"
	"sidus.io/home-call/postgresdb"
)

type TestPostgres struct {
	pool      *dockertest.Pool
	container *dockertest.Resource
}

func NewTestPostgres() (*TestPostgres, error) {
	return &TestPostgres{}, nil
}

func (p *TestPostgres) Start(ctx context.Context) error {
	var err error
	p.pool, err = dockertest.NewPool("")
	if err != nil {
		return fmt.Errorf("failed to create docker pool: %w", err)
	}

	var (
		password = "test"
		dbName   = "test"
		user     = "test"
	)

	p.container, err = p.pool.Run("postgres", "latest", []string{
		fmt.Sprintf("POSTGRES_PASSWORD=%s", password),
		fmt.Sprintf("POSTGRES_DB=%s", dbName),
		fmt.Sprintf("POSTGRES_USER=%s", user),
	})
	if err != nil {
		return fmt.Errorf("failed to start postgres container: %w", err)
	}

	err = p.pool.Retry(func() error {
		db, err := postgresdb.NewDirectConnection(ctx, p.ConnectionConfig(), slog.Default())
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		return db.Ping()
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	return nil
}

func (p *TestPostgres) Stop() error {
	err := p.pool.Purge(p.container)
	if err != nil {
		return fmt.Errorf("failed to purge postgres container: %w", err)
	}
	return nil
}

func (p *TestPostgres) ConnectionConfig() postgresdb.DirectConfig {
	return postgresdb.DirectConfig{
		Hostname: "localhost",
		Port:     p.container.GetPort("5432/tcp"),
		UserName: "test",
		Password: "test",
		Database: "test",
	}
}
