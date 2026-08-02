package cli

import (
	"log/slog"
	"testing"
)

func TestParseLogLevelAcceptsAliases(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  slog.Level
	}{
		{name: "debug", input: "debug", want: slog.LevelDebug},
		{name: "info", input: "info", want: slog.LevelInfo},
		{name: "warn alias", input: "warning", want: slog.LevelWarn},
		{name: "error", input: "error", want: slog.LevelError},
		{name: "fatal maps to error", input: "fatal", want: slog.LevelError},
		{name: "panic maps to error", input: "panic", want: slog.LevelError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseLogLevel(tc.input)
			if err != nil {
				t.Fatalf("parseLogLevel() unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("parseLogLevel() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestParseLogLevelRejectsInvalid(t *testing.T) {
	_, err := parseLogLevel("trace")
	if err == nil {
		t.Fatal("parseLogLevel() error = nil, want non-nil")
	}
}
