package app

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Launcher starts one configured application without waiting for it to exit.
type Launcher interface {
	Launch(context.Context, Application) error
}

type LaunchResult struct {
	Application Application
	Err         error
	Duration    time.Duration
}

type LaunchSummary struct {
	Workspace Workspace
	Results   []LaunchResult
}

func (s LaunchSummary) Succeeded() int {
	count := 0
	for _, result := range s.Results {
		if result.Err == nil {
			count++
		}
	}
	return count
}

func (s LaunchSummary) Failed() int { return len(s.Results) - s.Succeeded() }

// LaunchService resolves workspaces and launches their applications. Results
// are independent: one failure never prevents other launch attempts.
type LaunchService struct {
	config   *Service
	launcher Launcher
}

func NewLaunchService(config *Service, launcher Launcher) *LaunchService {
	return &LaunchService{config: config, launcher: launcher}
}

// Begin starts all applications concurrently and returns a buffered result
// stream. The stream closes after every attempt completes.
func (s *LaunchService) Begin(ctx context.Context, reference string) (Workspace, []Application, <-chan LaunchResult, error) {
	workspace, cfg, err := s.config.ResolveWorkspace(reference)
	if err != nil {
		return Workspace{}, nil, nil, err
	}
	byID := make(map[string]Application, len(cfg.Applications))
	for _, item := range cfg.Applications {
		byID[item.ID] = item
	}
	applications := make([]Application, 0, len(workspace.Applications))
	for _, id := range workspace.Applications {
		item, ok := byID[id]
		if !ok {
			return Workspace{}, nil, nil, fmt.Errorf("workspace %q references missing application %q", workspace.Name, id)
		}
		applications = append(applications, item)
	}
	results := make(chan LaunchResult, len(applications))
	var wg sync.WaitGroup
	for _, item := range applications {
		item := item
		wg.Add(1)
		go func() {
			defer wg.Done()
			started := time.Now()
			err := s.launcher.Launch(ctx, item)
			results <- LaunchResult{Application: item, Err: err, Duration: time.Since(started)}
		}()
	}
	go func() {
		wg.Wait()
		close(results)
	}()
	return workspace, applications, results, nil
}

func (s *LaunchService) Start(ctx context.Context, reference string) (LaunchSummary, error) {
	workspace, applications, stream, err := s.Begin(ctx, reference)
	if err != nil {
		return LaunchSummary{}, err
	}
	byID := make(map[string]LaunchResult, len(applications))
	for result := range stream {
		byID[result.Application.ID] = result
	}
	ordered := make([]LaunchResult, 0, len(applications))
	for _, item := range applications {
		ordered = append(ordered, byID[item.ID])
	}
	return LaunchSummary{Workspace: workspace, Results: ordered}, nil
}
