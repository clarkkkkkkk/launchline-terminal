package launcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/launchline/launchline/internal/app"
)

type processRecord struct {
	WorkspaceID   string `json:"workspace_id"`
	ApplicationID string `json:"application_id"`
	PID           int    `json:"pid"`
	Identity      string `json:"identity"`
}

// ManagedLauncher records only direct child processes created by Launchline.
// Records are separate from user config and are verified against OS start IDs.
type ManagedLauncher struct {
	*PlatformLauncher
	directory    string
	identity     func(int) (string, error)
	closeProcess func(int, string) error
}

func NewManaged(directory string) *ManagedLauncher {
	return &ManagedLauncher{PlatformLauncher: New(), directory: directory, identity: processIdentity, closeProcess: requestProcessClose}
}

func (l *ManagedLauncher) LaunchInWorkspace(ctx context.Context, workspaceID string, application app.Application) error {
	if err := os.MkdirAll(l.directory, 0700); err != nil {
		return fmt.Errorf("prepare process tracking: %w", err)
	}
	return l.launch(ctx, application, func(process *os.Process) error {
		identity, err := l.identity(process.Pid)
		if errors.Is(err, os.ErrProcessDone) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("application started, but its process could not be tracked: %w", err)
		}
		record := processRecord{WorkspaceID: workspaceID, ApplicationID: application.ID, PID: process.Pid, Identity: identity}
		if err := l.saveRecord(record); err != nil {
			return fmt.Errorf("application started, but its process record could not be saved: %w", err)
		}
		return nil
	})
}

func (l *ManagedLauncher) saveRecord(record processRecord) error {
	file, err := os.CreateTemp(l.directory, ".process-")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if err := json.NewEncoder(file).Encode(record); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), file.Name()+".json")
}

func (l *ManagedLauncher) StopInWorkspace(ctx context.Context, workspaceID string, application app.Application) (int, error) {
	entries, err := os.ReadDir(l.directory)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read process records: %w", err)
	}
	requested := 0
	var failures []error
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return requested, err
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(l.directory, entry.Name())
		data, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			failures = append(failures, err)
			continue
		}
		var record processRecord
		if err := json.Unmarshal(data, &record); err != nil {
			failures = append(failures, fmt.Errorf("invalid process record %s: %w", entry.Name(), err))
			continue
		}
		if record.WorkspaceID != workspaceID || record.ApplicationID != application.ID {
			continue
		}
		if record.PID <= 0 || record.Identity == "" {
			failures = append(failures, fmt.Errorf("invalid process identity in %s", entry.Name()))
			continue
		}
		identity, err := l.identity(record.PID)
		if errors.Is(err, os.ErrProcessDone) || (err == nil && identity != record.Identity) {
			if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				failures = append(failures, err)
			}
			continue
		}
		if err != nil {
			failures = append(failures, fmt.Errorf("verify process %d: %w", record.PID, err))
			continue
		}
		// The platform adapter verifies identity again while obtaining a process
		// handle where supported. Never fall back to matching executable names.
		if err := l.closeProcess(record.PID, record.Identity); err != nil {
			if errors.Is(err, os.ErrProcessDone) {
				_ = os.Remove(path)
				continue
			}
			failures = append(failures, fmt.Errorf("request close for process %d: %w", record.PID, err))
			continue
		}
		requested++
		// Keep the record until exit is observed. A user may cancel a save dialog.
	}
	return requested, errors.Join(failures...)
}
