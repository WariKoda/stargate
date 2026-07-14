package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/term"

	appconfig "github.com/nibra180/stargate-tui/internal/config"
	sshexecutor "github.com/nibra180/stargate-tui/internal/ssh"
	"github.com/nibra180/stargate-tui/internal/tui"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintf(os.Stderr, "gate: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("gate", flag.ContinueOnError)
	configPath := flags.String("config", "", "path to SSH config (overrides STARGATE_CONFIG)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}

	loaded, err := appconfig.Load(*configPath)
	if err != nil {
		return err
	}

	executor, err := sshexecutor.New(loaded.Path)
	if err != nil {
		return err
	}
	if err := requireInteractiveTerminal(term.IsTerminal); err != nil {
		return err
	}

	return runSessionLoop(loaded.Hosts, executor)
}

func runSessionLoop(hosts []appconfig.Host, executor sshexecutor.Executor) error {
	var resume *tui.SessionResume

	for {
		program := tea.NewProgram(
			tui.New(hosts, executor, resume),
			tea.WithAltScreen(),
			tea.WithMouseCellMotion(),
		)
		finalModel, err := program.Run()
		if err != nil {
			return fmt.Errorf("run TUI: %w", err)
		}

		model, ok := finalModel.(tui.Model)
		if !ok {
			return fmt.Errorf("unexpected model type %T", finalModel)
		}

		host, connect := model.PendingConnect()
		if !connect {
			return nil
		}

		command, err := executor.Command(host.Alias)
		if err != nil {
			resume = &tui.SessionResume{Host: host, Err: err}
			continue
		}
		command.Stdin = os.Stdin
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		command.Env = sshEnvironment(os.Environ())
		resume = &tui.SessionResume{Host: host, Err: command.Run()}
	}
}

// sshEnvironment mirrors Ghostty's SSH shell wrapper when Stargate launches
// the OpenSSH binary directly. Without this fallback, remote hosts lacking the
// xterm-ghostty terminfo entry cannot redraw line-editing operations correctly.
func sshEnvironment(environ []string) []string {
	if environmentValue(environ, "TERM") != "xterm-ghostty" || !hasGhosttySSHIntegration(environ) {
		return environ
	}

	result := make([]string, 0, len(environ))
	for _, entry := range environ {
		if strings.HasPrefix(entry, "TERM=") {
			continue
		}
		result = append(result, entry)
	}
	return append(result, "TERM=xterm-256color")
}

func hasGhosttySSHIntegration(environ []string) bool {
	for feature := range strings.SplitSeq(environmentValue(environ, "GHOSTTY_SHELL_FEATURES"), ",") {
		switch strings.TrimSpace(feature) {
		case "ssh-env", "ssh-terminfo":
			return true
		}
	}
	return false
}

func environmentValue(environ []string, name string) string {
	prefix := name + "="
	for _, entry := range environ {
		if strings.HasPrefix(entry, prefix) {
			return strings.TrimPrefix(entry, prefix)
		}
	}
	return ""
}

func requireInteractiveTerminal(isTerminal func(uintptr) bool) error {
	if !isTerminal(os.Stdin.Fd()) || !isTerminal(os.Stdout.Fd()) {
		return errors.New("interactive terminal required on stdin and stdout")
	}
	return nil
}
