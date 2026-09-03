package discovery

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

type CatalogRepository interface {
	Load() (Catalog, error)
	Save(Catalog) error
	Path() string
}

type CatalogCorruptError struct {
	Path       string
	BackupPath string
	Cause      error
}

func (e *CatalogCorruptError) Error() string {
	if e.BackupPath == "" {
		return fmt.Sprintf("application catalog at %s is invalid: %v; refresh to rebuild it", e.Path, e.Cause)
	}
	return fmt.Sprintf("application catalog at %s is invalid: %v (a safety copy is at %s; refresh to rebuild it)", e.Path, e.Cause, e.BackupPath)
}
func (e *CatalogCorruptError) Unwrap() error { return e.Cause }

type FileCatalogRepository struct{ path string }

func NewFileCatalogRepository(path string) *FileCatalogRepository {
	return &FileCatalogRepository{path: path}
}
func (r *FileCatalogRepository) Path() string { return r.path }

func (r *FileCatalogRepository) Load() (Catalog, error) {
	data, err := os.ReadFile(r.path)
	if errors.Is(err, os.ErrNotExist) {
		previous := r.path + ".previous"
		if _, previousErr := os.Stat(previous); previousErr == nil {
			if renameErr := os.Rename(previous, r.path); renameErr != nil {
				return Catalog{}, fmt.Errorf("recover interrupted catalog write: %w", renameErr)
			}
			return r.Load()
		}
		return EmptyCatalog(), nil
	}
	if err != nil {
		return Catalog{}, fmt.Errorf("read application catalog %s: %w", r.path, err)
	}
	var catalog Catalog
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&catalog); err != nil {
		return Catalog{}, r.corrupt(data, fmt.Errorf("decode JSON: %w", err))
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return Catalog{}, r.corrupt(data, errors.New("catalog contains trailing JSON"))
	}
	if catalog.Version != CatalogVersion {
		return Catalog{}, r.corrupt(data, fmt.Errorf("unsupported catalog version %d", catalog.Version))
	}
	catalog.Applications = Normalize(catalog.Applications)
	return catalog, nil
}

func (r *FileCatalogRepository) Save(catalog Catalog) error {
	catalog.Version = CatalogVersion
	catalog.Applications = Normalize(catalog.Applications)
	dir := filepath.Dir(r.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create catalog directory: %w", err)
	}
	temp, err := os.CreateTemp(dir, ".catalog-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary catalog: %w", err)
	}
	tempPath := temp.Name()
	committed := false
	defer func() {
		_ = temp.Close()
		if !committed {
			_ = os.Remove(tempPath)
		}
	}()
	if err := temp.Chmod(0o600); err != nil {
		return err
	}
	encoder := json.NewEncoder(temp)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(catalog); err != nil {
		return err
	}
	if err := temp.Sync(); err != nil {
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := replaceCatalogFile(tempPath, r.path); err != nil {
		return fmt.Errorf("replace application catalog: %w", err)
	}
	committed = true
	return nil
}

func (r *FileCatalogRepository) corrupt(data []byte, cause error) error {
	if err := os.MkdirAll(filepath.Dir(r.path), 0o700); err != nil {
		return &CatalogCorruptError{Path: r.path, Cause: cause}
	}
	backup := r.path + ".corrupt-" + time.Now().UTC().Format("20060102T150405.000000000Z")
	if err := os.WriteFile(backup, data, 0o600); err != nil {
		return &CatalogCorruptError{Path: r.path, Cause: cause}
	}
	return &CatalogCorruptError{Path: r.path, BackupPath: backup, Cause: cause}
}

func replaceCatalogFile(source, destination string) error {
	if runtime.GOOS != "windows" {
		return os.Rename(source, destination)
	}
	previous := destination + ".previous"
	_ = os.Remove(previous)
	if err := os.Rename(destination, previous); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(source, destination); err != nil {
		_ = os.Rename(previous, destination)
		return err
	}
	_ = os.Remove(previous)
	return nil
}
