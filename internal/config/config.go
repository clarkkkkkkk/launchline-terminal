package config

import "fmt"

// CorruptError reports invalid configuration without replacing the source file.
type CorruptError struct {
	Path       string
	BackupPath string
	Cause      error
}

func (e *CorruptError) Error() string {
	if e.BackupPath != "" {
		return fmt.Sprintf("configuration at %s is invalid: %v (a safety copy is at %s)", e.Path, e.Cause, e.BackupPath)
	}
	return fmt.Sprintf("configuration at %s is invalid: %v", e.Path, e.Cause)
}

func (e *CorruptError) Unwrap() error { return e.Cause }
