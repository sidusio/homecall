package app

import (
	"connectrpc.com/connect"
	"context"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"sidus.io/home-call/gen/connect/homecall/v1alpha/homecallv1alphaconnect"
	"sidus.io/home-call/jitsi"
	"sidus.io/home-call/messaging"
	"sidus.io/home-call/migrations"
	"sidus.io/home-call/notifications"
	"sidus.io/home-call/notifications/directorynotifications"
	"sidus.io/home-call/notifications/firebasenotifications"
	"sidus.io/home-call/notifications/lognotifications"
	"sidus.io/home-call/postgresdb"
	"sidus.io/home-call/services/auth"
	"sidus.io/home-call/services/deviceapi"
	"sidus.io/home-call/services/officeapi"
	"sidus.io/home-call/services/tenantapi"
	"sidus.io/home-call/util"
	"time"
)

func Run(ctx context.Context, logger *slog.Logger, cfg Config) error {
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

	// Notifications
	notificationService, err := setupNotificationService(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to setup notification service: %w", err)
	}

	// Service layer
	tenantService := tenantapi.NewService(db, logger.With("component", "tenantapi"), 2)
	deviceService := deviceapi.NewService(db, broker, logger.With("component", "deviceapi"))
	officeService := officeapi.NewService(db, broker, jitsiApp, logger.With("component", "officeapi"), tenantService, notificationService)
	logger.Info("service layer created")

	// Auth interceptor
	authIssuerUrl, err := url.Parse(cfg.AuthIssuer)
	if err != nil {
		return fmt.Errorf("failed to parse auth issuer url: %w", err)
	}
	authInterceptor, err := auth.NewAuthInterceptor(authIssuerUrl, cfg.AuthAudience, cfg.AuthDisabled, db)
	if err != nil {
		return fmt.Errorf("failed to create auth interceptor: %w", err)
	}

	// Http server
	httpServer, err := setupHttpServer(logger, cfg, deviceService, officeService, tenantService, authInterceptor)
	if err != nil {
		return fmt.Errorf("failed to setup http server: %w", err)
	}

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
	<-broker.Started()
	logger.Debug("broker started")

	eg.Go(func() error {
		logger.Info("listening on port", "port", cfg.Port)
		err := util.ListenAndServe(
			ctx,
			httpServer,
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

func setupNotificationService(cfg Config, logger *slog.Logger) (notifications.Service, error) {
	switch {
	case cfg.MockNotificationsDir != "":
		logger.Info("using mock notifications service", "directory", cfg.MockNotificationsDir)
		service, err := directorynotifications.NewService(cfg.MockNotificationsDir)
		if err != nil {
			return nil, fmt.Errorf("failed to create mock notifications service: %w", err)
		}
		return service, nil
	case cfg.FirebaseProjectId != "":
		logger.Info("using firebase notifications service", "project_id", cfg.FirebaseProjectId)
		service, err := firebasenotifications.NewService(context.Background(), cfg.FirebaseProjectId)
		if err != nil {
			return nil, fmt.Errorf("failed to create firebase notifications service: %w", err)
		}
		return service, nil
	default:
		logger.Warn("no notification service configured, notifications will be logged")
		return lognotifications.NewService(logger), nil
	}
}

func setupJitsiApp(cfg Config) (*jitsi.App, error) {
	jitsiKeyData := []byte(cfg.JitsiKeyRaw)
	if len(jitsiKeyData) == 0 {
		var err error
		jitsiKeyData, err = os.ReadFile(cfg.JitsiKeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read jitsi key file: %w", err)
		}
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
	officeService *officeapi.Service,
	tenantService *tenantapi.Service,
	authInterceptor *auth.AuthInterceptor,
) (*http.Server, error) {
	deviceInterceptors := []connect.Interceptor{
		newContextInterceptor(),
	}
	officeInterceptors := []connect.Interceptor{
		newContextInterceptor(),
		authInterceptor,
	}

	mux := http.NewServeMux()
	mux.Handle(homecallv1alphaconnect.NewDeviceServiceHandler(
		deviceService,
		connect.WithInterceptors(deviceInterceptors...),
	))
	mux.Handle(homecallv1alphaconnect.NewOfficeServiceHandler(
		officeService,
		connect.WithInterceptors(officeInterceptors...),
	))
	mux.Handle(homecallv1alphaconnect.NewTenantServiceHandler(
		tenantService,
		connect.WithInterceptors(officeInterceptors...),
	))

	server := &http.Server{
		Handler: http.MaxBytesHandler(h2c.NewHandler(mux, &http2.Server{}), 1<<20 /* 1mb */),
		Addr:    fmt.Sprintf(":%s", cfg.Port),
	}
	return server, nil
}

type contextInterceptor struct{}

func newContextInterceptor() *contextInterceptor {
	return &contextInterceptor{}
}

func (i *contextInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	// Same as previous UnaryInterceptorFunc.
	return func(
		ctx context.Context,
		req connect.AnyRequest,
	) (connect.AnyResponse, error) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		return next(ctx, req)
	}
}

func (*contextInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(
		ctx context.Context,
		spec connect.Spec,
	) connect.StreamingClientConn {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		return next(ctx, spec)
	}
}

func (i *contextInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(
		ctx context.Context,
		conn connect.StreamingHandlerConn,
	) error {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		return next(ctx, conn)
	}
}
