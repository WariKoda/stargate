package main

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunHandlesHelpBeforeLoadingConfig(t *testing.T) {
	t.Parallel()

	if err := run([]string{"--help"}); !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("run(--help) error = %v, want flag.ErrHelp", err)
	}
}

func TestRunRejectsPositionalArguments(t *testing.T) {
	t.Parallel()

	err := run([]string{"unexpected"})
	if err == nil || !strings.Contains(err.Error(), "unexpected arguments") {
		t.Fatalf("run(unexpected) error = %v", err)
	}
}

func TestRequireInteractiveTerminal(t *testing.T) {
	t.Parallel()

	if err := requireInteractiveTerminal(func(uintptr) bool { return true }); err != nil {
		t.Fatalf("requireInteractiveTerminal(all terminals): %v", err)
	}
	if err := requireInteractiveTerminal(func(uintptr) bool { return false }); err == nil {
		t.Fatal("requireInteractiveTerminal(no terminals) returned nil error")
	}
}

func TestRunChecksForSSHBeforeStartingTUI(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(configPath, []byte("Host server\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv("PATH", t.TempDir())

	err := run([]string{"--config", configPath})
	if err == nil || !strings.Contains(err.Error(), "find ssh in PATH") {
		t.Fatalf("run without ssh error = %v, want PATH context", err)
	}
}
