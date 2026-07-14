package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestResolvePathPrecedenceAndSymlinks(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	defaultPath := filepath.Join(home, ".ssh", "config")
	writeTestFile(t, defaultPath, "Host default\n")
	environmentPath := filepath.Join(home, "environment-config")
	writeTestFile(t, environmentPath, "Host environment\n")
	explicitPath := filepath.Join(home, "explicit-config")
	writeTestFile(t, explicitPath, "Host explicit\n")
	symlinkPath := filepath.Join(home, "config-link")
	if err := os.Symlink(explicitPath, symlinkPath); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	t.Run("default", func(t *testing.T) {
		t.Setenv(ConfigEnvironment, "")
		got, err := ResolvePath("")
		if err != nil {
			t.Fatalf("ResolvePath: %v", err)
		}
		if got != defaultPath {
			t.Errorf("ResolvePath() = %q, want %q", got, defaultPath)
		}
	})

	t.Run("environment over default", func(t *testing.T) {
		t.Setenv(ConfigEnvironment, environmentPath)
		got, err := ResolvePath("")
		if err != nil {
			t.Fatalf("ResolvePath: %v", err)
		}
		if got != environmentPath {
			t.Errorf("ResolvePath() = %q, want %q", got, environmentPath)
		}
	})

	t.Run("explicit over environment and resolves symlink", func(t *testing.T) {
		t.Setenv(ConfigEnvironment, environmentPath)
		got, err := ResolvePath(symlinkPath)
		if err != nil {
			t.Fatalf("ResolvePath: %v", err)
		}
		if got != explicitPath {
			t.Errorf("ResolvePath() = %q, want %q", got, explicitPath)
		}
	})
}

func TestResolvePathRejectsMissingAndUnsupportedHomePath(t *testing.T) {
	t.Setenv(ConfigEnvironment, "")

	if _, err := ResolvePath(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("ResolvePath(missing) returned nil error")
	}
	if _, err := ResolvePath("~someone/config"); err == nil {
		t.Fatal("ResolvePath(~someone/config) returned nil error")
	}
}

func TestLoadDiscoversIncludedAndConcreteHosts(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	includeDir := filepath.Join(home, "includes")
	includedPath := filepath.Join(includeDir, "jump.conf")
	writeTestFile(t, includedPath, `
Host jumpbox
    HostName jump.example.com
    ProxyCommand ssh gateway -W %h:%p

Host *.internal !blocked
    User ignored
`)

	rootPath := filepath.Join(home, "config")
	writeTestFile(t, rootPath, strings.Join([]string{
		"Include " + filepath.Join(includeDir, "*.conf"),
		"Host production prod-alias *.example !blocked",
		"    HostName prod.example.com",
		"    User deploy",
		"Host *",
		"    Port 2200",
		"",
	}, "\n"))

	result, err := Load(rootPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if result.Path != rootPath {
		t.Errorf("Result.Path = %q, want %q", result.Path, rootPath)
	}

	aliases := make([]string, 0, len(result.Hosts))
	for _, host := range result.Hosts {
		aliases = append(aliases, host.Alias)
	}
	wantAliases := []string{"jumpbox", "production", "prod-alias"}
	if !reflect.DeepEqual(aliases, wantAliases) {
		t.Errorf("aliases = %#v, want %#v", aliases, wantAliases)
	}

	jumpbox := result.Hosts[0]
	if jumpbox.HostName != "jump.example.com" {
		t.Errorf("jumpbox HostName = %q, want jump.example.com", jumpbox.HostName)
	}
	if jumpbox.Port != "2200" {
		t.Errorf("jumpbox Port = %q, want 2200", jumpbox.Port)
	}
	if !jumpbox.ViaJump {
		t.Error("jumpbox ViaJump = false, want true")
	}
	if jumpbox.SourceFile != includedPath {
		t.Errorf("jumpbox SourceFile = %q, want %q", jumpbox.SourceFile, includedPath)
	}

	production := result.Hosts[1]
	if production.User != "deploy" {
		t.Errorf("production User = %q, want deploy", production.User)
	}
	if production.Port != "2200" {
		t.Errorf("production Port = %q, want 2200", production.Port)
	}
}

func TestLoadDeduplicatesAliasesAtFirstDeclaration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")
	writeTestFile(t, path, `
Host duplicate
    User first
Host duplicate
    User second
`)

	result, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(result.Hosts) != 1 {
		t.Fatalf("len(Hosts) = %d, want 1", len(result.Hosts))
	}
	if result.Hosts[0].User != "first" {
		t.Errorf("User = %q, want first", result.Hosts[0].User)
	}
}

func TestLoadSkipsMatchBlocks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")
	writeTestFile(t, path, `
Match host hidden
    User conditional
Host visible
    User regular
`)

	result, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(result.Hosts) != 1 || result.Hosts[0].Alias != "visible" {
		t.Fatalf("Hosts = %#v, want only visible", result.Hosts)
	}
}

func TestLoadRejectsUnsupportedMatchExec(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")
	writeTestFile(t, path, `
Match exec "true"
    User conditional
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load(Match exec) returned nil error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "match exec") {
		t.Fatalf("Load(Match exec) error = %q, want Match Exec context", err)
	}
}

func TestResolveIncludePattern(t *testing.T) {
	t.Parallel()

	home := "/home/tester"
	tests := map[string]string{
		"config.d/*":          "/home/tester/.ssh/config.d/*",
		"~/.ssh/other/*":      "/home/tester/.ssh/other/*",
		"/etc/ssh/ssh_config": "/etc/ssh/ssh_config",
	}
	for input, want := range tests {
		if got := resolveIncludePattern(input, home); got != want {
			t.Errorf("resolveIncludePattern(%q) = %q, want %q", input, got, want)
		}
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create test directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write test file: %v", err)
	}
}
