package app

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrDuplicateName = errors.New("name already exists")
	ErrNoDefault     = errors.New("no default workspace is configured")
)

// Repository persists the entire local configuration atomically.
type Repository interface {
	Load() (Config, error)
	Save(Config) error
	Path() string
}

// Service owns configuration validation and application/workspace operations.
// Both the CLI and TUI use this type so behavior stays consistent.
type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }
func (s *Service) Load() (Config, error)  { return s.repo.Load() }
func (s *Service) ConfigPath() string     { return s.repo.Path() }

func newID(prefix string) (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate identifier: %w", err)
	}
	return prefix + "_" + hex.EncodeToString(b), nil
}

func cleanName(name string) string { return strings.TrimSpace(name) }
func sameName(a, b string) bool    { return strings.EqualFold(cleanName(a), cleanName(b)) }

func validateApplication(input Application) error {
	if cleanName(input.Name) == "" {
		return errors.New("application name is required")
	}
	if strings.TrimSpace(input.Path) == "" {
		return errors.New("application path is required")
	}
	return nil
}

func (s *Service) AddApplication(input Application) (Application, error) {
	if err := validateApplication(input); err != nil {
		return Application{}, err
	}
	cfg, err := s.repo.Load()
	if err != nil {
		return Application{}, err
	}
	for _, existing := range cfg.Applications {
		if sameName(existing.Name, input.Name) {
			return Application{}, fmt.Errorf("application %q: %w", cleanName(input.Name), ErrDuplicateName)
		}
	}
	input.ID, err = newID("app")
	if err != nil {
		return Application{}, err
	}
	input.Name, input.Path = cleanName(input.Name), strings.TrimSpace(input.Path)
	input.Arguments = append([]string(nil), input.Arguments...)
	cfg.Applications = append(cfg.Applications, input)
	if err := s.repo.Save(cfg); err != nil {
		return Application{}, err
	}
	return input, nil
}

// LinkDiscoveredApplication adds or refreshes one catalog application in user
// configuration. It deliberately stores only applications the user selects,
// not the complete discovery catalog.
func (s *Service) LinkDiscoveredApplication(input Application) (Application, error) {
	if strings.TrimSpace(input.DiscoveryID) == "" {
		return Application{}, errors.New("discovery identifier is required")
	}
	if err := validateApplication(input); err != nil {
		return Application{}, err
	}
	cfg, err := s.repo.Load()
	if err != nil {
		return Application{}, err
	}
	for i, existing := range cfg.Applications {
		if existing.DiscoveryID == input.DiscoveryID {
			input.ID = existing.ID
			input.Name = availableApplicationName(cfg, cleanName(input.Name), existing.ID)
			input.Path = strings.TrimSpace(input.Path)
			input.Arguments = append([]string(nil), input.Arguments...)
			input.Unavailable = false
			cfg.Applications[i] = input
			if err := s.repo.Save(cfg); err != nil {
				return Application{}, err
			}
			return input, nil
		}
	}
	input.ID, err = newID("app")
	if err != nil {
		return Application{}, err
	}
	input.Name, input.Path = availableApplicationName(cfg, cleanName(input.Name), ""), strings.TrimSpace(input.Path)
	input.Arguments = append([]string(nil), input.Arguments...)
	cfg.Applications = append(cfg.Applications, input)
	if err := s.repo.Save(cfg); err != nil {
		return Application{}, err
	}
	return input, nil
}

func availableApplicationName(cfg Config, desired, excludeID string) string {
	available := func(candidate string) bool {
		for _, existing := range cfg.Applications {
			if existing.ID != excludeID && sameName(existing.Name, candidate) {
				return false
			}
		}
		return true
	}
	if available(desired) {
		return desired
	}
	base := desired + " (discovered)"
	if available(base) {
		return base
	}
	for number := 2; ; number++ {
		candidate := fmt.Sprintf("%s (discovered %d)", desired, number)
		if available(candidate) {
			return candidate
		}
	}
}

// ReconcileDiscoveredApplications marks linked catalog entries unavailable
// without deleting workspace references. Manual entries are never changed.
func (s *Service) ReconcileDiscoveredApplications(available map[string]bool) error {
	catalog := make(map[string]Application, len(available))
	for id := range available {
		catalog[id] = Application{DiscoveryID: id}
	}
	return s.ReconcileDiscoveredCatalog(catalog)
}

