package discovery

import (
	"context"
	"errors"
	"time"
)

type RefreshResult struct {
	Catalog  Catalog
	Added    int
	Removed  int
	Warnings []error
}

type Service struct {
	repository CatalogRepository
	discoverer Discoverer
	now        func() time.Time
}

func NewService(repository CatalogRepository, discoverer Discoverer) *Service {
	return &Service{repository: repository, discoverer: discoverer, now: time.Now}
}

func (s *Service) Load() (Catalog, error) { return s.repository.Load() }
func (s *Service) Path() string           { return s.repository.Path() }

func (s *Service) Refresh(ctx context.Context) (RefreshResult, error) {
	previous, loadErr := s.repository.Load()
	if loadErr != nil {
		previous = EmptyCatalog()
	}
	items, discoverErr := s.discoverer.Discover(ctx)
	if ctx.Err() != nil {
		return RefreshResult{Catalog: previous}, ctx.Err()
	}
	items = Normalize(items)
	if discoverErr != nil && len(items) == 0 && len(previous.Applications) > 0 {
		return RefreshResult{Catalog: previous, Warnings: []error{discoverErr}}, discoverErr
	}
	if discoverErr != nil {
		// A partial refresh cannot prove that a missing cached target vanished.
		// Merge cached entries so a failing optional source never destroys data.
		items = Normalize(append(items, previous.Applications...))
	}
	before, after := map[string]bool{}, map[string]bool{}
	for _, item := range previous.Applications {
		before[item.ID] = true
	}
	for _, item := range items {
		after[item.ID] = true
	}
	result := RefreshResult{Catalog: Catalog{Version: CatalogVersion, RefreshedAt: s.now().UTC(), Applications: items}}
	for id := range after {
		if !before[id] {
			result.Added++
		}
	}
	for id := range before {
		if !after[id] {
			result.Removed++
		}
	}
	var partial *PartialError
	if errors.As(discoverErr, &partial) {
		result.Warnings = append(result.Warnings, partial.Errors...)
	} else if discoverErr != nil {
		result.Warnings = append(result.Warnings, discoverErr)
	}
	if err := s.repository.Save(result.Catalog); err != nil {
		return result, err
	}
	return result, discoverErr
}
