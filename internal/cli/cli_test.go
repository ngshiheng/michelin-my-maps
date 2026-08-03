package cli

import (
	"context"
	"testing"
)

func TestRunNoArgsShowsUsageWithoutError(t *testing.T) {
	t.Parallel()

	err := Run(context.Background(), []string{})
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
}

func TestRunUnknownCommandReturnsError(t *testing.T) {
	t.Parallel()

	err := Run(context.Background(), []string{"unknown"})
	if err == nil {
		t.Fatal("Run() error = nil, want non-nil")
	}
}

func TestRunVersionCommand(t *testing.T) {
	t.Parallel()

	err := Run(context.Background(), []string{"version"})
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
}
