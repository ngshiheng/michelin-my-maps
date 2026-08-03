package cli

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ngshiheng/michelin-my-maps/v4/internal/scraper"
)

func runScrape(ctx context.Context, args []string) error {
	cmd := newFlagSet("scrape")

	opts, helpShown, err := parseCommandOptions(cmd, "skip using scrape cache", args)
	if err != nil {
		return err
	}
	if helpShown {
		return nil
	}
	if cmd.NArg() > 1 {
		return fmt.Errorf("scrape accepts at most one URL argument")
	}

	app, err := scraper.New(opts.ignoreCache)
	if err != nil {
		return fmt.Errorf("failed to create live scraper: %w", err)
	}

	urlArg := cmd.Arg(0)
	slog.Info("running scrape command")
	if urlArg != "" {
		return app.Run(ctx, urlArg)
	}
	return app.RunAll(ctx)
}
