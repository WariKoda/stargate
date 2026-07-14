package main

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	sshexecutor "github.com/nibra180/stargate-tui/internal/ssh"
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

func TestPrintCommandWritesPreview(t *testing.T) {
	binDir := t.TempDir()
	binaryPath := filepath.Join(binDir, "ssh")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatalf("write fake ssh: %v", err)
	}
	t.Setenv("PATH", binDir)

	executor, err := sshexecutor.New("/tmp/ssh config")
	if err != nil {
		t.Fatalf("New executor: %v", err)
	}
	var output bytes.Buffer
	if err := printCommand(&output, executor, "production"); err != nil {
		t.Fatalf("printCommand: %v", err)
	}
	if got, want := output.String(), binaryPath+" -F '/tmp/ssh config' -- production\n"; got != want {
		t.Errorf("printCommand output = %q, want %q", got, want)
	}
}

func TestSSHEnvironmentMirrorsGhosttyWrapper(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		environ []string
		want    []string
	}{
		{
			name:    "ssh-env falls back to widely supported terminfo",
			environ: []string{"PATH=/usr/bin", "TERM=xterm-ghostty", "GHOSTTY_SHELL_FEATURES=path,ssh-env,title"},
			want:    []string{"PATH=/usr/bin", "GHOSTTY_SHELL_FEATURES=path,ssh-env,title", "TERM=xterm-256color"},
		},
		{
			name:    "ssh-terminfo also has a safe fallback",
			environ: []string{"TERM=xterm-ghostty", "GHOSTTY_SHELL_FEATURES=ssh-terminfo"},
			want:    []string{"GHOSTTY_SHELL_FEATURES=ssh-terminfo", "TERM=xterm-256color"},
		},
		{
			name:    "Ghostty without SSH integration remains unchanged",
			environ: []string{"TERM=xterm-ghostty", "GHOSTTY_SHELL_FEATURES=path,title"},
			want:    []string{"TERM=xterm-ghostty", "GHOSTTY_SHELL_FEATURES=path,title"},
		},
		{
			name:    "other terminals remain unchanged",
			environ: []string{"TERM=screen-256color", "GHOSTTY_SHELL_FEATURES=ssh-env"},
			want:    []string{"TERM=screen-256color", "GHOSTTY_SHELL_FEATURES=ssh-env"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := sshEnvironment(test.environ); !slices.Equal(got, test.want) {
				t.Errorf("sshEnvironment() = %#v, want %#v", got, test.want)
			}
		})
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
