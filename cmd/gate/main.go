package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

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

	program := tea.NewProgram(
		tui.New(loaded.Hosts, executor),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := program.Run(); err != nil {
		return fmt.Errorf("run TUI: %w", err)
	}
	return nil
}

func requireInteractiveTerminal(isTerminal func(uintptr) bool) error {
	if !isTerminal(os.Stdin.Fd()) || !isTerminal(os.Stdout.Fd()) {
		return errors.New("interactive terminal required on stdin and stdout")
	}
	return nil
}
