package config

import (
	"testing"
)

func TestBuildHostUsesEffectiveValuesAndDefaults(t *testing.T) {
	t.Parallel()

	doc := parseTestDocument(t, `
Host production
    HostName prod.example.com
    User deploy
    IdentityFile ~/.ssh/production
    ProxyJump bastion

Host *
    User fallback
    Port 2200
`)

	host := buildHost(doc, "production", "/tmp/config")

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
	if host.ProxyJump != "bastion" {
		t.Errorf("ProxyJump = %q, want bastion", host.ProxyJump)
	}
	if host.ProxyCommand != "" {
		t.Errorf("ProxyCommand = %q, want empty", host.ProxyCommand)
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

	doc := parseTestDocument(t, "Host local\n")

	host := buildHost(doc, "local", "/tmp/config")

	if host.HostName != "local" {
		t.Errorf("HostName = %q, want local", host.HostName)
	}
	if host.Port != "22" {
		t.Errorf("Port = %q, want 22", host.Port)
	}
}

func TestBuildHostTreatsNoneAsNoJump(t *testing.T) {
	t.Parallel()

	doc := parseTestDocument(t, `
Host direct
    ProxyJump none
    ProxyCommand NONE
`)

	host := buildHost(doc, "direct", "/tmp/config")
	if host.ProxyJump != "none" {
		t.Errorf("ProxyJump = %q, want none", host.ProxyJump)
	}
	if host.ProxyCommand != "NONE" {
		t.Errorf("ProxyCommand = %q, want NONE", host.ProxyCommand)
	}
	if host.ViaJump {
		t.Error("ViaJump = true, want false")
	}
}

func TestBuildHostExpandsHostNameTokens(t *testing.T) {
	t.Parallel()

	doc := parseTestDocument(t, `
Host web literal
    HostName %h.corp.example
Host literal
    HostName 100%%-%h-%x
`)

	if host := buildHost(doc, "web", "/tmp/config"); host.HostName != "web.corp.example" {
		t.Errorf("HostName = %q, want web.corp.example", host.HostName)
	}
	if host := buildHost(doc, "literal", "/tmp/config"); host.HostName != "literal.corp.example" {
		t.Errorf("first HostName wins: got %q, want literal.corp.example", host.HostName)
	}

	tests := map[string]string{
		"%h.example": "alias.example",
		"100%%":      "100%",
		"%%h":        "%h",
		"keep%x":     "keep%x",
		"trailing%":  "trailing%",
	}
	for input, want := range tests {
		if got := expandHostName(input, "alias"); got != want {
			t.Errorf("expandHostName(%q) = %q, want %q", input, got, want)
		}
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

func parseTestDocument(t *testing.T, content string) *document {
	t.Helper()
	doc, err := parseDocument("/tmp/config", []byte(content), t.TempDir(), 0)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	return doc
}
