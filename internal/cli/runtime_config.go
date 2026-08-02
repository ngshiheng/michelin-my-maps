package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
)

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

func parseLogFormat(value string) (string, error) {
	format := strings.ToLower(strings.TrimSpace(value))
	switch format {
	case "text", "json":
		return format, nil
	default:
		return "", fmt.Errorf("invalid log format %q: expected text or json", value)
	}
}
