package cli

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/lmittmann/tint"
	"golang.org/x/term"
)

func setupLogging(levelStr, formatStr string) error {
	level, err := parseLogLevel(levelStr)
	if err != nil {
		return fmt.Errorf("invalid log level %q: %w", levelStr, err)
	}

	format, err := parseLogFormat(formatStr)
	if err != nil {
		return err
	}

	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	switch format {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	default:
		handler = tint.NewTextHandler(os.Stdout, &tint.Options{
			Level:   level,
			NoColor: !term.IsTerminal(int(os.Stdout.Fd())),
		})
	}

	slog.SetDefault(slog.New(handler))
	return nil
}

func parseLogLevel(value string) (slog.Level, error) {
	level := strings.ToLower(strings.TrimSpace(value))
	switch level {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error", "fatal", "panic":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("expected debug, info, warning, error, fatal, or panic")
	}
}
