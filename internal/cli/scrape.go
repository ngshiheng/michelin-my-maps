package cli

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ngshiheng/michelin-my-maps/v4/internal/scraper"
)

func runScrape(ctx context.Context, args []string) error {
	cmd := newFlagSet("scrape")
	logLevel := cmd.String("log-level", defaultLogLevel(), "log level (debug, info, warning, error, fatal, panic)")
	logFormat := cmd.String("log-format", defaultLogFormat(), "log format (text or json)")
	ignoreCache := cmd.Bool("no-cache", false, "skip using scrape cache")

	helpShown, err := parseCommandFlags(cmd, args)
	if err != nil {
		return err
	}
	if helpShown {
		return nil
	}
	if cmd.NArg() > 1 {
		return fmt.Errorf("scrape accepts at most one URL argument")
	}
	if err := setupLogging(*logLevel, *logFormat); err != nil {
		return err
	}

	app, err := scraper.New(*ignoreCache)
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
