package cli

import (
	"testing"
)

func TestDefaultLogLevelUsesEnv(t *testing.T) {
	t.Setenv(envLogLevel, "debug")

	got := defaultLogLevel()
	if got != "debug" {
		t.Fatalf("defaultLogLevel() = %q, want %q", got, "debug")
	}
}

func TestDefaultLogLevelFallsBackToInfo(t *testing.T) {
	t.Setenv(envLogLevel, "")

	got := defaultLogLevel()
	if got != defaultLogLevelName {
		t.Fatalf("defaultLogLevel() = %q, want %q", got, defaultLogLevelName)
	}
}

func TestDefaultLogFormatUsesEnv(t *testing.T) {
	t.Setenv(envLogFormat, "json")

	got := defaultLogFormat()
	if got != "json" {
		t.Fatalf("defaultLogFormat() = %q, want %q", got, "json")
	}
}

func TestDefaultLogFormatFallsBackToText(t *testing.T) {
	t.Setenv(envLogFormat, "")

	got := defaultLogFormat()
	if got != defaultLogFormatName {
		t.Fatalf("defaultLogFormat() = %q, want %q", got, defaultLogFormatName)
	}
}

func TestEnvOrDefault(t *testing.T) {
	t.Setenv("MYM_TEST_KEY", " custom ")
	got := envOrDefault("MYM_TEST_KEY", "fallback")
	if got != "custom" {
		t.Fatalf("envOrDefault() = %q, want %q", got, "custom")
	}

	t.Setenv("MYM_TEST_KEY", "")
	got = envOrDefault("MYM_TEST_KEY", "fallback")
	if got != "fallback" {
		t.Fatalf("envOrDefault() = %q, want %q", got, "fallback")
	}
}

func TestParseCommandFlagsReturnsHelpShown(t *testing.T) {
	fs := newFlagSet("test")
	fs.Bool("verbose", false, "enable verbose output")

	helpShown, err := parseCommandFlags(fs, []string{"-h"})
	if err != nil {
		t.Fatalf("parseCommandFlags() unexpected error: %v", err)
	}
	if !helpShown {
		t.Fatal("parseCommandFlags() helpShown = false, want true")
	}
}

func TestParseCommandFlagsReturnsErrorForUnknownFlag(t *testing.T) {
	fs := newFlagSet("test")
	fs.Bool("verbose", false, "enable verbose output")

	helpShown, err := parseCommandFlags(fs, []string{"--unknown"})
	if err == nil {
		t.Fatal("parseCommandFlags() error = nil, want non-nil")
	}
	if helpShown {
		t.Fatal("parseCommandFlags() helpShown = true, want false")
	}
}

func TestParseCommandFlagsParsesKnownFlag(t *testing.T) {
	fs := newFlagSet("test")
	verbose := fs.Bool("verbose", false, "enable verbose output")

	helpShown, err := parseCommandFlags(fs, []string{"--verbose"})
	if err != nil {
		t.Fatalf("parseCommandFlags() unexpected error: %v", err)
	}
	if helpShown {
		t.Fatal("parseCommandFlags() helpShown = true, want false")
	}
	if !*verbose {
		t.Fatal("verbose flag = false, want true")
	}
}

func TestParseLogFormatAcceptsTextAndJSON(t *testing.T) {
	t.Run("text", func(t *testing.T) {
		got, err := parseLogFormat("text")
		if err != nil {
			t.Fatalf("parseLogFormat() unexpected error: %v", err)
		}
		if got != "text" {
			t.Fatalf("parseLogFormat() = %q, want %q", got, "text")
		}
	})

	t.Run("json upper-case", func(t *testing.T) {
		got, err := parseLogFormat("JSON")
		if err != nil {
			t.Fatalf("parseLogFormat() unexpected error: %v", err)
		}
		if got != "json" {
			t.Fatalf("parseLogFormat() = %q, want %q", got, "json")
		}
	})
}

func TestParseLogFormatRejectsInvalid(t *testing.T) {
	_, err := parseLogFormat("pretty")
	if err == nil {
		t.Fatal("parseLogFormat() error = nil, want non-nil")
	}
}