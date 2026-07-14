package tui

import (
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	appconfig "github.com/nibra180/stargate-tui/internal/config"
)

func TestHostItemPresentationAndFilterValue(t *testing.T) {
	t.Parallel()

	item := hostItem{host: appconfig.Host{
		Alias:    "production",
		HostName: "2001:db8::10",
		User:     "deploy",
		Port:     "2200",
		ViaJump:  true,
	}}

	if got := item.Title(); got != "production" {
		t.Errorf("Title() = %q, want production", got)
	}
	if got := item.Description(); got != "deploy@[2001:db8::10]:2200  • via Jump-Host" {
		t.Errorf("Description() = %q", got)
	}
	for _, value := range []string{"production", "2001:db8::10", "deploy", "2200"} {
		if !strings.Contains(item.FilterValue(), value) {
			t.Errorf("FilterValue() = %q, want %q", item.FilterValue(), value)
		}
	}
}

func TestModelRendersHostListAndEmptyState(t *testing.T) {
	t.Parallel()

	hosts := []appconfig.Host{{
		Alias:    "production",
		HostName: "prod.example.com",
		User:     "deploy",
		Port:     "22",
		ViaJump:  true,
	}}
	model := New(hosts, nil, nil)
	model = updateModel(t, model, tea.WindowSizeMsg{Width: 100, Height: 30})

	view := model.View()
	for _, expected := range []string{
		"Stargate",
		"production",
		"deploy@prod.example.com:22",
		"via Jump-Host",
		"filter",
		"connect",
		"click",
	} {
		if !strings.Contains(view, expected) {
			t.Errorf("View() does not contain %q", expected)
		}
	}

	empty := New(nil, nil, nil)
	if view := empty.View(); !strings.Contains(view, "No connectable hosts.") {
		t.Errorf("empty View() = %q, want clear empty state", view)
	}
}

func TestModelNavigationAndFuzzyFilter(t *testing.T) {
	t.Parallel()

	model := New([]appconfig.Host{
		{Alias: "development", HostName: "dev.example.com", Port: "22"},
		{Alias: "production", HostName: "prod.example.com", Port: "22"},
		{Alias: "jumpbox", HostName: "gateway.example.com", Port: "22"},
	}, nil, nil)

	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyDown})
	selected, ok := model.SelectedHost()
	if !ok || selected.Alias != "production" {
		t.Fatalf("SelectedHost() = %#v, %t; want production", selected, ok)
	}

	model.list.SetFilterText("gateway")
	if model.list.FilterState() != list.FilterApplied {
		t.Fatalf("FilterState() = %s, want filter applied", model.list.FilterState())
	}
	visible := model.list.VisibleItems()
	if len(visible) != 1 {
		t.Fatalf("len(VisibleItems()) = %d, want 1", len(visible))
	}
	filtered, ok := visible[0].(hostItem)
	if !ok || filtered.host.Alias != "jumpbox" {
		t.Fatalf("filtered item = %#v, want jumpbox", visible[0])
	}
}

func TestModelDirectTypingStartsAndExtendsFilter(t *testing.T) {
	t.Parallel()

	for _, firstRune := range []rune{'p', 'j', 'k', 'q'} {
		model := New([]appconfig.Host{{Alias: "project", HostName: "example.com", Port: "22"}}, nil, nil)
		model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{firstRune}})
		if model.list.FilterState() != list.Filtering {
			t.Errorf("typing %q left filter state at %s", firstRune, model.list.FilterState())
		}
		if got := model.list.FilterValue(); got != string(firstRune) {
			t.Errorf("typing %q set filter to %q", firstRune, got)
		}
	}

	model := New([]appconfig.Host{{Alias: "project", HostName: "example.com", Port: "22"}}, nil, nil)
	model.list.SetFilterText("pro")
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if model.list.FilterState() != list.Filtering || model.list.FilterValue() != "prod" {
		t.Errorf("typing after applied filter produced state=%s value=%q", model.list.FilterState(), model.list.FilterValue())
	}
}

func TestModelSlashStillStartsFiltering(t *testing.T) {
	t.Parallel()

	model := New([]appconfig.Host{{Alias: "server", HostName: "example.com", Port: "22"}}, nil, nil)
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if model.list.FilterState() != list.Filtering {
		t.Fatalf("FilterState() = %s, want filtering", model.list.FilterState())
	}
	if model.list.FilterValue() != "" {
		t.Errorf("slash shortcut set filter value to %q, want empty", model.list.FilterValue())
	}
}

func TestModelHelpShortcutDoesNotStartFilter(t *testing.T) {
	t.Parallel()

	model := New([]appconfig.Host{{Alias: "server", HostName: "example.com", Port: "22"}}, nil, nil)
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if !model.list.Help.ShowAll {
		t.Error("? did not open extended help")
	}
	if model.list.FilterState() != list.Unfiltered {
		t.Errorf("? changed filter state to %s", model.list.FilterState())
	}
}

