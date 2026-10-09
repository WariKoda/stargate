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

func buildHost(doc *document, alias, sourceFile string) Host {
	hostName := expandHostName(doc.get(alias, "HostName"), alias)
	if hostName == "" {
		hostName = alias
	}

	port := doc.get(alias, "Port")
	if port == "" {
		port = sshconfig.Default("Port")
	}

	proxyJump := doc.get(alias, "ProxyJump")
	proxyCommand := doc.get(alias, "ProxyCommand")

	return Host{
		Alias:        alias,
		HostName:     hostName,
		User:         doc.get(alias, "User"),
		Port:         port,
		IdentityFile: doc.get(alias, "IdentityFile"),
		ProxyJump:    proxyJump,
		ProxyCommand: proxyCommand,
		ViaJump:      activeProxy(proxyJump) || activeProxy(proxyCommand),
		SourceFile:   sourceFile,
	}
}

// expandHostName applies the only tokens OpenSSH accepts in HostName: %h for
// the alias and %% for a literal percent sign.
func expandHostName(hostName, alias string) string {
	if !strings.Contains(hostName, "%") {
		return hostName
	}

	var expanded strings.Builder
	for index := 0; index < len(hostName); index++ {
		if hostName[index] != '%' || index+1 == len(hostName) {
			expanded.WriteByte(hostName[index])
			continue
		}
		switch hostName[index+1] {
		case 'h':
			expanded.WriteString(alias)
			index++
		case '%':
			expanded.WriteByte('%')
			index++
		default:
			expanded.WriteByte('%')
		}
	}
	return expanded.String()
}

func isConcreteAlias(pattern string) bool {
	return pattern != "" && !strings.HasPrefix(pattern, "!") && !strings.ContainsAny(pattern, "*?")
}

func activeProxy(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && !strings.EqualFold(value, "none")
}
