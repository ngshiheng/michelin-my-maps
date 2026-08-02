package cli

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ngshiheng/michelin-my-maps/v4/internal/scraper"
	"github.com/ngshiheng/michelin-my-maps/v4/internal/session"
)

const defaultBrowserTimeout = 60 * time.Second

func runSession(ctx context.Context, args []string) error {
	cmd := newFlagSet("session")
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
	if cmd.NArg() > 0 {
		return fmt.Errorf("session does not accept positional arguments")
	}
	if err := setupLogging(*logLevel, *logFormat); err != nil {
		return err
	}

	slog.Info("running session command")
	cookies, err := session.GetCookies(ctx, defaultBrowserTimeout)
	if err != nil {
		return err
	}

	app, err := scraper.New(*ignoreCache)
	if err != nil {
		return fmt.Errorf("failed to create scraper: %w", err)
	}
	if err := app.InitCookies(cookies); err != nil {
		return fmt.Errorf("failed to persist session cookies: %w", err)
	}

	slog.Info("session stored", "cookie_count", len(cookies))
	slog.Info("session command completed")
	return nil
}
