package app

import (
	"context"
	"errors"
	"sync"
	"testing"
)

type managedFake struct {
	mu                sync.Mutex
	launched, stopped map[string]string
}

func (m *managedFake) Launch(context.Context, Application) error {
	return errors.New("untracked launch called")
}
func (m *managedFake) LaunchInWorkspace(_ context.Context, w string, a Application) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.launched[a.ID] = w
	return nil
}
func (m *managedFake) StopInWorkspace(_ context.Context, w string, a Application) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopped[a.ID] = w
	if a.ID == "bad" {
		return 0, errors.New("denied")
	}
	return 1, nil
}

func TestWorkspaceStopIsIndependentAndOrdered(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Applications = []Application{{ID: "good", Name: "Good", Path: "good"}, {ID: "bad", Name: "Bad", Path: "bad"}}
	cfg.Workspaces = []Workspace{{ID: "w", Name: "My Work", Applications: []string{"bad", "good"}}}
	cfg.DefaultWorkspaceID = "w"
	fake := &managedFake{launched: map[string]string{}, stopped: map[string]string{}}
	service := NewLaunchService(NewService(&memoryRepo{cfg: cfg}), fake)
	if _, err := service.Start(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	if fake.launched["good"] != "w" || fake.launched["bad"] != "w" {
		t.Fatal("launch ownership was not passed")
	}
	summary, err := service.Stop(context.Background(), "My Work")
	if err != nil || summary.Failed() != 1 || summary.Requested() != 1 || len(fake.stopped) != 2 || summary.Results[0].Application.ID != "bad" {
		t.Fatalf("summary=%#v err=%v", summary, err)
	}
	if _, err := service.Stop(context.Background(), "unknown"); err == nil {
		t.Fatal("accepted unknown workspace")
	}
}
