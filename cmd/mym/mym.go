package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ngshiheng/michelin-my-maps/v4/internal/cli"
)

func main() {
	if err := os.Setenv("TZ", time.UTC.String()); err != nil {
		slog.Warn("failed to set TZ", "error", err)
	}
	time.Local = time.UTC

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// NOTE: first signal triggers graceful shutdown via context cancellation
	// second signal exits immediately so users are not stuck waiting
	forceExitSignals := make(chan os.Signal, 1)

	signal.Notify(forceExitSignals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(forceExitSignals)

	go func() {
		<-ctx.Done()
		<-forceExitSignals
		slog.Error("received second interrupt, forcing exit")
		os.Exit(130)
	}()

	if err := cli.Run(ctx, os.Args[1:]); err != nil {
		slog.Error("command failed", "error", err)
		os.Exit(1)
	}
}
