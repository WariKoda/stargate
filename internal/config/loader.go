package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sshconfig "github.com/kevinburke/ssh_config"
)

const (
	// ConfigEnvironment is the environment variable used to override the SSH config path.
	ConfigEnvironment = "STARGATE_CONFIG"
	maxIncludeDepth   = 5
)

// Result contains the resolved config path and all concrete SSH aliases.
type Result struct {
	Path string
	// Explicit reports whether Path came from --config or STARGATE_CONFIG
	// instead of the OpenSSH default ~/.ssh/config.
	Explicit bool
	Hosts    []Host
}

// ResolvePath resolves the SSH config path using this precedence:
// explicit path, STARGATE_CONFIG, ~/.ssh/config. Symlinks are resolved.
func ResolvePath(explicitPath string) (string, error) {
	path, _, err := resolvePath(explicitPath)
	return path, err
}

func resolvePath(explicitPath string) (string, bool, error) {
	path := strings.TrimSpace(explicitPath)
	if path == "" {
		path = strings.TrimSpace(os.Getenv(ConfigEnvironment))
	}
	explicit := path != ""
	if !explicit {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", false, fmt.Errorf("resolve home directory: %w", err)
		}
		path = filepath.Join(home, ".ssh", "config")
	}

	expanded, err := expandHome(path)
	if err != nil {
		return "", false, err
	}
	absolute, err := filepath.Abs(expanded)
	if err != nil {
		return "", false, fmt.Errorf("resolve absolute config path %q: %w", expanded, err)
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", false, fmt.Errorf("resolve SSH config %q: %w", absolute, err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", false, fmt.Errorf("stat SSH config %q: %w", resolved, err)
	}
	if !info.Mode().IsRegular() {
		return "", false, fmt.Errorf("SSH config %q is not a regular file", resolved)
	}

	return resolved, explicit, nil
}

// Load resolves and parses an SSH config and returns its concrete aliases.
func Load(explicitPath string) (Result, error) {
	path, explicit, err := resolvePath(explicitPath)
	if err != nil {
		return Result{}, err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return Result{}, fmt.Errorf("resolve home directory: %w", err)
	}

	root, err := loadDocument(path, home, 0)
	if err != nil {
		return Result{}, err
	}

	aliases := make([]discoveredAlias, 0)
	root.discoverAliases(&aliases)

	hosts := make([]Host, 0, len(aliases))
	seenAliases := make(map[string]struct{}, len(aliases))
	for _, discovered := range aliases {
		if _, exists := seenAliases[discovered.name]; exists {
			continue
		}
		seenAliases[discovered.name] = struct{}{}
		hosts = append(hosts, buildHost(root, discovered.name, discovered.sourceFile))
	}

	return Result{Path: path, Explicit: explicit, Hosts: hosts}, nil
}

type discoveredAlias struct {
	name       string
	sourceFile string
}

func (d *document) discoverAliases(aliases *[]discoveredAlias) {
	for _, block := range d.config.Hosts {
		if isMatchBlock(block) {
			continue
		}

		for _, pattern := range block.Patterns {
			name := pattern.String()
			if isConcreteAlias(name) {
				*aliases = append(*aliases, discoveredAlias{name: name, sourceFile: d.path})
			}
		}

		for _, node := range block.Nodes {
			kv, ok := node.(*sshconfig.KV)
			if !ok {
				continue
			}
			for _, included := range d.includes[kv] {
				included.discoverAliases(aliases)
			}
		}
	}
}

func resolveIncludePattern(pattern, home string) string {
	switch {
	case filepath.IsAbs(pattern):
		return pattern
	case strings.HasPrefix(pattern, "~/"):
		return filepath.Join(home, pattern[2:])
	default:
		return filepath.Join(home, ".ssh", pattern)
	}
}

func isMatchBlock(block *sshconfig.Host) bool {
	declaration, _, _ := strings.Cut(strings.TrimSpace(block.String()), "\n")
	return strings.HasPrefix(strings.ToLower(declaration), "match ") || strings.HasPrefix(strings.ToLower(declaration), "match=")
}

func expandHome(path string) (string, error) {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		return home, nil
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		return filepath.Join(home, path[2:]), nil
	}
	if strings.HasPrefix(path, "~") {
		return "", errors.New("SSH config path supports only ~ or ~/ home expansion")
	}
	return path, nil
}
