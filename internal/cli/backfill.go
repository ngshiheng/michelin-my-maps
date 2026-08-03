package cli

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ngshiheng/michelin-my-maps/v4/internal/backfill"
)

func runBackfill(ctx context.Context, args []string) error {
	cmd := newFlagSet("backfill")

	opts, helpShown, err := parseCommandOptions(cmd, "skip using wayback cache", args)
	if err != nil {
		return err
	}
	if helpShown {
		return nil
	}
	if cmd.NArg() > 1 {
		return fmt.Errorf("backfill accepts at most one URL argument")
	}

	app, err := backfill.New(opts.ignoreCache)
	if err != nil {
		return fmt.Errorf("failed to create backfill scraper: %w", err)
	}

	urlArg := cmd.Arg(0)
	slog.Info("running backfill command")
	if urlArg != "" {
		return app.Run(ctx, urlArg)
	}
	return app.RunAll(ctx)
}
