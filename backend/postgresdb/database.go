package postgresdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func NewDirectConnection(ctx context.Context, cfg DirectConfig, logger *slog.Logger) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

type DirectConfig struct {
	Hostname string
	Port     string
	UserName string
	Password string
	Database string
}

// DSN returns the data source name for the connection.
// Ex. "postgres://username:password@localhost:5432/database_name"
func (c DirectConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s", c.UserName, c.Password, c.Hostname, c.Port, c.Database)

}