// ReconcileDiscoveredCatalog refreshes linked launch metadata and marks absent
// entries unavailable in one atomic configuration update.
func (s *Service) ReconcileDiscoveredCatalog(available map[string]Application) error {
	cfg, err := s.repo.Load()
	if err != nil {
		return err
	}
	changed := false
	for i := range cfg.Applications {
		if cfg.Applications[i].DiscoveryID == "" {
			continue
		}
		catalogItem, exists := available[cfg.Applications[i].DiscoveryID]
		unavailable := !exists
		if cfg.Applications[i].Unavailable != unavailable {
			cfg.Applications[i].Unavailable = unavailable
			changed = true
		}
		if exists && catalogItem.Path != "" {
			catalogItem.ID = cfg.Applications[i].ID
			catalogItem.DiscoveryID = cfg.Applications[i].DiscoveryID
			catalogItem.Name = availableApplicationName(cfg, cleanName(catalogItem.Name), catalogItem.ID)
			catalogItem.Path = strings.TrimSpace(catalogItem.Path)
			catalogItem.Arguments = append([]string(nil), catalogItem.Arguments...)
			catalogItem.Unavailable = false
			if !sameApplication(cfg.Applications[i], catalogItem) {
				cfg.Applications[i] = catalogItem
				changed = true
			}
		}
	}
	if !changed {
		return nil
	}
	return s.repo.Save(cfg)
}

func sameApplication(a, b Application) bool {
	if a.ID != b.ID || a.Name != b.Name || a.Path != b.Path || a.Kind != b.Kind || a.DiscoveryID != b.DiscoveryID || a.Source != b.Source || a.Unavailable != b.Unavailable || len(a.Arguments) != len(b.Arguments) {
		return false
	}
	for i := range a.Arguments {
		if a.Arguments[i] != b.Arguments[i] {
			return false
		}
	}
	return true
}

func (s *Service) UpdateApplication(id string, input Application) (Application, error) {
	if err := validateApplication(input); err != nil {
		return Application{}, err
	}
	cfg, err := s.repo.Load()
	if err != nil {
		return Application{}, err
	}
	index := -1
	for i, existing := range cfg.Applications {
		if existing.ID == id {
			index = i
		} else if sameName(existing.Name, input.Name) {
			return Application{}, fmt.Errorf("application %q: %w", cleanName(input.Name), ErrDuplicateName)
		}
	}
	if index < 0 {
		return Application{}, fmt.Errorf("application %q: %w", id, ErrNotFound)
	}
	input.ID = id
	input.Name, input.Path = cleanName(input.Name), strings.TrimSpace(input.Path)
	input.Arguments = append([]string(nil), input.Arguments...)
	cfg.Applications[index] = input
	if err := s.repo.Save(cfg); err != nil {
		return Application{}, err
	}
	return input, nil
}

func (s *Service) DeleteApplication(id string) error {
	cfg, err := s.repo.Load()
	if err != nil {
		return err
	}
	index := -1
	for i, item := range cfg.Applications {
		if item.ID == id {
			index = i
			break
		}
	}
	if index < 0 {
		return fmt.Errorf("application %q: %w", id, ErrNotFound)
	}
	cfg.Applications = append(cfg.Applications[:index], cfg.Applications[index+1:]...)
	for i := range cfg.Workspaces {
		kept := cfg.Workspaces[i].Applications[:0]
		for _, appID := range cfg.Workspaces[i].Applications {
			if appID != id {
				kept = append(kept, appID)
			}
		}
		cfg.Workspaces[i].Applications = kept
	}
	return s.repo.Save(cfg)
}

func validateWorkspace(input Workspace) error {
	if cleanName(input.Name) == "" {
		return errors.New("workspace name is required")
	}
	return nil
}

func validateApplicationIDs(cfg Config, ids []string) error {
	known := make(map[string]bool, len(cfg.Applications))
	for _, item := range cfg.Applications {
		known[item.ID] = true
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if !known[id] {
			return fmt.Errorf("workspace references unknown application %q", id)
		}
		if seen[id] {
			return fmt.Errorf("workspace contains duplicate application reference %q", id)
		}
		seen[id] = true
	}
	return nil
}

func (s *Service) CreateWorkspace(input Workspace) (Workspace, error) {
	if err := validateWorkspace(input); err != nil {
		return Workspace{}, err
	}
	cfg, err := s.repo.Load()
	if err != nil {
		return Workspace{}, err
	}
	for _, existing := range cfg.Workspaces {
		if sameName(existing.Name, input.Name) {
			return Workspace{}, fmt.Errorf("workspace %q: %w", cleanName(input.Name), ErrDuplicateName)
		}
	}
	if err := validateApplicationIDs(cfg, input.Applications); err != nil {
		return Workspace{}, err
	}
	input.ID, err = newID("ws")
	if err != nil {
		return Workspace{}, err
	}
	input.Name = cleanName(input.Name)
	input.Applications = append([]string(nil), input.Applications...)
	cfg.Workspaces = append(cfg.Workspaces, input)
	if cfg.DefaultWorkspaceID == "" {
		cfg.DefaultWorkspaceID = input.ID
	}
	if err := s.repo.Save(cfg); err != nil {
		return Workspace{}, err
	}
	return input, nil
}

