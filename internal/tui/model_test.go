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
	model := New(hosts, nil)
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

	empty := New(nil, nil)
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
	}, nil)

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

func TestModelSlashStartsFiltering(t *testing.T) {
	t.Parallel()

	model := New([]appconfig.Host{{Alias: "server", HostName: "example.com", Port: "22"}}, nil)
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if model.list.FilterState() != list.Filtering {
		t.Fatalf("FilterState() = %s, want filtering", model.list.FilterState())
	}
}

func TestModelRunsAndReturnsFromSSHProcess(t *testing.T) {
	t.Parallel()

	connector := &stubConnector{}
	model := New([]appconfig.Host{{Alias: "production", HostName: "prod.example.com", Port: "22"}}, connector)
	var command *exec.Cmd
	model.runProcess = func(cmd *exec.Cmd, callback tea.ExecCallback) tea.Cmd {
		command = cmd
		return func() tea.Msg { return callback(nil) }
	}

	updated, requestCmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = modelFromTea(t, updated)
	if requestCmd == nil {
		t.Fatal("Enter returned nil command")
	}
	request, ok := requestCmd().(connectRequestedMsg)
	if !ok {
		t.Fatal("Enter command did not return connectRequestedMsg")
	}

	updated, processCmd := model.Update(request)
	model = modelFromTea(t, updated)
	if processCmd == nil || command == nil {
		t.Fatal("connect request did not create a process command")
	}
	if !model.Connecting() {
		t.Error("Connecting() = false while SSH process is active")
	}
	if got := command.Args; len(got) != 3 || got[2] != "production" {
		t.Errorf("SSH command args = %#v, want selected alias", got)
	}

	finished, ok := processCmd().(connectionFinishedMsg)
	if !ok {
		t.Fatal("process command did not return connectionFinishedMsg")
	}
	model = updateModel(t, model, finished)
	if model.Connecting() {
		t.Error("Connecting() = true after SSH process ended")
	}
	requested, ok := model.RequestedHost()
	if !ok || requested.Alias != "production" {
		t.Fatalf("RequestedHost() = %#v, %t; want production", requested, ok)
	}
	if !strings.Contains(model.View(), "SSH-Sitzung beendet: production") {
		t.Error("View() does not show completed SSH session status")
	}
}

func TestModelShowsConnectorAndExitErrors(t *testing.T) {
	t.Parallel()

	host := appconfig.Host{Alias: "broken", HostName: "broken.example.com", Port: "22"}
	model := New([]appconfig.Host{host}, &stubConnector{err: errors.New("cannot build command")})
	model = updateModel(t, model, connectRequestedMsg{host: host})
	if !strings.Contains(model.View(), "cannot build command") {
		t.Error("View() does not show connector error")
	}

	exitError := exec.Command("sh", "-c", "exit 23").Run()
	model = updateModel(t, model, connectionFinishedMsg{host: host, err: exitError})
	if !strings.Contains(model.View(), "Exit-Code 23") {
		t.Error("View() does not show SSH exit code")
	}
}

func TestModelHandlesMissingConnector(t *testing.T) {
	t.Parallel()

	host := appconfig.Host{Alias: "server", HostName: "example.com", Port: "22"}
	model := New([]appconfig.Host{host}, nil)
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
	}, &stubConnector{})

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
	}, &stubConnector{})

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

	model := New([]appconfig.Host{{Alias: "server", HostName: "example.com", Port: "22"}}, nil)
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

func (s *stubConnector) Command(alias string) (*exec.Cmd, error) {
	if s.err != nil {
		return nil, s.err
	}
	return exec.Command("ssh", "--", alias), nil
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
