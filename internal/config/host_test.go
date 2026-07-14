package config

import (
	"strings"
	"testing"

	sshconfig "github.com/kevinburke/ssh_config"
)

func TestBuildHostUsesEffectiveValuesAndDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := sshconfig.Decode(strings.NewReader(`
Host production
    HostName prod.example.com
    User deploy
    IdentityFile ~/.ssh/production
    ProxyJump bastion

Host *
    User fallback
    Port 2200
`))
	if err != nil {
		t.Fatalf("decode config: %v", err)
	}

	host, err := buildHost(cfg, "production", "/tmp/config")
	if err != nil {
		t.Fatalf("build host: %v", err)
	}

	if host.Alias != "production" {
		t.Errorf("Alias = %q, want production", host.Alias)
	}
	if host.HostName != "prod.example.com" {
		t.Errorf("HostName = %q, want prod.example.com", host.HostName)
	}
	if host.User != "deploy" {
		t.Errorf("User = %q, want deploy", host.User)
	}
	if host.Port != "2200" {
		t.Errorf("Port = %q, want 2200", host.Port)
	}
	if host.IdentityFile != "~/.ssh/production" {
		t.Errorf("IdentityFile = %q, want ~/.ssh/production", host.IdentityFile)
	}
	if !host.ViaJump {
		t.Error("ViaJump = false, want true")
	}
	if host.SourceFile != "/tmp/config" {
		t.Errorf("SourceFile = %q, want /tmp/config", host.SourceFile)
	}
}

func TestBuildHostFallsBackToAliasAndDefaultPort(t *testing.T) {
	t.Parallel()

	cfg, err := sshconfig.Decode(strings.NewReader("Host local\n"))
	if err != nil {
		t.Fatalf("decode config: %v", err)
	}

	host, err := buildHost(cfg, "local", "/tmp/config")
	if err != nil {
		t.Fatalf("build host: %v", err)
	}

	if host.HostName != "local" {
		t.Errorf("HostName = %q, want local", host.HostName)
	}
	if host.Port != "22" {
		t.Errorf("Port = %q, want 22", host.Port)
	}
}

func TestBuildHostTreatsNoneAsNoJump(t *testing.T) {
	t.Parallel()

	cfg, err := sshconfig.Decode(strings.NewReader(`
Host direct
    ProxyJump none
    ProxyCommand NONE
`))
	if err != nil {
		t.Fatalf("decode config: %v", err)
	}

	host, err := buildHost(cfg, "direct", "/tmp/config")
	if err != nil {
		t.Fatalf("build host: %v", err)
	}
	if host.ViaJump {
		t.Error("ViaJump = true, want false")
	}
}

func TestIsConcreteAlias(t *testing.T) {
	t.Parallel()

	tests := map[string]bool{
		"server":         true,
		"server.example": true,
		"":               false,
		"*":              false,
		"*.example":      false,
		"server?":        false,
		"!blocked":       false,
	}

	for pattern, want := range tests {
		if got := isConcreteAlias(pattern); got != want {
			t.Errorf("isConcreteAlias(%q) = %t, want %t", pattern, got, want)
		}
	}
}
