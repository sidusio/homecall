package api

import (
	"context"
	"errors"
	"fmt"
	"github.com/kelseyhightower/envconfig"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sidus.io/home-call/app"
	"sidus.io/home-call/gen/connect/homecall/v1alpha/homecallv1alphaconnect"
	"time"
)

type TestApp struct {
	port     string
	config   TestAppConfig
	runError chan error
	cancel   context.CancelFunc
	pg       *TestPostgres
}

type TestAppConfig struct {
	NotificationDir string
}

type TestAppOption func(config *TestAppConfig)

func WithNotificationDir(dir string) TestAppOption {
	return func(config *TestAppConfig) {
		config.NotificationDir = dir
	}
}

func NewTestApp(options ...TestAppOption) (*TestApp, error) {
	config := TestAppConfig{}
	for _, option := range options {
		option(&config)
	}

	return &TestApp{
		config:   config,
		runError: make(chan error),
	}, nil
}

func (a *TestApp) ApiAddress() string {
	return fmt.Sprintf("http://localhost:%s", a.port)
}

func (a *TestApp) OfficeClient() homecallv1alphaconnect.OfficeServiceClient {
	return homecallv1alphaconnect.NewOfficeServiceClient(http.DefaultClient, a.ApiAddress())
}

func (a *TestApp) DeviceClient() homecallv1alphaconnect.DeviceServiceClient {
	return homecallv1alphaconnect.NewDeviceServiceClient(http.DefaultClient, a.ApiAddress())

}

func (a *TestApp) TenantClient() homecallv1alphaconnect.TenantServiceClient {
	return homecallv1alphaconnect.NewTenantServiceClient(http.DefaultClient, a.ApiAddress())
}

func (a *TestApp) NotificationsDir() string {
	return a.config.NotificationDir
}

func (a *TestApp) Start(ctx context.Context) error {
	ctx, a.cancel = context.WithCancel(ctx)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	pg, err := NewTestPostgres()
	if err != nil {
		return fmt.Errorf("failed to create test postgres: %w", err)
	}
	err = pg.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start test postgres: %w", err)
	}
	a.pg = pg
	dbConfig := pg.ConnectionConfig()

	port, err := getNextAvailablePort()
	if err != nil {
		return fmt.Errorf("failed to get next available port: %w", err)
	}

	var cfg app.Config
	err = envconfig.Process("TEST_HOMECALL", &cfg)
	if err != nil && err.Error() != "required key DB_PASSWORD missing value" {
		return fmt.Errorf("failed to process config: %w", err)
	}
	cfg.DBHost = dbConfig.Hostname
	cfg.DBPort = dbConfig.Port
	cfg.DBUser = dbConfig.UserName
	cfg.DBPassword = dbConfig.Password
	cfg.DBName = dbConfig.Database
	cfg.Port = port
	cfg.AuthDisabled = true
	cfg.JitsiKeyRaw = dummyPemKey
	cfg.MockNotificationsDir = a.config.NotificationDir

	a.port = port

	go func() {
		err = app.Run(ctx, logger, cfg)
		a.cancel()
		if err != nil {
			a.runError <- fmt.Errorf("failed to run app: %w", err)
		}
		close(a.runError)
	}()

	err = waitForAddress(ctx, fmt.Sprintf("localhost:%s", port))
	if err != nil {
		return fmt.Errorf("failed to wait for address: %w", err)
	}
	return nil
}

func (a *TestApp) Stop() error {
	a.cancel()
	pgErr := a.pg.Stop()
	return errors.Join(pgErr, <-a.runError)
}

func waitForAddress(ctx context.Context, address string) error {
	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			conn, _ := net.Dial("tcp", address)
			if conn != nil {
				defer conn.Close()
				return
			}
			time.Sleep(time.Second / 10)
		}
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("context done")
	case <-time.After(time.Second * 10):
		return fmt.Errorf("timeout")
	case <-done:
		return nil
	}
}
