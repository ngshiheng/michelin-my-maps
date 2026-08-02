package cli

import (
	"context"
	"fmt"
	"os"
)

var commands = map[string]func(ctx context.Context, args []string) error{
	"scrape":   runScrape,
	"session":  runSession,
	"backfill": runBackfill,
	"version":  runVersion,
}

const usageText = `usage:
	%s scrape [url]     scrape latest data
	%s backfill [url]   backfill from wayback
	%s version          show version

common flags:
	--log-level <level> debug|info|warning|error
	-log-format <fmt>   text|json
`

func Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		usage()
		return nil
	}

	cmd, rest := args[0], args[1:]

	if cmd == "-h" || cmd == "--help" {
		usage()
		return nil
	}

	handler, ok := commands[cmd]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown command: %q\n\n", cmd)
		usage()
		return fmt.Errorf("unknown command %q", cmd)
	}
	return handler(ctx, rest)
}

func usage() {
	fmt.Printf(usageText, os.Args[0], os.Args[0], os.Args[0])
}
