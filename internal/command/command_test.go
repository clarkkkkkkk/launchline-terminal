package command

import (
	"strings"
	"testing"
)

func TestRegistryCommandsAliasesAndQuotedWorkspace(t *testing.T) {
	registry := NewRegistry()
	tests := map[string]Action{
		"/help": ActionHelp, "?": ActionHelp, "/exit": ActionExit,
		"/applications": ActionApplications, "/apps": ActionApplications,
		"/workspaces": ActionWorkspaces, "/workspace School": ActionWorkspace,
		"/start": ActionStart, "/refresh": ActionRefresh, "/settings": ActionSettings,
		"/version": ActionVersion, "/clear": ActionClear, "/add": ActionAdd,
	}
	for input, want := range tests {
		invocation, err := registry.Parse(input)
		if err != nil || invocation.Definition.Action != want {
			t.Fatalf("%q: %#v %v", input, invocation, err)
		}
	}
	invocation, err := registry.Parse(`  /start   "Mobile Development" `)
	if err != nil || len(invocation.Arguments) != 1 || invocation.Arguments[0] != "Mobile Development" {
		t.Fatalf("quoted parse: %#v %v", invocation, err)
	}
}

func TestUnknownSuggestionAndCompletion(t *testing.T) {
	registry := NewRegistry()
	if _, err := registry.Parse("/applicatons"); err == nil || !strings.Contains(err.Error(), "/applications") {
		t.Fatalf("missing suggestion: %v", err)
	}
	completed, matches := registry.Complete("/app", nil)
	if completed != "/applications" || len(matches) != 1 {
		t.Fatalf("command completion: %q %#v", completed, matches)
	}
	completed, _ = registry.Complete("/start mob", []string{"Development", "Mobile Development"})
	if completed != `/start "Mobile Development"` {
		t.Fatalf("workspace completion: %q", completed)
	}
}

func TestHistoryPreservesDraft(t *testing.T) {
	var history History
	history.Add("/help")
	history.Add("/apps")
	if got := history.Previous("/sta"); got != "/apps" {
		t.Fatalf("previous=%q", got)
	}
	if got := history.Previous(""); got != "/help" {
		t.Fatalf("previous=%q", got)
	}
	if got := history.Next(); got != "/apps" {
		t.Fatalf("next=%q", got)
	}
	if got := history.Next(); got != "/sta" {
		t.Fatalf("draft=%q", got)
	}
}
