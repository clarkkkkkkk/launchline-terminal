package discovery

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sort"
	"strings"
	"time"
)

const CatalogVersion = 1

const (
	KindExecutable = "executable"
	KindDesktop    = "desktop-entry"
	KindBundle     = "app-bundle"
	KindShortcut   = "shortcut"
)

// Application is a normalized, locally discovered launch target.
type Application struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Target    string            `json:"target"`
	Arguments []string          `json:"arguments,omitempty"`
	Kind      string            `json:"kind"`
	Source    string            `json:"source"`
	Platform  string            `json:"platform"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type Catalog struct {
	Version      int           `json:"version"`
	RefreshedAt  time.Time     `json:"refreshed_at"`
	Applications []Application `json:"applications"`
}

func EmptyCatalog() Catalog {
	return Catalog{Version: CatalogVersion, Applications: []Application{}}
}

type Discoverer interface {
	Discover(context.Context) ([]Application, error)
}

type DiscovererFunc func(context.Context) ([]Application, error)

func (f DiscovererFunc) Discover(ctx context.Context) ([]Application, error) { return f(ctx) }

// PartialError means at least one source failed while other useful sources
// continued. Callers may keep cached entries when this error is returned.
type PartialError struct{ Errors []error }

func (e *PartialError) Error() string {
	parts := make([]string, 0, len(e.Errors))
	for _, err := range e.Errors {
		parts = append(parts, err.Error())
	}
	return "application discovery completed with warnings: " + strings.Join(parts, "; ")
}

func (e *PartialError) Unwrap() error { return errors.Join(e.Errors...) }

type namedSource struct {
	name string
	find Discoverer
}

type multiDiscoverer struct{ sources []namedSource }

func (d multiDiscoverer) Discover(ctx context.Context) ([]Application, error) {
	var applications []Application
	var failures []error
	for _, source := range d.sources {
		if err := ctx.Err(); err != nil {
			return applications, err
		}
		items, err := source.find.Discover(ctx)
		applications = append(applications, items...)
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", source.name, err))
		}
	}
	applications = Normalize(applications)
	if len(failures) > 0 {
		return applications, &PartialError{Errors: failures}
	}
	return applications, nil
}

func NewPlatformDiscoverer() Discoverer { return newPlatformDiscoverer() }

func sortApplications(items []Application) {
	sort.SliceStable(items, func(i, j int) bool {
		left, right := strings.ToLower(items[i].Name), strings.ToLower(items[j].Name)
		if left != right {
			return left < right
		}
		return items[i].ID < items[j].ID
	})
}

func platformName() string { return runtime.GOOS }
