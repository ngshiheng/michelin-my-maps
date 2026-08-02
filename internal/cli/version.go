package cli

import (
	"context"
	"fmt"
	"runtime/debug"
)

func runVersion(_ context.Context, _ []string) error {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		fmt.Println("unable to determine build information")
		return nil
	}

	version := "development"
	if buildInfo.Main.Version != "" {
		version = buildInfo.Main.Version
	}

	fmt.Printf("version: %s\n", version)
	return nil
}
