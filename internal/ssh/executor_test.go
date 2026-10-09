package ssh

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNewFindsSSHAndBuildsSafeCommand(t *testing.T) {
	binDir := t.TempDir()
	binaryPath := filepath.Join(binDir, binaryName)
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatalf("write fake ssh: %v", err)
	}
	t.Setenv("PATH", binDir)

	executor, err := New("/tmp/ssh-config")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	command, err := executor.Command("-production")
	if err != nil {
		t.Fatalf("Command: %v", err)
	}

	want := []string{binaryPath, "-F", "/tmp/ssh-config", "--", "-production"}
	if !reflect.DeepEqual(command.Args, want) {
		t.Errorf("Command.Args = %#v, want %#v", command.Args, want)
	}
}

func TestPreviewQuotesTheExactCommand(t *testing.T) {
	t.Parallel()

	executor := Executor{binaryPath: "/opt/OpenSSH Client/ssh", configPath: "/home/user/.ssh/work config"}
	preview, err := executor.Preview("customer's-host")
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}

	want := `'/opt/OpenSSH Client/ssh' -F '/home/user/.ssh/work config' -- 'customer'"'"'s-host'`
	if preview != want {
		t.Errorf("Preview() = %q, want %q", preview, want)
	}
}

func TestNewRejectsMissingBinary(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	if _, err := New("/tmp/config"); err == nil || !strings.Contains(err.Error(), "PATH") {
		t.Fatalf("New without ssh error = %v, want PATH context", err)
	}
}

func TestCommandOmitsConfigFlagWithoutExplicitConfig(t *testing.T) {
	binDir := t.TempDir()
	binaryPath := filepath.Join(binDir, binaryName)
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatalf("write fake ssh: %v", err)
	}
	t.Setenv("PATH", binDir)

	executor, err := New("  ")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	command, err := executor.Command("production")
	if err != nil {
		t.Fatalf("Command: %v", err)
	}

	want := []string{binaryPath, "--", "production"}
	if !reflect.DeepEqual(command.Args, want) {
		t.Errorf("Command.Args = %#v, want %#v", command.Args, want)
	}
}

func TestCommandValidatesAliasAndExecutor(t *testing.T) {
	t.Parallel()

	executor := Executor{binaryPath: "/usr/bin/ssh", configPath: "/tmp/config"}
	for _, alias := range []string{"", "two hosts", "line\nbreak", "nul\x00byte"} {
		if _, err := executor.Command(alias); err == nil {
			t.Errorf("Command(%q) returned nil error", alias)
		}
	}

	if _, err := (Executor{configPath: "/tmp/config"}).Command("host"); err == nil {
		t.Error("Command with empty binary path returned nil error")
	}
}
