package main

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestSetupLogger(t *testing.T) {
	tests := []struct {
		name     string
		verbose  bool
		expected slog.Level
	}{
		{
			name:     "verbose mode should set debug level",
			verbose:  true,
			expected: slog.LevelDebug,
		},
		{
			name:     "non-verbose mode should set info level",
			verbose:  false,
			expected: slog.LevelInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verbose = tt.verbose
			logger := setupLogger()

			if logger == nil {
				t.Fatal("expected logger to be non-nil")
			}

			if !logger.Enabled(context.TODO(), tt.expected) {
				t.Errorf("expected logger to be enabled at level %v", tt.expected)
			}
		})
	}
}

func TestRunCheck_InvalidURL(t *testing.T) {
	urls = []string{":/invalid"}
	verbose = false
	validityDaysWarning = 90

	cmd := &cobra.Command{}
	err := runCheck(cmd, []string{})

	if err == nil {
		t.Error("expected error for invalid URL, got nil")
	}

	if !strings.Contains(err.Error(), "Invalid URL") {
		t.Errorf("expected error message to contain 'Invalid URL', got %v", err)
	}
}

func TestRunCheck_NegativeWarningThreshold(t *testing.T) {
	urls = []string{"https://example.com"}
	validityDaysWarning = -1
	verbose = false

	cmd := &cobra.Command{}
	err := runCheck(cmd, []string{})

	if err == nil {
		t.Error("expected error for negative warning threshold, got nil")
	}

	if !strings.Contains(err.Error(), "Warning threshold must be a non-negative integer") {
		t.Errorf("expected error message about warning threshold, got %v", err)
	}
}

func TestRunCheck_ValidURL(t *testing.T) {
	urls = []string{"https://example.com"}
	validityDaysWarning = 90
	verbose = false

	cmd := &cobra.Command{}
	err := runCheck(cmd, []string{})

	if err != nil {
		t.Errorf("expected no error for valid URL, got %v", err)
	}
}

func TestRunCheck_MultipleURLs(t *testing.T) {
	urls = []string{"https://example.com", "https://google.com"}
	validityDaysWarning = 90
	verbose = false

	cmd := &cobra.Command{}
	err := runCheck(cmd, []string{})

	if err != nil {
		t.Errorf("expected no error for multiple valid URLs, got %v", err)
	}
}

func TestRunCheck_URLWithPort(t *testing.T) {
	urls = []string{"https://example.com:443"}
	validityDaysWarning = 90
	verbose = false

	cmd := &cobra.Command{}
	err := runCheck(cmd, []string{})

	if err != nil {
		t.Errorf("expected no error for URL with port, got %v", err)
	}
}

func TestRunCheck_ZeroWarningThreshold(t *testing.T) {
	urls = []string{"https://example.com"}
	validityDaysWarning = 0
	verbose = false

	cmd := &cobra.Command{}
	err := runCheck(cmd, []string{})

	if err != nil {
		t.Errorf("expected no error for zero warning threshold, got %v", err)
	}
}

func TestInit(t *testing.T) {
	if rootCmd == nil {
		t.Fatal("rootCmd should not be nil")
	}

	if checkCmd == nil {
		t.Fatal("checkCmd should not be nil")
	}

	if rootCmd.Use != "sharp-cred-manager" {
		t.Errorf("expected rootCmd.Use to be 'sharp-cred-manager', got '%s'", rootCmd.Use)
	}

	if checkCmd.Use != "check" {
		t.Errorf("expected checkCmd.Use to be 'check', got '%s'", checkCmd.Use)
	}

	verboseFlag := rootCmd.PersistentFlags().Lookup("verbose")
	if verboseFlag == nil {
		t.Error("expected 'verbose' flag to be defined")
	}

	warningFlag := checkCmd.Flags().Lookup("warning-threshold")
	if warningFlag == nil {
		t.Error("expected 'warning-threshold' flag to be defined")
	}

	urlFlag := checkCmd.Flags().Lookup("url")
	if urlFlag == nil {
		t.Error("expected 'url' flag to be defined")
	}
}

func TestMain_CommandStructure(t *testing.T) {
	if !rootCmd.HasSubCommands() {
		t.Error("expected rootCmd to have subcommands")
	}

	hasCheckCmd := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "check" {
			hasCheckCmd = true
			break
		}
	}

	if !hasCheckCmd {
		t.Error("expected rootCmd to have 'check' subcommand")
	}
}
