package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/launchline/launchline/internal/app"
)

type FileRepository struct{ path string }

func NewFileRepository(path string) *FileRepository { return &FileRepository{path: path} }

func NewDefaultRepository() (*FileRepository, error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, err
	}
	return NewFileRepository(path), nil
}

func (r *FileRepository) Path() string { return r.path }

func (r *FileRepository) Load() (app.Config, error) {
	data, err := os.ReadFile(r.path)
	if errors.Is(err, os.ErrNotExist) {
		previous := r.path + ".previous"
		if _, previousErr := os.Stat(previous); previousErr == nil {
			if renameErr := os.Rename(previous, r.path); renameErr != nil {
				return app.Config{}, fmt.Errorf("recover interrupted configuration write from %s: %w", previous, renameErr)
			}
			return r.Load()
		}
		cfg := app.DefaultConfig()
		if err := r.Save(cfg); err != nil {
			return app.Config{}, fmt.Errorf("create initial configuration: %w", err)
		}
		return cfg, nil
	}
	if err != nil {
		return app.Config{}, fmt.Errorf("read configuration %s: %w", r.path, err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return app.Config{}, r.corrupt(data, errors.New("file is empty"))
	}
	var cfg app.Config
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return app.Config{}, r.corrupt(data, fmt.Errorf("decode JSON: %w", err))
	}
	if err := ensureEOF(decoder); err != nil {
		return app.Config{}, r.corrupt(data, err)
	}
	migrated := cfg.Version == 1
	if migrated {
		// v2 is additive. Existing IDs, workspace membership, default choice,
		// paths, arguments, and compact-logo preference remain unchanged.
		cfg.Version = app.CurrentSchemaVersion
	}
	if cfg.Applications == nil {
		cfg.Applications = []app.Application{}
	}
	if cfg.Workspaces == nil {
		cfg.Workspaces = []app.Workspace{}
	}
	if err := app.ValidateConfig(cfg); err != nil {
		return app.Config{}, r.corrupt(data, err)
	}
	if migrated {
		if err := r.Save(cfg); err != nil {
			return app.Config{}, fmt.Errorf("migrate configuration from version 1: %w", err)
		}
	}
	return cfg, nil
}

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("configuration contains multiple JSON values")
		}
		return fmt.Errorf("decode trailing JSON: %w", err)
	}
	return nil
}

func (r *FileRepository) corrupt(data []byte, cause error) error {
	stamp := time.Now().UTC().Format("20060102T150405.000000000Z")
	backup := r.path + ".corrupt-" + stamp
	if err := os.MkdirAll(filepath.Dir(r.path), 0o700); err != nil {
		return &CorruptError{Path: r.path, Cause: fmt.Errorf("%v; could not create safety-copy directory: %w", cause, err)}
	}
	file, err := os.OpenFile(backup, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return &CorruptError{Path: r.path, Cause: fmt.Errorf("%v; could not create safety copy: %w", cause, err)}
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return &CorruptError{Path: r.path, Cause: fmt.Errorf("%v; could not write safety copy: %w", cause, err)}
	}
	if err := file.Close(); err != nil {
		return &CorruptError{Path: r.path, Cause: fmt.Errorf("%v; could not close safety copy: %w", cause, err)}
	}
	return &CorruptError{Path: r.path, BackupPath: backup, Cause: cause}
}