func TestModelPreviewsAndCopiesSelectedCommand(t *testing.T) {
	t.Parallel()

	model := New([]appconfig.Host{{Alias: "production", HostName: "prod.example.com", Port: "22"}}, &stubConnector{}, nil)
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyCtrlP})
	if !strings.Contains(model.View(), "ssh -- production") {
		t.Error("command preview is not shown")
	}

	var copied string
	message, ok := copyCommand("ssh -- production", func(value string) error {
		copied = value
		return nil
	})().(commandCopiedMsg)
	if !ok || message.err != nil || copied != "ssh -- production" {
		t.Fatalf("copyCommand() = %#v, copied %q", message, copied)
	}
	model = updateModel(t, model, message)
	if !strings.Contains(model.View(), "Copied: ssh -- production") {
		t.Error("successful copy status is not shown")
	}
}

func TestModelRunsOptInConnectivityChecks(t *testing.T) {
	t.Parallel()

	checker := &stubConnectivityChecker{}
	model := New(
		[]appconfig.Host{{Alias: "production", HostName: "prod.example.com", Port: "22"}},
		&stubConnector{},
		nil,
		ConnectivityOptions{Checker: checker, CheckOnStart: true},
	)
	if !strings.Contains(model.View(), "checking TCP") {
		t.Error("pending TCP check is not shown")
	}

	command := model.Init()
	if command == nil {
		t.Fatal("Init() did not start the connectivity check")
	}
	message, ok := command().(connectivityCheckedMsg)
	if !ok {
		t.Fatalf("connectivity command returned %T", command())
	}
	model = updateModel(t, model, message)
	if checker.alias != "production" || !strings.Contains(model.View(), "TCP reachable") {
		t.Errorf("connectivity result missing; checked %q, view %q", checker.alias, model.View())
	}

	model = updateModel(t, model, connectivityCheckedMsg{alias: "production", err: errors.New("refused")})
	if !strings.Contains(model.View(), "TCP unavailable") {
		t.Error("failed TCP check is not shown")
	}
}

func TestModelStartsSelectedConnectivityCheckFromTUI(t *testing.T) {
	t.Parallel()

	checker := &stubConnectivityChecker{}
	model := New(
		[]appconfig.Host{{Alias: "production", HostName: "prod.example.com", Port: "22"}},
		&stubConnector{},
		nil,
		ConnectivityOptions{Checker: checker},
	)
	if strings.Contains(model.View(), "checking TCP") {
		t.Error("manual connectivity check started before the shortcut")
	}

	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	model = modelFromTea(t, updated)
	if command == nil || !strings.Contains(model.View(), "checking TCP") {
		t.Fatal("Ctrl+R did not start the selected host check")
	}
	message, ok := command().(connectivityCheckedMsg)
	if !ok || message.alias != "production" || checker.alias != "production" {
		t.Fatalf("Ctrl+R check = %#v, checked %q", message, checker.alias)
	}
	model = updateModel(t, model, message)
	if !strings.Contains(model.View(), "TCP reachable") {
		t.Error("manual connectivity result is not shown")
	}
}

func TestModelQuitsToConnect(t *testing.T) {
	t.Parallel()

	connector := &stubConnector{}
	model := New([]appconfig.Host{{Alias: "production", HostName: "prod.example.com", Port: "22"}}, connector, nil)

	updated, requestCmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = modelFromTea(t, updated)
	if requestCmd == nil {
		t.Fatal("Enter returned nil command")
	}
	request, ok := requestCmd().(connectRequestedMsg)
	if !ok {
		t.Fatal("Enter command did not return connectRequestedMsg")
	}

	updated, quitCmd := model.Update(request)
	model = modelFromTea(t, updated)
	if quitCmd == nil {
		t.Fatal("connect request did not return quit command")
	}
	if _, ok := quitCmd().(tea.QuitMsg); !ok {
		t.Fatalf("connect request returned %T, want tea.QuitMsg", quitCmd())
	}

	host, ok := model.PendingConnect()
	if !ok || host.Alias != "production" {
		t.Fatalf("PendingConnect() = %#v, %t; want production", host, ok)
	}
	requested, ok := model.RequestedHost()
	if !ok || requested.Alias != "production" {
		t.Fatalf("RequestedHost() = %#v, %t; want production", requested, ok)
	}
}

func TestModelClearsFilterAfterSSHSession(t *testing.T) {
	t.Parallel()

	hosts := []appconfig.Host{
		{Alias: "development", HostName: "dev.example.com", Port: "22"},
		{Alias: "production", HostName: "prod.example.com", Port: "22"},
	}
	model := New(hosts, &stubConnector{}, &SessionResume{Host: hosts[1]})
	if model.list.FilterState() != list.Unfiltered {
		t.Errorf("FilterState() = %s, want unfiltered", model.list.FilterState())
	}
	if model.list.FilterValue() != "" {
		t.Errorf("FilterValue() = %q, want empty", model.list.FilterValue())
	}
	if len(model.list.VisibleItems()) != len(hosts) {
		t.Errorf("len(VisibleItems()) = %d, want %d", len(model.list.VisibleItems()), len(hosts))
	}
	selected, ok := model.SelectedHost()
	if !ok || selected.Alias != "production" {
		t.Errorf("SelectedHost() = %#v, %t; want production", selected, ok)
	}
}

