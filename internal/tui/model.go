package tui

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	appconfig "github.com/warikoda/stargate/internal/config"
)

const (
	defaultWidth    = 80
	defaultHeight   = 24
	hostItemHeight  = 2
	hostItemSpacing = 1
)

// SessionResume carries SSH session results into the next TUI iteration.
type SessionResume struct {
	Host appconfig.Host
	Err  error
}

// Model is the root Bubble Tea model for Stargate.
type Model struct {
	list           list.Model
	keys           keyMap
	styles         styles
	connector      Connector
	checker        ConnectivityChecker
	checkOnStart   bool
	requestedHost  *appconfig.Host
	pendingConnect *appconfig.Host
	resumeMessage  string
	showLogo       bool
}

// Connector builds the native SSH command for a selected host alias.
type Connector interface {
	Command(alias string) (*exec.Cmd, error)
	Preview(alias string) (string, error)
}

// ConnectivityChecker tests a host endpoint without opening an SSH session.
type ConnectivityChecker interface {
	Check(host appconfig.Host) error
}

// ConnectivityOptions configures manual and automatic TCP checks.
type ConnectivityOptions struct {
	Checker      ConnectivityChecker
	CheckOnStart bool
}

type connectivityState uint8

const (
	connectivityUnchecked connectivityState = iota
	connectivityChecking
	connectivityReachable
	connectivityUnavailable
)

type hostItem struct {
	host         appconfig.Host
	connectivity connectivityState
}

func (i hostItem) Title() string {
	return i.host.Alias
}

func (i hostItem) Description() string {
	address := i.host.HostName
	if strings.Contains(address, ":") && !strings.HasPrefix(address, "[") {
		address = "[" + address + "]"
	}
	if i.host.User != "" {
		address = i.host.User + "@" + address
	}
	if i.host.Port != "" {
		address += ":" + i.host.Port
	}
	if i.host.ViaJump {
		address += "  • via Jump-Host"
	}
	switch i.connectivity {
	case connectivityChecking:
		address += "  • checking TCP…"
	case connectivityReachable:
		address += "  • TCP reachable"
	case connectivityUnavailable:
		address += "  • TCP unavailable"
	}
	return address
}

func (i hostItem) FilterValue() string {
	return strings.Join([]string{
		i.host.Alias,
		i.host.HostName,
		i.host.User,
		i.host.Port,
	}, " ")
}

type connectRequestedMsg struct {
	host appconfig.Host
}

type showResumeStatusMsg struct {
	message string
}

type connectivityCheckedMsg struct {
	alias string
	err   error
}

type commandCopiedMsg struct {
	preview string
	err     error
}

// New creates a Stargate TUI model for the given SSH hosts and connector.
func New(hosts []appconfig.Host, connector Connector, resume *SessionResume, connectivityOptions ...ConnectivityOptions) Model {
	appStyles := newStyles()
	keys := newKeyMap()
	delegate := newHostDelegate(appStyles)
	var connectivity ConnectivityOptions
	if len(connectivityOptions) > 0 {
		connectivity = connectivityOptions[0]
	}

	items := make([]list.Item, 0, len(hosts))
	for _, host := range hosts {
		state := connectivityUnchecked
		if connectivity.Checker != nil && connectivity.CheckOnStart {
			state = connectivityChecking
		}
		items = append(items, hostItem{host: host, connectivity: state})
	}

	hostList := list.New(items, delegate, defaultWidth, defaultHeight)
	configureListKeyMap(&hostList)
	hostList.Title = "Stargate"
	hostList.StatusMessageLifetime = 5 * time.Second
	hostList.SetStatusBarItemName("Host", "connectable hosts")
	hostList.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{keys.connect, keys.preview}
	}
	hostList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{keys.connect, keys.preview, keys.copy, keys.check}
	}
	applyListStyles(&hostList, appStyles)

	model := Model{
		list:         hostList,
		keys:         keys,
		styles:       appStyles,
		connector:    connector,
		checker:      connectivity.Checker,
		checkOnStart: connectivity.CheckOnStart,
	}
	if resume != nil {
		host := resume.Host
		model.requestedHost = &host
		model.resumeMessage = connectionFinishedStatus(host, resume.Err)
		model.selectHostByAlias(host.Alias)
		model.list.ResetFilter()
	}
	model.resize(defaultWidth, defaultHeight)

	return model
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd {
	commands := make([]tea.Cmd, 0, len(m.list.Items())+1)
	if m.resumeMessage != "" {
		message := m.resumeMessage
		commands = append(commands, func() tea.Msg {
			return showResumeStatusMsg{message: message}
		})
	}
	if m.checker != nil && m.checkOnStart {
		for _, item := range m.list.Items() {
			host, ok := item.(hostItem)
			if ok {
				commands = append(commands, checkConnectivity(m.checker, host.host))
			}
		}
	}
	return tea.Batch(commands...)
}

// SelectedHost returns the currently highlighted SSH host.
func (m Model) SelectedHost() (appconfig.Host, bool) {
	item, ok := m.list.SelectedItem().(hostItem)
	if !ok {
		return appconfig.Host{}, false
	}
	return item.host, true
}

// RequestedHost returns the host most recently requested with Enter.
func (m Model) RequestedHost() (appconfig.Host, bool) {
	if m.requestedHost == nil {
		return appconfig.Host{}, false
	}
	return *m.requestedHost, true
}

// PendingConnect reports whether the TUI quit to hand off to an SSH session.
func (m Model) PendingConnect() (appconfig.Host, bool) {
	if m.pendingConnect == nil {
		return appconfig.Host{}, false
	}
	return *m.pendingConnect, true
}

func (m *Model) selectHostByAlias(alias string) {
	for index, item := range m.list.Items() {
		host, ok := item.(hostItem)
		if !ok || host.host.Alias != alias {
			continue
		}
		m.list.Select(index)
		return
	}
}

func checkConnectivity(checker ConnectivityChecker, host appconfig.Host) tea.Cmd {
	return func() tea.Msg {
		return connectivityCheckedMsg{alias: host.Alias, err: checker.Check(host)}
	}
}

func requestConnection(host appconfig.Host) tea.Cmd {
	return func() tea.Msg {
		return connectRequestedMsg{host: host}
	}
}

func connectionFinishedStatus(host appconfig.Host, err error) string {
	if err == nil {
		return fmt.Sprintf("SSH-Sitzung beendet: %s", host.Alias)
	}

	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return fmt.Sprintf("SSH fehlgeschlagen: %s (Exit-Code %d)", host.Alias, exitError.ExitCode())
	}
	return fmt.Sprintf("SSH fehlgeschlagen: %s (%v)", host.Alias, err)
}
