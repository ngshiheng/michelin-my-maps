package cli

import (
	"context"
	"fmt"
	"os"
)

var commands = map[string]func(ctx context.Context, args []string) error{
	"version":  runVersion,
	"scrape":   runScrape,
	"session":  runSession,
	"backfill": runBackfill,
}

const usageText = `usage:
	%s version          show version
	%s scrape [url]     scrape latest data
	%s backfill [url]   backfill wayback data
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
