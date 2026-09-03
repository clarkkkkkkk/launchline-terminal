package app

// Workspace is an ordered collection of registered application IDs.
type Workspace struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Applications []string `json:"applications"`
}

const CurrentSchemaVersion = 2

// Config is the complete persisted Launchline configuration.
type Config struct {
	Version            int           `json:"version"`
	DefaultWorkspaceID string        `json:"default_workspace_id,omitempty"`
	CompactLogo        bool          `json:"compact_logo,omitempty"`
	Applications       []Application `json:"applications"`
	Workspaces         []Workspace   `json:"workspaces"`
}

func DefaultConfig() Config {
	return Config{Version: CurrentSchemaVersion, Applications: []Application{}, Workspaces: []Workspace{}}
}
