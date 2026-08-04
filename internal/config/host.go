package config

import (
	"strings"

	sshconfig "github.com/kevinburke/ssh_config"
)

// Host is a concrete SSH alias displayed by Stargate.
type Host struct {
	Alias        string
	HostName     string
	User         string
	Port         string
	IdentityFile string
	ProxyJump    string
	ProxyCommand string
	ViaJump      bool
	SourceFile   string
}

func buildHost(cfg *sshconfig.Config, alias, sourceFile string) (Host, error) {
	hostName, err := cfg.Get(alias, "HostName")
	if err != nil {
		return Host{}, err
	}
	if hostName == "" {
		hostName = alias
	}

	user, err := cfg.Get(alias, "User")
	if err != nil {
		return Host{}, err
	}

	port, err := cfg.Get(alias, "Port")
	if err != nil {
		return Host{}, err
	}
	if port == "" {
		port = sshconfig.Default("Port")
	}

	identityFile, err := cfg.Get(alias, "IdentityFile")
	if err != nil {
		return Host{}, err
	}

	proxyJump, err := cfg.Get(alias, "ProxyJump")
	if err != nil {
		return Host{}, err
	}
	proxyCommand, err := cfg.Get(alias, "ProxyCommand")
	if err != nil {
		return Host{}, err
	}

	return Host{
		Alias:        alias,
		HostName:     hostName,
		User:         user,
		Port:         port,
		IdentityFile: identityFile,
		ProxyJump:    proxyJump,
		ProxyCommand: proxyCommand,
		ViaJump:      activeProxy(proxyJump) || activeProxy(proxyCommand),
		SourceFile:   sourceFile,
	}, nil
}

func isConcreteAlias(pattern string) bool {
	return pattern != "" && !strings.HasPrefix(pattern, "!") && !strings.ContainsAny(pattern, "*?")
}

func activeProxy(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && !strings.EqualFold(value, "none")
}
