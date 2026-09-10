//go:build linux || darwin

package launcher

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/launchline/launchline/internal/app"
)

func TestManagedProcessHelper(t *testing.T) {
	ready := os.Getenv("LAUNCHLINE_STOP_HELPER_READY")
	if ready == "" {
		return
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM)
	defer signal.Stop(signals)
	if err := os.WriteFile(ready, []byte("ready"), 0600); err != nil {
		t.Fatal(err)
	}
	select {
	case <-signals:
		if err := os.WriteFile(ready+".closed", []byte("closed"), 0600); err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("helper was not closed")
	}
}

func waitForFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", path)
}

func TestStopAfterRestartKeepsOtherWorkspaceRunning(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	l := NewManaged(directory)
	a := app.Application{ID: "a", Name: "Helper", Path: executable, Arguments: []string{"-test.run=^TestManagedProcessHelper$"}}
	first := filepath.Join(t.TempDir(), "first")
	second := filepath.Join(t.TempDir(), "second")
	t.Setenv("LAUNCHLINE_STOP_HELPER_READY", first)
	if err := l.LaunchInWorkspace(context.Background(), "first-workspace", a); err != nil {
		t.Fatal(err)
	}
	waitForFile(t, first)
	t.Setenv("LAUNCHLINE_STOP_HELPER_READY", second)
	if err := l.LaunchInWorkspace(context.Background(), "second-workspace", a); err != nil {
		t.Fatal(err)
	}
	waitForFile(t, second)
	restarted := NewManaged(directory)
	count, err := restarted.StopInWorkspace(context.Background(), "first-workspace", a)
	if err != nil || count != 1 {
		t.Fatalf("first stop: %d %v", count, err)
	}
	waitForFile(t, first+".closed")
	if _, err := os.Stat(second + ".closed"); !os.IsNotExist(err) {
		t.Fatal("stopped another workspace")
	}
	count, err = restarted.StopInWorkspace(context.Background(), "second-workspace", a)
	if err != nil || count != 1 {
		t.Fatalf("second stop: %d %v", count, err)
	}
	waitForFile(t, second+".closed")
	deadline := time.Now().Add(2 * time.Second)
	for {
		count, err = restarted.StopInWorkspace(context.Background(), "first-workspace", a)
		if err != nil {
			t.Fatal(err)
		}
		if count == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("exited helper still reported running")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
