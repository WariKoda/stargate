package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestOmarchyThemeFilePrefersCurrentStateLocation(t *testing.T) {
	stateDir := t.TempDir()
	configDir := t.TempDir()
	t.Setenv("STARGATE_THEME_FILE", "")
	t.Setenv("XDG_STATE_HOME", stateDir)
	t.Setenv("XDG_CONFIG_HOME", configDir)

	stateFile := writeThemeFile(t, stateDir)
	writeThemeFile(t, configDir)

	if got := omarchyThemeFile(); got != stateFile {
		t.Fatalf("omarchyThemeFile() = %q, want %q", got, stateFile)
	}
}

func TestOmarchyThemeFileSupportsLegacyConfigLocation(t *testing.T) {
	stateDir := t.TempDir()
	configDir := t.TempDir()
	t.Setenv("STARGATE_THEME_FILE", "")
	t.Setenv("XDG_STATE_HOME", stateDir)
	t.Setenv("XDG_CONFIG_HOME", configDir)

	configFile := writeThemeFile(t, configDir)

	if got := omarchyThemeFile(); got != configFile {
		t.Fatalf("omarchyThemeFile() = %q, want %q", got, configFile)
	}
}

func TestLoadOmarchyPaletteUsesThemeRoles(t *testing.T) {
	file := filepath.Join(t.TempDir(), "colors.toml")
	content := `
accent = "#dcd7ba"
muted = "#54546d"
background = "#1f1f28"
foreground = "#c8c093"
`
	if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	colors, err := loadOmarchyPalette(file)
	if err != nil {
		t.Fatal(err)
	}

	assertColor(t, "accent", colors.accent, "#dcd7ba")
	assertColor(t, "muted", colors.muted, "#54546d")
	assertColor(t, "foreground", colors.foreground, "#c8c093")
	assertColor(t, "accent text", colors.accentText, "#1f1f28")
}

func TestANSIPaletteUsesTerminalColors(t *testing.T) {
	colors := ansiPalette()
	assertColor(t, "accent", colors.accent, "4")
	assertColor(t, "muted", colors.muted, "8")
	assertColor(t, "foreground", colors.foreground, "7")
	assertColor(t, "accent text", colors.accentText, "0")
}

func TestHelpHintsUseReadableThemeForeground(t *testing.T) {
	t.Setenv("STARGATE_THEME", themeANSI)

	model := New(nil, nil, nil)
	for name, style := range map[string]lipgloss.Style{
		"short key":         model.list.Help.Styles.ShortKey,
		"short description": model.list.Help.Styles.ShortDesc,
		"full key":          model.list.Help.Styles.FullKey,
		"full description":  model.list.Help.Styles.FullDesc,
	} {
		if got := style.GetForeground(); got != lipgloss.Color("7") {
			t.Errorf("%s foreground = %v, want ANSI foreground", name, got)
		}
	}
	if !model.list.Help.Styles.ShortKey.GetBold() || !model.list.Help.Styles.FullKey.GetBold() {
		t.Error("help keys are not bold")
	}
}

func writeThemeFile(t *testing.T, baseDir string) string {
	t.Helper()
	file := filepath.Join(baseDir, "omarchy", "current", "theme", "colors.toml")
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("background = \"#000000\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return file
}

func assertColor(t *testing.T, name string, got lipgloss.TerminalColor, want string) {
	t.Helper()
	if got != lipgloss.Color(want) {
		t.Errorf("%s = %v, want %s", name, got, want)
	}
}
