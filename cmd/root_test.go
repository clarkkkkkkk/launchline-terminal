package cmd

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/launchline/launchline/internal/app"
	"github.com/launchline/launchline/internal/config"
)

type recordingLauncher struct {
	mu      sync.Mutex
	started []app.Application
	failed  string
}

func (r *recordingLauncher) Launch(_ context.Context, item app.Application) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.started = append(r.started, item)
	if item.Name == r.failed {
		return context.DeadlineExceeded
	}
	return nil
}

func testDependencies(t *testing.T) (Dependencies, *recordingLauncher) {
	t.Helper()
	repo := config.NewFileRepository(filepath.Join(t.TempDir(), "config.json"))
	service := app.NewService(repo)
	recorder := &recordingLauncher{}
	return Dependencies{Config: service, Launch: app.NewLaunchService(service, recorder), RunTUI: func(*app.Service, *app.LaunchService) error { return nil }}, recorder
}

func executeForTest(t *testing.T, deps Dependencies, args ...string) (string, error) {
	t.Helper()
	var output bytes.Buffer
	command := NewRootCommand(deps, &output, &output)
	command.SetArgs(args)
	err := command.Execute()
	return output.String(), err
}

func TestCommandWorkflow(t *testing.T) {
	deps, recorder := testDependencies(t)
	if output, err := executeForTest(t, deps, "add", "--name", "Editor", "--path", "/opt/editor", "--arg=--new-window"); err != nil || !strings.Contains(output, "Added Editor") {
		t.Fatalf("add: %q %v", output, err)
	}
	if output, err := executeForTest(t, deps, "workspace", "create", "--name", "Development", "--app", "Editor", "--default"); err != nil || !strings.Contains(output, "Created workspace Development") {
		t.Fatalf("workspace create: %q %v", output, err)
	}
	output, err := executeForTest(t, deps, "start", "development")
	if err != nil || !strings.Contains(output, "✓ Editor") || !strings.Contains(output, "1 applications launched") {
		t.Fatalf("start: %q %v", output, err)
	}
	recorder.mu.Lock()
	started := len(recorder.started)
	recorder.mu.Unlock()
	if started != 1 {
		t.Fatalf("launched %d applications", started)
	}
	if output, err := executeForTest(t, deps, "config"); err != nil || !strings.Contains(output, "Applications: 1") || !strings.Contains(output, "Workspaces: 1") {
		t.Fatalf("config: %q %v", output, err)
	}
}

func TestStartReportsPartialFailure(t *testing.T) {
	deps, recorder := testDependencies(t)
	_, _ = executeForTest(t, deps, "add", "--name", "Good", "--path", "good")
	_, _ = executeForTest(t, deps, "add", "--name", "Bad", "--path", "bad")
	_, _ = executeForTest(t, deps, "workspace", "create", "--name", "Work", "--app", "Good", "--app", "Bad")
	recorder.failed = "Bad"
	output, err := executeForTest(t, deps, "start")
	if err == nil || !strings.Contains(output, "1 applications failed") || !strings.Contains(output, "× Bad") {
		t.Fatalf("partial failure: %q %v", output, err)
	}
	recorder.mu.Lock()
	started := len(recorder.started)
	recorder.mu.Unlock()
	if started != 2 {
		t.Fatalf("expected both attempts, got %d", started)
	}
}

func TestHelpVersionAndDashboardEntry(t *testing.T) {
	deps, _ := testDependencies(t)
	opened := false
	deps.RunTUI = func(*app.Service, *app.LaunchService) error { opened = true; return nil }
	if _, err := executeForTest(t, deps); err != nil || !opened {
		t.Fatalf("dashboard did not open: %v", err)
	}
	if output, err := executeForTest(t, deps, "help"); err != nil || !strings.Contains(output, "One command. Your entire workspace.") {
		t.Fatalf("help: %q %v", output, err)
	}
	if output, err := executeForTest(t, deps, "version"); err != nil || !strings.Contains(output, "launchline dev") {
		t.Fatalf("version: %q %v", output, err)
	}
}
