package main

import (
	"context"
	"crypto/sha1"
	"fmt"
	"github.com/go-jet/jet/v2/generator/postgres"
	"github.com/ory/dockertest/v3"
	"io/fs"
	"log/slog"
	"os"
	"os/signal"
	"sidus.io/home-call/migrations"
	"sidus.io/home-call/postgresdb"
	"strconv"
)

const hashPath = "./gen/jetdb/.migrations_hash"

func main() {
	ctx, cleanup := signal.NotifyContext(context.Background(), os.Interrupt)
	go func(ctx context.Context, cleanup context.CancelFunc) {
		<-ctx.Done()
		cleanup()
	}(ctx, cleanup)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	err := run(ctx, logger)
	if err != nil {
		logger.ErrorContext(ctx, "failed to run", "error", err)
		os.Exit(1)
	}

	os.Exit(0)

}

func run(ctx context.Context, logger *slog.Logger) error {
	logger.Info("Checking if migrations have changed")
	_, err := os.Stat(hashPath)
	if err == nil {
		prevHash, err := os.ReadFile(hashPath)
		if err != nil {
			return fmt.Errorf("failed to read previous hash: %w", err)
		}

		currentHash := migrationsHash()
		if string(prevHash) == currentHash {
			logger.Warn("MigrationsFs have not changed, skipping generation")
			return nil
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check previous hash: %w", err)
	}
	logger.Info("MigrationsFs have changed, regenerating")

	logger.Info("Cleaning up old generated code", "path", "./gen/jetdb")
	err = os.RemoveAll("./gen/jetdb")
	if err != nil {
		return fmt.Errorf("failed to remove old generated code: %w", err)
	}

	pool, err := dockertest.NewPool("")
	if err != nil {
		return fmt.Errorf("failed to create docker pool: %w", err)
	}

	var (
		password = "jet"
		dbName   = "jetdb"
		user     = "jet"
	)

	resource, err := pool.Run("postgres", "latest", []string{
		fmt.Sprintf("POSTGRES_PASSWORD=%s", password),
		fmt.Sprintf("POSTGRES_DB=%s", dbName),
		fmt.Sprintf("POSTGRES_USER=%s", user),
	})
	if err != nil {
		return fmt.Errorf("failed to start postgres container: %w", err)
	}
	defer func() {
		err := pool.Purge(resource)
		if err != nil {
			logger.Error("failed to purge container", "error", err)
		}
	}()

	port := resource.GetPort("5432/tcp")
	logger.Info("Postgres container started", "port", port, "password", password, "dbName", dbName, "user", user)

	// Wait for postgres to start.
	err = pool.Retry(func() error {
		db, err := postgresdb.NewDirectConnection(ctx, postgresdb.DirectConfig{
			Hostname: "localhost",
			Port:     port,
			UserName: user,
			Password: password,
			Database: dbName,
		}, logger)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		return db.Ping()
	})
	if err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Database connection ready")

	db, err := postgresdb.NewDirectConnection(ctx, postgresdb.DirectConfig{
		Hostname: "localhost",
		Port:     port,
		UserName: user,
		Password: password,
		Database: dbName,
	}, logger)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	logger.Info("Migrating database")
	migrator, err := postgresdb.NewMigrator(ctx, db, logger, postgresdb.MigrationConfig{
		ApplyMigrations: true,
		MigrationsFS:    migrations.Migrations,
		MigrationsPath:  ".",
	})
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}

	err = migrator.Run(ctx)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	portNumber, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("failed to convert port to number: %w", err)
	}

	schemas := []string{"internal_data", "internal_state"}
	for _, schema := range schemas {
		logger.Info("Generating code", "schema", schema)
		err = postgres.Generate("./gen",
			postgres.DBConnection{
				Host:       "localhost",
				Port:       portNumber,
				User:       user,
				Password:   password,
				DBName:     dbName,
				SchemaName: schema,
				SslMode:    "disable",
			})
		if err != nil {
			return fmt.Errorf("failed to generate: %w", err)
		}
	}

	err = os.WriteFile(hashPath, []byte(migrationsHash()), 0644)
	if err != nil {
		return fmt.Errorf("failed to write hash: %w", err)
	}

	return nil
}

func migrationsHash() string {
	hash := sha1.New()
	err := fs.WalkDir(migrations.Migrations, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("failed to walk dir: %w", err)
		}

		hash.Write([]byte(path))

		if d.IsDir() {
			return nil
		}

		file, err := fs.ReadFile(migrations.Migrations, path)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		hash.Write(file)
		return nil
	})
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}