func TestModelShowsConnectorAndExitErrors(t *testing.T) {
	t.Parallel()

	host := appconfig.Host{Alias: "broken", HostName: "broken.example.com", Port: "22"}
	model := New([]appconfig.Host{host}, &stubConnector{err: errors.New("cannot build command")}, nil)
	model = updateModel(t, model, connectRequestedMsg{host: host})
	if !strings.Contains(model.View(), "cannot build command") {
		t.Error("View() does not show connector error")
	}

	model = New([]appconfig.Host{host}, &stubConnector{}, &SessionResume{
		Host: host,
		Err:  exec.Command("sh", "-c", "exit 23").Run(),
	})
	model = updateModel(t, model, tea.WindowSizeMsg{Width: 100, Height: 30})
	if cmd := model.Init(); cmd != nil {
		model = updateModel(t, model, cmd())
	}
	if !strings.Contains(model.View(), "Exit-Code 23") {
		t.Error("View() does not show SSH exit code")
	}
}

func TestModelHandlesMissingConnector(t *testing.T) {
	t.Parallel()

	host := appconfig.Host{Alias: "server", HostName: "example.com", Port: "22"}
	model := New([]appconfig.Host{host}, nil, nil)
	model = updateModel(t, model, connectRequestedMsg{host: host})
	if !strings.Contains(model.View(), "kein Executor") {
		t.Error("View() does not show missing executor error")
	}
}

func TestModelMouseWheelNavigatesAndClickConnects(t *testing.T) {
	t.Parallel()

	model := New([]appconfig.Host{
		{Alias: "first", HostName: "first.example.com", Port: "22"},
		{Alias: "second", HostName: "second.example.com", Port: "22"},
	}, &stubConnector{}, nil)

	model = updateModel(t, model, mouseMessage(3, 3, tea.MouseButtonWheelDown))
	selected, ok := model.SelectedHost()
	if !ok || selected.Alias != "second" {
		t.Fatalf("mouse wheel selected %#v, want second", selected)
	}
	model = updateModel(t, model, mouseMessage(3, 3, tea.MouseButtonWheelUp))

	updated, cmd := model.Update(mouseMessage(3, 6, tea.MouseButtonLeft))
	model = modelFromTea(t, updated)
	selected, ok = model.SelectedHost()
	if !ok || selected.Alias != "second" {
		t.Fatalf("mouse click selected %#v, want second", selected)
	}
	if cmd == nil {
		t.Fatal("mouse click returned nil connection command")
	}
	request, ok := cmd().(connectRequestedMsg)
	if !ok || request.host.Alias != "second" {
		t.Fatalf("mouse click request = %#v, want second", request)
	}
}

func TestModelMouseIgnoresSpacingAndOutsideClicks(t *testing.T) {
	t.Parallel()

	model := New([]appconfig.Host{
		{Alias: "first", HostName: "first.example.com", Port: "22"},
		{Alias: "second", HostName: "second.example.com", Port: "22"},
	}, &stubConnector{}, nil)

	for _, message := range []tea.MouseMsg{
		mouseMessage(3, 5, tea.MouseButtonLeft), // spacing after the first item
		mouseMessage(0, 3, tea.MouseButtonLeft), // outside the app's left padding
		mouseMessage(3, 1, tea.MouseButtonLeft), // title bar
	} {
		updated, cmd := model.Update(message)
		model = modelFromTea(t, updated)
		if cmd != nil {
			t.Fatal("non-item mouse click returned a command")
		}
	}

	selected, ok := model.SelectedHost()
	if !ok || selected.Alias != "first" {
		t.Fatalf("non-item clicks selected %#v, want first", selected)
	}
}

func TestModelEnterDoesNotConnectWhileFiltering(t *testing.T) {
	t.Parallel()

	model := New([]appconfig.Host{{Alias: "server", HostName: "example.com", Port: "22"}}, nil, nil)
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	if _, ok := model.RequestedHost(); ok {
		t.Fatal("Enter requested a host while filter input was active")
	}
}

func mouseMessage(x, y int, button tea.MouseButton) tea.MouseMsg {
	return tea.MouseMsg(tea.MouseEvent{
		X:      x,
		Y:      y,
		Action: tea.MouseActionPress,
		Button: button,
	})
}

type stubConnector struct {
	err error
}

type stubConnectivityChecker struct {
	alias string
	err   error
}

func (s *stubConnectivityChecker) Check(host appconfig.Host) error {
	s.alias = host.Alias
	return s.err
}

func (s *stubConnector) Command(alias string) (*exec.Cmd, error) {
	if s.err != nil {
		return nil, s.err
	}
	return exec.Command("ssh", "--", alias), nil
}

func (s *stubConnector) Preview(alias string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return "ssh -- " + alias, nil
}

func updateModel(t *testing.T, model Model, msg tea.Msg) Model {
	t.Helper()
	updated, _ := model.Update(msg)
	return modelFromTea(t, updated)
}

func modelFromTea(t *testing.T, model tea.Model) Model {
	t.Helper()
	updated, ok := model.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want tui.Model", model)
	}
	return updated
}
