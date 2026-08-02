package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
)

type commandOptions struct {
	ignoreCache bool
	logFormat   string
	logLevel    string
}

const (
	envLogLevel          = "MYM_LOG_LEVEL"
	envLogFormat         = "MYM_LOG_FORMAT"
	defaultLogLevelName  = "info"
	defaultLogFormatName = "text"
)

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	return fs
}

func defaultLogLevel() string {
	return envOrDefault(envLogLevel, defaultLogLevelName)
}

func defaultLogFormat() string {
	return envOrDefault(envLogFormat, defaultLogFormatName)
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func parseCommandFlags(fs *flag.FlagSet, args []string) (helpShown bool, err error) {
	err = fs.Parse(args)
	if err == nil {
		return false, nil
	}
	if errors.Is(err, flag.ErrHelp) {
		return true, nil
	}
	return false, err
}

func parseCommandOptions(cmd *flag.FlagSet, cacheHelp string, args []string) (*commandOptions, bool, error) {
	logLevel := cmd.String("log-level", defaultLogLevel(), "log level (debug, info, warning, error, fatal, panic)")
	logFormat := cmd.String("log-format", defaultLogFormat(), "log format (text or json)")
	ignoreCache := cmd.Bool("no-cache", false, cacheHelp)

	helpShown, err := parseCommandFlags(cmd, args)
	if err != nil {
		return nil, false, err
	}
	if helpShown {
		return nil, true, nil
	}

	if err := setupLogging(*logLevel, *logFormat); err != nil {
		return nil, false, err
	}

	return &commandOptions{
		ignoreCache: *ignoreCache,
		logFormat:   *logFormat,
		logLevel:    *logLevel,
	}, false, nil
}

func parseLogFormat(value string) (string, error) {
	format := strings.ToLower(strings.TrimSpace(value))
	switch format {
	case "text", "json":
		return format, nil
	default:
		return "", fmt.Errorf("invalid log format %q: expected text or json", value)
	}
}
