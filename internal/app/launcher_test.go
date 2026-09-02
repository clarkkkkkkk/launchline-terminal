package app

import (
	"context"
	"errors"
	"testing"
)

type fakeLauncher struct{ failID string }

func (f fakeLauncher) Launch(_ context.Context, application Application) error {
	if application.ID == f.failID {
		return errors.New("simulated failure")
	}
	return nil
}

func TestLaunchContinuesAfterApplicationFailure(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Applications = []Application{{ID: "a", Name: "A", Path: "a"}, {ID: "b", Name: "B", Path: "b"}, {ID: "c", Name: "C", Path: "c"}}
	cfg.Workspaces = []Workspace{{ID: "w", Name: "Work", Applications: []string{"a", "b", "c"}}}
	cfg.DefaultWorkspaceID = "w"
	service := NewService(&memoryRepo{cfg: cfg})
	summary, err := NewLaunchService(service, fakeLauncher{failID: "b"}).Start(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if summary.Succeeded() != 2 || summary.Failed() != 1 || len(summary.Results) != 3 {
		t.Fatalf("unexpected summary: %#v", summary)
	}
	if summary.Results[1].Err == nil {
		t.Fatal("failed application result was lost")
	}
}
