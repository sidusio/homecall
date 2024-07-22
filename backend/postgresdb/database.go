package postgresdb

import (
	"cloud.google.com/go/cloudsqlconn"
	"cloud.google.com/go/cloudsqlconn/postgres/pgxv4"
	"cloud.google.com/go/compute/metadata"
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
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
	Role     string
}

// DSN returns the data source name for the connection.
// Ex. "postgres://username:password@localhost:5432/database_name"
func (c DirectConfig) DSN() string {
	var options []string
	if c.Role != "" {
		options = append(options, "role="+c.Role)
	}

	optionsString := ""
	if len(options) > 0 {
		optionsString = "?" + strings.Join(options, "&")
	}

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s%s", c.UserName, c.Password, c.Hostname, c.Port, c.Database, optionsString)

}

func NewCloudSQLConnection(ctx context.Context, cfg CloudSQLConfig, logger *slog.Logger) (*sql.DB, error) {
	err := cfg.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	if cfg.AutoDetectUserName {
		serviceAccount, err := metadata.Email("default")
		if err != nil {
			panic(err)
		}
		cfg.UserName = strings.TrimSuffix(serviceAccount, ".gserviceaccount.com")
	}

	driverName := fmt.Sprintf("cloudsql-%s", randomString(10))

	cleanup, err := pgxv4.RegisterDriver(driverName, cfg.CloudSQLOptions()...)
	if err != nil {
		return nil, fmt.Errorf("failed to register driver: %w", err)
	}

	// Cleanup the driver when the context is done.
	go func(ctx context.Context, cleanup func() error) {
		<-ctx.Done()
		err := cleanup()
		if err != nil {
			logger.ErrorContext(ctx, "failed to cleanup driver", "error", err)
		}
	}(ctx, cleanup)

	db, err := sql.Open(driverName, cfg.DSN())
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

type CloudSQLConfig struct {
	InstanceName       string
	AutoDetectUserName bool
	UserName           string
	DatabaseName       string
	UsePrivateIP       bool
	Role               string
}

func (c CloudSQLConfig) CloudSQLOptions() []cloudsqlconn.Option {
	options := []cloudsqlconn.Option{
		cloudsqlconn.WithIAMAuthN(),
	}
	if c.UsePrivateIP {
		options = append(options, cloudsqlconn.WithDefaultDialOptions(cloudsqlconn.WithPrivateIP()))
	}
	return options
}

func (c CloudSQLConfig) DSN() string {
	options := []string{
		"host=" + c.InstanceName,
		"user=" + c.UserName,
		"sslmode=disable",
		"database=" + c.DatabaseName,
	}
	if c.Role != "" {
		options = append(options, "role="+c.Role)
	}
	return strings.Join(options, " ")
}

func (c CloudSQLConfig) Validate() error {
	if c.InstanceName == "" {
		return fmt.Errorf("instance name is required")
	}
	if c.DatabaseName == "" {
		return fmt.Errorf("database name is required")
	}
	if c.AutoDetectUserName && c.UserName != "" {
		return fmt.Errorf("cannot use AutoDetectUserName and provide a UserName")
	}
	if !c.AutoDetectUserName && c.UserName == "" {
		return fmt.Errorf("must provide a UserName or set AutoDetectUserName to true")
	}
	return nil
}

func randomString(length int) string {
	randomChars := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	randomName := make([]rune, length)
	for i := range randomName {
		randomName[i] = randomChars[rand.Intn(len(randomChars))]
	}
	return string(randomName)
}
