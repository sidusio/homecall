package main

import (
	"context"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kelseyhightower/envconfig"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sidus.io/home-call/gen/connect/homecall/v1alpha/homecallv1alphaconnect"
	"sidus.io/home-call/jitsi"
	"sidus.io/home-call/messaging"
	"sidus.io/home-call/migrations"
	"sidus.io/home-call/postgresdb"
	"sidus.io/home-call/services/deviceapi"
	"sidus.io/home-call/services/officeapi"
	"sidus.io/home-call/util"
	"time"
)

const (
	appName = "homecall"
)

type Config struct {
	DBHost     string `envconfig:"DB_HOST" default:"localhost"`
	DBPort     string `envconfig:"DB_PORT" default:"5432"`
	DBUser     string `envconfig:"DB_USER" default:"homecall"`
	DBPassword string `envconfig:"DB_PASSWORD" required:"true"`
	DBName     string `envconfig:"DB_NAME" default:"homecall"`

	Port string `envconfig:"PORT" default:"8080"`

	JitsiAppId   string `envconfig:"JITSI_APP_ID" required:"true"`
	JitsiKeyId   string `envconfig:"JITSI_KEY_ID" required:"true"`
	JitsiKeyFile string `envconfig:"JITSI_KEY_FILE" required:"true"`
}

func main() {
	ctx, cleanup := signal.NotifyContext(context.Background(), os.Interrupt)
	go func(ctx context.Context, cleanup context.CancelFunc) {
		<-ctx.Done()
		cleanup()
	}(ctx, cleanup)

	var cfg Config
	err := envconfig.Process(appName, &cfg)
	if err != nil {
		slog.Error("failed to process env vars", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	err = run(ctx, logger, cfg)
	if err != nil {
		logger.ErrorContext(ctx, "failed to run", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *slog.Logger, cfg Config) error {
	// Database
	db, err := postgresdb.NewDirectConnection(ctx, postgresdb.DirectConfig{
		Hostname: cfg.DBHost,
		Port:     cfg.DBPort,
		UserName: cfg.DBUser,
		Password: cfg.DBPassword,
		Database: cfg.DBName,
	}, logger)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() {
		err := db.Close()
		if err != nil {
			logger.Error("failed to close database", "error", err)
		}
	}()
	logger.Info("connected to database")

	// Migrations
	logger.Info("running migrations, might take a while...")
	migrator, err := postgresdb.NewMigrator(ctx, db, logger.With("component", "migrations"), postgresdb.MigrationConfig{
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
	logger.Info("migrations done")

	// Jitsi
	jitsiApp, err := setupJitsiApp(cfg)
	if err != nil {
		return fmt.Errorf("failed to setup jitsi app: %w", err)
	}
	logger.Info("jitsi app configured")

	// Messaging
	broker, err := messaging.NewBroker(logger)
	if err != nil {
		return fmt.Errorf("failed to create broker: %w", err)
	}
	defer func() {
		err := broker.Close()
		if err != nil {
			logger.Error("failed to close broker", "error", err)
		}
	}()
	logger.Info("message broker created")

	// Service layer
	deviceService := deviceapi.NewService(db, broker)
	officeService := officeapi.NewOfficeService(db, broker, jitsiApp)
	logger.Info("service layer created")

	// Error group for application main lifecycle
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		logger.Info("starting message broker")
		err := broker.Run(ctx)
		if err != nil {
			return fmt.Errorf("broker exited: %w", err)
		}
		return nil
	})
	eg.Go(func() error {
		logger.Info("listening on port", "port", cfg.Port)
		err := util.ListenAndServe(
			ctx,
			setupHttpServer(logger, cfg, deviceService, officeService),
			15*time.Second,
		)
		if err != nil {
			return fmt.Errorf("failed to listen and serve: %w", err)
		}
		return nil
	})

	err = eg.Wait()
	if err != nil {
		return fmt.Errorf("application crashed: %w", err)
	}
	return nil
}

func setupJitsiApp(cfg Config) (*jitsi.App, error) {
	jitsiKeyData, err := os.ReadFile(cfg.JitsiKeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read jitsi key file: %w", err)
	}
	jitsiKey, err := jwt.ParseRSAPrivateKeyFromPEM(jitsiKeyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse jitsi key: %w", err)
	}
	return jitsi.NewApp(cfg.JitsiAppId, cfg.JitsiKeyId, jitsiKey), nil
}

func setupHttpServer(
	logger *slog.Logger,
	cfg Config,
	deviceService *deviceapi.Service,
	officeService *officeapi.OfficeService,
) *http.Server {
	mux := http.NewServeMux()
	mux.Handle(homecallv1alphaconnect.NewDeviceServiceHandler(deviceService))
	mux.Handle(homecallv1alphaconnect.NewOfficeServiceHandler(officeService))

	server := &http.Server{
		Handler: h2c.NewHandler(mux, &http2.Server{}),
		Addr:    fmt.Sprintf(":%s", cfg.Port),
	}
	return server
}
