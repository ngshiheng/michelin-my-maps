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

	go func() {
		<-ctx.Done()
		stop()
	}()

	if err := cli.Run(ctx, os.Args[1:]); err != nil {
		slog.Error("command failed", "error", err)
		os.Exit(1)
	}
}
