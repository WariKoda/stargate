package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/lipgloss"
)

const (
	themeAuto    = "auto"
	themeANSI    = "ansi"
	themeOmarchy = "omarchy"
	themeDefault = "default"
)

type palette struct {
	accent     lipgloss.TerminalColor
	muted      lipgloss.TerminalColor
	foreground lipgloss.TerminalColor
	accentText lipgloss.TerminalColor
}

type omarchyColors struct {
	Accent         string `toml:"accent"`
	Foreground     string `toml:"foreground"`
	DarkForeground string `toml:"dark_foreground"`
	Background     string `toml:"background"`
	Muted          string `toml:"muted"`
	Blue           string `toml:"blue"`
	Color0         string `toml:"color0"`
	Color4         string `toml:"color4"`
	Color7         string `toml:"color7"`
	Color8         string `toml:"color8"`
}

func loadPalette() palette {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("STARGATE_THEME")))
	if mode == "" {
		mode = themeAuto
	}

	switch mode {
	case themeANSI:
		return ansiPalette()
	case themeOmarchy:
		if result, err := loadOmarchyPalette(omarchyThemeFile()); err == nil {
			return result
		}
		return defaultPalette()
	case themeAuto:
		if result, err := loadOmarchyPalette(omarchyThemeFile()); err == nil {
			return result
		}
		return ansiPalette()
	case themeDefault:
		fallthrough
	default:
		return defaultPalette()
	}
}

func omarchyThemeFile() string {
	if file := strings.TrimSpace(os.Getenv("STARGATE_THEME_FILE")); file != "" {
		return file
	}

	home, _ := os.UserHomeDir()
	stateDir := strings.TrimSpace(os.Getenv("XDG_STATE_HOME"))
	if stateDir == "" && home != "" {
		stateDir = filepath.Join(home, ".local", "state")
	}
	configDir := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
	if configDir == "" && home != "" {
		configDir = filepath.Join(home, ".config")
	}

	var candidates []string
	if stateDir != "" {
		candidates = append(candidates, filepath.Join(stateDir, "omarchy", "current", "theme", "colors.toml"))
	}
	if configDir != "" {
		candidates = append(candidates, filepath.Join(configDir, "omarchy", "current", "theme", "colors.toml"))
	}
	for _, file := range candidates {
		if _, err := os.Stat(file); err == nil {
			return file
		}
	}
	if len(candidates) > 0 {
		return candidates[0]
	}
	return ""
}

func defaultPalette() palette {
	return palette{
		accent:     lipgloss.AdaptiveColor{Light: "#4A2ED6", Dark: "#9B87FF"},
		muted:      lipgloss.AdaptiveColor{Light: "#6B5FA8", Dark: "#AFA3D8"},
		foreground: lipgloss.AdaptiveColor{Light: "#37474F", Dark: "#BAC2DE"},
		accentText: lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#111111"},
	}
}

func ansiPalette() palette {
	return palette{
		accent:     lipgloss.Color("4"),
		muted:      lipgloss.Color("8"),
		foreground: lipgloss.Color("7"),
		accentText: lipgloss.Color("0"),
	}
}

func loadOmarchyPalette(file string) (palette, error) {
	if file == "" {
		return palette{}, fmt.Errorf("empty theme path")
	}

	var colors omarchyColors
	if _, err := toml.DecodeFile(file, &colors); err != nil {
		return palette{}, err
	}

	foreground := normalizeHex(first(colors.Foreground, colors.Color7, "#ffffff"))
	return palette{
		accent:     lipgloss.Color(normalizeHex(first(colors.Accent, colors.Blue, colors.Color4, foreground))),
		muted:      lipgloss.Color(normalizeHex(first(colors.Muted, colors.DarkForeground, colors.Color8, colors.Color0, foreground))),
		foreground: lipgloss.Color(foreground),
		accentText: lipgloss.Color(normalizeHex(first(colors.Background, colors.Color0, "#000000"))),
	}, nil
}

func first(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func normalizeHex(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"'`)
	if !strings.HasPrefix(value, "#") {
		value = "#" + value
	}
	return value
}
