package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"

	appconfig "github.com/warikoda/stargate/internal/config"
)

const detailLabelWidth = 11

func (m Model) hostDetailsView() string {
	host, ok := m.SelectedHost()
	if !ok {
		return ""
	}

	width := m.detailWidth
	lines := []string{
		ansi.Truncate(m.styles.detailTitle.Render("Details · "+host.Alias), width, "…"),
		m.detailLine("Host name", valueOrFallback(host.HostName, "—"), width),
		m.detailLine("User", valueOrFallback(host.User, "SSH default"), width),
		m.detailLine("Port", valueOrFallback(host.Port, "SSH default"), width),
		m.detailLine("Identity", valueOrFallback(host.IdentityFile, "SSH default"), width),
		m.detailLine("Source", valueOrFallback(host.SourceFile, "—"), width),
		m.detailLine("Route", hostRoute(host), width),
	}
	return strings.Join(lines, "\n")
}

func (m Model) detailLine(label, value string, width int) string {
	labelText := fmt.Sprintf("%-*s", detailLabelWidth, label)
	valueWidth := max(0, width-detailLabelWidth)
	return m.styles.detailLabel.Render(labelText) + ansi.Truncate(value, valueWidth, "…")
}

func hostRoute(host appconfig.Host) string {
	switch {
	case activeProxyValue(host.ProxyJump):
		return "ProxyJump via " + host.ProxyJump
	case activeProxyValue(host.ProxyCommand):
		return "ProxyCommand: " + host.ProxyCommand
	case host.ViaJump:
		return "via proxy"
	default:
		return "Direct"
	}
}

func activeProxyValue(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && !strings.EqualFold(value, "none")
}

func valueOrFallback(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
