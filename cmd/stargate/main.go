package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	appconfig "github.com/nibra/stargate/internal/config"
	"github.com/nibra/stargate/internal/tui"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintf(os.Stderr, "stargate: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("stargate", flag.ContinueOnError)
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

	program := tea.NewProgram(tui.New(loaded.Hosts), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		return fmt.Errorf("run TUI: %w", err)
	}
	return nil
}
