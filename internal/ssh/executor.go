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

// New locates the native ssh client and creates an Executor. An empty
// configPath lets ssh load its default user and system-wide configs.
func New(configPath string) (Executor, error) {
	binaryPath, err := exec.LookPath(binaryName)
	if err != nil {
		return Executor{}, fmt.Errorf("find %s in PATH: %w", binaryName, err)
	}

	return Executor{
		binaryPath: binaryPath,
		configPath: strings.TrimSpace(configPath),
	}, nil
}

// Command builds an interactive ssh command for alias. The -- separator keeps
// aliases beginning with a dash from being interpreted as command options.
// -F is only passed for an explicit config because it makes ssh skip the
// system-wide /etc/ssh/ssh_config, including policies such as host key checks.
func (e Executor) Command(alias string) (*exec.Cmd, error) {
	if err := validateAlias(alias); err != nil {
		return nil, err
	}
	if e.binaryPath == "" {
		return nil, errors.New("SSH executable path must not be empty")
	}

	arguments := make([]string, 0, 4)
	if e.configPath != "" {
		arguments = append(arguments, "-F", e.configPath)
	}
	arguments = append(arguments, "--", alias)
	return exec.Command(e.binaryPath, arguments...), nil
}

// Preview returns the exact OpenSSH invocation as a shell-safe command line.
func (e Executor) Preview(alias string) (string, error) {
	command, err := e.Command(alias)
	if err != nil {
		return "", err
	}

	arguments := make([]string, len(command.Args))
	for index, argument := range command.Args {
		arguments[index] = quoteShellArgument(argument)
	}
	return strings.Join(arguments, " "), nil
}

func quoteShellArgument(argument string) string {
	if argument != "" && strings.IndexFunc(argument, func(character rune) bool {
		return !unicode.IsLetter(character) && !unicode.IsDigit(character) && !strings.ContainsRune("@%_+=:,./-", character)
	}) == -1 {
		return argument
	}
	return "'" + strings.ReplaceAll(argument, "'", `'"'"'`) + "'"
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
