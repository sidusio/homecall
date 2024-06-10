package main

import (
	"context"
	"github.com/kelseyhightower/envconfig"
	"log/slog"
	"os"
	"os/signal"
	"sidus.io/home-call/app"
)

const (
	appName = "homecall"
)

func main() {
	ctx, cleanup := signal.NotifyContext(context.Background(), os.Interrupt)
	go func(ctx context.Context, cleanup context.CancelFunc) {
		<-ctx.Done()
		cleanup()
	}(ctx, cleanup)

	var cfg app.Config
	err := envconfig.Process(appName, &cfg)
	if err != nil {
		slog.Error("failed to process env vars", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	err = app.Run(ctx, logger, cfg)
	if err != nil {
		logger.ErrorContext(ctx, "failed to run", "error", err)
		os.Exit(1)
	}
	logger.Debug("shutdown successful")
}
