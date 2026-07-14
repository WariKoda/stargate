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
	Path  string
	Hosts []Host
}

// ResolvePath resolves the SSH config path using this precedence:
// explicit path, STARGATE_CONFIG, ~/.ssh/config. Symlinks are resolved.
func ResolvePath(explicitPath string) (string, error) {
	path := strings.TrimSpace(explicitPath)
	if path == "" {
		path = strings.TrimSpace(os.Getenv(ConfigEnvironment))
	}
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		path = filepath.Join(home, ".ssh", "config")
	}

	expanded, err := expandHome(path)
	if err != nil {
		return "", err
	}
	absolute, err := filepath.Abs(expanded)
	if err != nil {
		return "", fmt.Errorf("resolve absolute config path %q: %w", expanded, err)
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("resolve SSH config %q: %w", absolute, err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("stat SSH config %q: %w", resolved, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("SSH config %q is not a regular file", resolved)
	}

	return resolved, nil
}

// Load resolves and parses an SSH config and returns its concrete aliases.
func Load(explicitPath string) (Result, error) {
	path, err := ResolvePath(explicitPath)
	if err != nil {
		return Result{}, err
	}

	cfg, err := decodeFile(path)
	if err != nil {
		return Result{}, err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return Result{}, fmt.Errorf("resolve home directory: %w", err)
	}

	aliases := make([]discoveredAlias, 0)
	visited := make(map[string]struct{})
	if err := discoverAliases(path, cfg, home, 0, visited, &aliases); err != nil {
		return Result{}, err
	}

	hosts := make([]Host, 0, len(aliases))
	seenAliases := make(map[string]struct{}, len(aliases))
	for _, discovered := range aliases {
		if _, exists := seenAliases[discovered.name]; exists {
			continue
		}

		host, err := buildHost(cfg, discovered.name, discovered.sourceFile)
		if err != nil {
			return Result{}, fmt.Errorf("resolve SSH host %q: %w", discovered.name, err)
		}
		seenAliases[discovered.name] = struct{}{}
		hosts = append(hosts, host)
	}

	return Result{Path: path, Hosts: hosts}, nil
}

type discoveredAlias struct {
	name       string
	sourceFile string
}

func decodeFile(path string) (*sshconfig.Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open SSH config %q: %w", path, err)
	}
	defer file.Close()

	cfg, err := sshconfig.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("parse SSH config %q: %w", path, err)
	}

	return cfg, nil
}

func discoverAliases(
	path string,
	cfg *sshconfig.Config,
	home string,
	depth int,
	visited map[string]struct{},
	aliases *[]discoveredAlias,
) error {
	if depth > maxIncludeDepth {
		return sshconfig.ErrDepthExceeded
	}

	canonicalPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return fmt.Errorf("resolve included SSH config %q: %w", path, err)
	}
	if _, exists := visited[canonicalPath]; exists {
		return nil
	}
	visited[canonicalPath] = struct{}{}

	for _, block := range cfg.Hosts {
		if isMatchBlock(block) {
			continue
		}

		for _, pattern := range block.Patterns {
			name := pattern.String()
			if isConcreteAlias(name) {
				*aliases = append(*aliases, discoveredAlias{name: name, sourceFile: canonicalPath})
			}
		}

		for _, node := range block.Nodes {
			include, ok := node.(*sshconfig.Include)
			if !ok {
				continue
			}
			if err := discoverIncludes(include, home, depth, visited, aliases); err != nil {
				return err
			}
		}
	}

	return nil
}

func discoverIncludes(
	include *sshconfig.Include,
	home string,
	depth int,
	visited map[string]struct{},
	aliases *[]discoveredAlias,
) error {
	directives := includeDirectives(include)
	seenMatches := make(map[string]struct{})
	for _, directive := range directives {
		pattern := resolveIncludePattern(directive, home)
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return fmt.Errorf("expand Include pattern %q: %w", directive, err)
		}
		for _, match := range matches {
			if _, exists := seenMatches[match]; exists {
				continue
			}
			seenMatches[match] = struct{}{}

			includedConfig, err := decodeFile(match)
			if err != nil {
				return err
			}
			if err := discoverAliases(match, includedConfig, home, depth+1, visited, aliases); err != nil {
				return err
			}
		}
	}

	return nil
}

// ssh_config exposes included values but not the directive paths. String is
// its stable public representation; the parser itself accepts space-separated
// Include paths, so Fields mirrors the supported syntax.
func includeDirectives(include *sshconfig.Include) []string {
	line := strings.TrimSpace(include.String())
	line = strings.TrimSpace(strings.TrimPrefix(line, "Include"))
	line = strings.TrimSpace(strings.TrimPrefix(line, "="))
	if include.Comment != "" {
		line = strings.TrimSuffix(line, " #"+include.Comment)
	}
	return strings.Fields(line)
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
