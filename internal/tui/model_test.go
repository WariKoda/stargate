package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	appconfig "github.com/nibra/stargate/internal/config"
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
	model := New(hosts)
	model = updateModel(t, model, tea.WindowSizeMsg{Width: 100, Height: 30})

	view := model.View()
	for _, expected := range []string{"Stargate", "production", "deploy@prod.example.com:22", "via Jump-Host"} {
		if !strings.Contains(view, expected) {
			t.Errorf("View() does not contain %q", expected)
		}
	}

	empty := New(nil)
	if view := empty.View(); !strings.Contains(view, "No Hosts.") {
		t.Errorf("empty View() = %q, want No Hosts.", view)
	}
}

func TestModelNavigationAndFuzzyFilter(t *testing.T) {
	t.Parallel()

	model := New([]appconfig.Host{
		{Alias: "development", HostName: "dev.example.com", Port: "22"},
		{Alias: "production", HostName: "prod.example.com", Port: "22"},
		{Alias: "jumpbox", HostName: "gateway.example.com", Port: "22"},
	})

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

	model := New([]appconfig.Host{{Alias: "server", HostName: "example.com", Port: "22"}})
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if model.list.FilterState() != list.Filtering {
		t.Fatalf("FilterState() = %s, want filtering", model.list.FilterState())
	}
}

func TestModelEnterRequestsSelectedHost(t *testing.T) {
	t.Parallel()

	model := New([]appconfig.Host{{Alias: "production", HostName: "prod.example.com", Port: "22"}})
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = modelFromTea(t, updated)
	if cmd == nil {
		t.Fatal("Enter returned nil command")
	}

	message, ok := cmd().(connectRequestedMsg)
	if !ok {
		t.Fatalf("Enter command returned %T, want connectRequestedMsg", cmd())
	}
	model = updateModel(t, model, message)

	requested, ok := model.RequestedHost()
	if !ok || requested.Alias != "production" {
		t.Fatalf("RequestedHost() = %#v, %t; want production", requested, ok)
	}
	if !strings.Contains(model.View(), "Bereit zum Verbinden: production") {
		t.Error("View() does not show selection status")
	}
}

func TestModelEnterDoesNotConnectWhileFiltering(t *testing.T) {
	t.Parallel()

	model := New([]appconfig.Host{{Alias: "server", HostName: "example.com", Port: "22"}})
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	model = updateModel(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	if _, ok := model.RequestedHost(); ok {
		t.Fatal("Enter requested a host while filter input was active")
	}
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
