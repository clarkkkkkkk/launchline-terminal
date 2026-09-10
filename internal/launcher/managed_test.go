package launcher

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/launchline/launchline/internal/app"
)

func TestStopVerifiesOwnershipAndIdentity(t *testing.T) {
	for _, tt := range []struct {
		name, workspace, application, current string
		identityErr                           error
		wantClose                             bool
	}{
		{name: "owned", workspace: "w", application: "a", current: "original", wantClose: true},
		{name: "other workspace", workspace: "other", application: "a", current: "original"},
		{name: "other application", workspace: "w", application: "other", current: "original"},
		{name: "reused PID", workspace: "w", application: "a", current: "new process"},
		{name: "exited", workspace: "w", application: "a", identityErr: os.ErrProcessDone},
	} {
		t.Run(tt.name, func(t *testing.T) {
			l := NewManaged(t.TempDir())
			if err := l.saveRecord(processRecord{WorkspaceID: tt.workspace, ApplicationID: tt.application, PID: 123, Identity: "original"}); err != nil {
				t.Fatal(err)
			}
			l.identity = func(int) (string, error) { return tt.current, tt.identityErr }
			closed := false
			l.closeProcess = func(pid int, identity string) error {
				closed = true
				if pid != 123 || identity != "original" {
					t.Fatal("wrong process")
				}
				return nil
			}
			requested, err := l.StopInWorkspace(context.Background(), "w", app.Application{ID: "a"})
			if err != nil || closed != tt.wantClose || (requested > 0) != tt.wantClose {
				t.Fatalf("requested=%d closed=%v err=%v", requested, closed, err)
			}
		})
	}
}

func TestStopFailuresRemainRetryable(t *testing.T) {
	l := NewManaged(t.TempDir())
	if err := l.saveRecord(processRecord{WorkspaceID: "w", ApplicationID: "a", PID: 123, Identity: "original"}); err != nil {
		t.Fatal(err)
	}
	l.identity = func(int) (string, error) { return "original", nil }
	l.closeProcess = func(int, string) error { return errors.New("permission denied") }
	if _, err := l.StopInWorkspace(context.Background(), "w", app.Application{ID: "a"}); err == nil {
		t.Fatal("missing close error")
	}
	entries, _ := os.ReadDir(l.directory)
	if len(entries) != 1 {
		t.Fatal("failed close removed process ownership")
	}
	if err := os.WriteFile(filepath.Join(l.directory, "corrupt.json"), []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := l.StopInWorkspace(context.Background(), "w", app.Application{ID: "a"}); err == nil {
		t.Fatal("ignored corrupt state")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := l.StopInWorkspace(ctx, "w", app.Application{ID: "a"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
}
