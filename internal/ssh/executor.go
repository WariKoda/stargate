package ssh

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"unicode"
)

const binaryName = "ssh"

// Executor builds native OpenSSH commands for aliases loaded by Stargate.
type Executor struct {
	binaryPath string
	configPath string
}

// New locates the native ssh client and creates an Executor for configPath.
func New(configPath string) (Executor, error) {
	binaryPath, err := exec.LookPath(binaryName)
	if err != nil {
		return Executor{}, fmt.Errorf("find %s in PATH: %w", binaryName, err)
	}
	if strings.TrimSpace(configPath) == "" {
		return Executor{}, errors.New("SSH config path must not be empty")
	}

	return Executor{
		binaryPath: binaryPath,
		configPath: configPath,
	}, nil
}

// Command builds an interactive ssh command for alias. The -- separator keeps
// aliases beginning with a dash from being interpreted as command options.
func (e Executor) Command(alias string) (*exec.Cmd, error) {
	if err := validateAlias(alias); err != nil {
		return nil, err
	}
	if e.binaryPath == "" {
		return nil, errors.New("SSH executable path must not be empty")
	}
	if e.configPath == "" {
		return nil, errors.New("SSH config path must not be empty")
	}

	return exec.Command(e.binaryPath, "-F", e.configPath, "--", alias), nil
}

func validateAlias(alias string) error {
	if alias == "" {
		return errors.New("SSH alias must not be empty")
	}
	if strings.IndexByte(alias, 0) >= 0 {
		return errors.New("SSH alias must not contain NUL bytes")
	}
	if strings.IndexFunc(alias, unicode.IsSpace) >= 0 {
		return fmt.Errorf("SSH alias %q must not contain whitespace", alias)
	}
	return nil
}