func (s *Service) UpdateWorkspace(id string, input Workspace) (Workspace, error) {
	if err := validateWorkspace(input); err != nil {
		return Workspace{}, err
	}
	cfg, err := s.repo.Load()
	if err != nil {
		return Workspace{}, err
	}
	index := -1
	for i, existing := range cfg.Workspaces {
		if existing.ID == id {
			index = i
		} else if sameName(existing.Name, input.Name) {
			return Workspace{}, fmt.Errorf("workspace %q: %w", cleanName(input.Name), ErrDuplicateName)
		}
	}
	if index < 0 {
		return Workspace{}, fmt.Errorf("workspace %q: %w", id, ErrNotFound)
	}
	if err := validateApplicationIDs(cfg, input.Applications); err != nil {
		return Workspace{}, err
	}
	input.ID, input.Name = id, cleanName(input.Name)
	input.Applications = append([]string(nil), input.Applications...)
	cfg.Workspaces[index] = input
	if err := s.repo.Save(cfg); err != nil {
		return Workspace{}, err
	}
	return input, nil
}

func (s *Service) DeleteWorkspace(id string) error {
	cfg, err := s.repo.Load()
	if err != nil {
		return err
	}
	index := -1
	for i, item := range cfg.Workspaces {
		if item.ID == id {
			index = i
			break
		}
	}
	if index < 0 {
		return fmt.Errorf("workspace %q: %w", id, ErrNotFound)
	}
	cfg.Workspaces = append(cfg.Workspaces[:index], cfg.Workspaces[index+1:]...)
	if cfg.DefaultWorkspaceID == id {
		cfg.DefaultWorkspaceID = ""
		if len(cfg.Workspaces) > 0 {
			cfg.DefaultWorkspaceID = cfg.Workspaces[0].ID
		}
	}
	return s.repo.Save(cfg)
}

func (s *Service) SetDefaultWorkspace(id string) error {
	cfg, err := s.repo.Load()
	if err != nil {
		return err
	}
	for _, item := range cfg.Workspaces {
		if item.ID == id {
			cfg.DefaultWorkspaceID = id
			return s.repo.Save(cfg)
		}
	}
	return fmt.Errorf("workspace %q: %w", id, ErrNotFound)
}

func (s *Service) SetCompactLogo(compact bool) error {
	cfg, err := s.repo.Load()
	if err != nil {
		return err
	}
	cfg.CompactLogo = compact
	return s.repo.Save(cfg)
}

// ResolveWorkspace accepts an exact ID or case-insensitive human name. A blank
// reference resolves the single configured default workspace.
func (s *Service) ResolveWorkspace(reference string) (Workspace, Config, error) {
	cfg, err := s.repo.Load()
	if err != nil {
		return Workspace{}, Config{}, err
	}
	if strings.TrimSpace(reference) == "" {
		if cfg.DefaultWorkspaceID == "" {
			return Workspace{}, cfg, ErrNoDefault
		}
		reference = cfg.DefaultWorkspaceID
	}
	var matches []Workspace
	for _, item := range cfg.Workspaces {
		if item.ID == reference || sameName(item.Name, reference) {
			matches = append(matches, item)
		}
	}
	if len(matches) == 0 {
		return Workspace{}, cfg, fmt.Errorf("workspace %q: %w", reference, ErrNotFound)
	}
	if len(matches) > 1 {
		return Workspace{}, cfg, fmt.Errorf("workspace %q is ambiguous", reference)
	}
	return matches[0], cfg, nil
}

// ValidateConfig checks persisted relational and uniqueness invariants.
func ValidateConfig(cfg Config) error {
	if cfg.Version != CurrentSchemaVersion {
		return fmt.Errorf("unsupported configuration version %d (this build supports %d)", cfg.Version, CurrentSchemaVersion)
	}
	appIDs, appNames := map[string]bool{}, map[string]bool{}
	for _, item := range cfg.Applications {
		if item.ID == "" {
			return errors.New("application has an empty identifier")
		}
		if err := validateApplication(item); err != nil {
			return fmt.Errorf("application %q: %w", item.ID, err)
		}
		key := strings.ToLower(cleanName(item.Name))
		if appIDs[item.ID] || appNames[key] {
			return errors.New("configuration contains duplicate application identifiers or names")
		}
		appIDs[item.ID], appNames[key] = true, true
	}
	wsIDs, wsNames := map[string]bool{}, map[string]bool{}
	for _, item := range cfg.Workspaces {
		if item.ID == "" {
			return errors.New("workspace has an empty identifier")
		}
		if err := validateWorkspace(item); err != nil {
			return fmt.Errorf("workspace %q: %w", item.ID, err)
		}
		key := strings.ToLower(cleanName(item.Name))
		if wsIDs[item.ID] || wsNames[key] {
			return errors.New("configuration contains duplicate workspace identifiers or names")
		}
		wsIDs[item.ID], wsNames[key] = true, true
		seen := map[string]bool{}
		for _, id := range item.Applications {
			if !appIDs[id] {
				return fmt.Errorf("workspace %q references unknown application %q", item.Name, id)
			}
			if seen[id] {
				return fmt.Errorf("workspace %q contains duplicate application %q", item.Name, id)
			}
			seen[id] = true
		}
	}
	if cfg.DefaultWorkspaceID != "" && !wsIDs[cfg.DefaultWorkspaceID] {
		return fmt.Errorf("default workspace %q does not exist", cfg.DefaultWorkspaceID)
	}
	return nil
}
