package main

import (
	"errors"
	"flag"
	"strings"
	"testing"
)

func TestRunHandlesHelpBeforeLoadingConfig(t *testing.T) {
	t.Parallel()

	if err := run([]string{"--help"}); !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("run(--help) error = %v, want flag.ErrHelp", err)
	}
}

func TestRunRejectsPositionalArguments(t *testing.T) {
	t.Parallel()

	err := run([]string{"unexpected"})
	if err == nil || !strings.Contains(err.Error(), "unexpected arguments") {
		t.Fatalf("run(unexpected) error = %v", err)
	}
}
