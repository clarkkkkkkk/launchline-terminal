package app

import (
	"context"
	"errors"
	"sync"
)

// WorkspaceLauncher can retain process ownership across CLI/TUI sessions.
// Implementations must never stop an unrelated or unverified process.
type WorkspaceLauncher interface {
	LaunchInWorkspace(context.Context, string, Application) error
	StopInWorkspace(context.Context, string, Application) (int, error)
}

type StopResult struct {
	Application Application
	Requested   int
	Err         error
}

type StopSummary struct {
	Workspace Workspace
	Results   []StopResult
}

func (s StopSummary) Requested() int {
	count := 0
	for _, result := range s.Results {
		count += result.Requested
	}
	return count
}

func (s StopSummary) Failed() int {
	count := 0
	for _, result := range s.Results {
		if result.Err != nil {
			count++
		}
	}
	return count
}

// Stop requests termination without force-killing applications. A request is
// not proof of exit: a GUI application may still need the user's attention.
func (s *LaunchService) Stop(ctx context.Context, reference string) (StopSummary, error) {
	workspace, cfg, err := s.config.ResolveWorkspace(reference)
	if err != nil {
		return StopSummary{}, err
	}
	launcher, ok := s.launcher.(WorkspaceLauncher)
	if !ok {
		return StopSummary{}, errors.New("this launcher does not support stopping applications")
	}
	byID := make(map[string]Application, len(cfg.Applications))
	for _, item := range cfg.Applications {
		byID[item.ID] = item
	}
	summary := StopSummary{Workspace: workspace, Results: make([]StopResult, len(workspace.Applications))}
	var wg sync.WaitGroup
	for i, id := range workspace.Applications {
		item := byID[id]
		wg.Add(1)
		go func(i int, item Application) {
			defer wg.Done()
			count, err := launcher.StopInWorkspace(ctx, workspace.ID, item)
			summary.Results[i] = StopResult{Application: item, Requested: count, Err: err}
		}(i, item)
	}
	wg.Wait()
	return summary, nil
}
