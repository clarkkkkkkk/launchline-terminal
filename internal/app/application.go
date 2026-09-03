package app

// Application is a manual or user-selected discovered program Launchline can start.
type Application struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Path        string   `json:"path"`
	Arguments   []string `json:"arguments,omitempty"`
	Kind        string   `json:"kind,omitempty"`
	DiscoveryID string   `json:"discovery_id,omitempty"`
	Source      string   `json:"source,omitempty"`
	Unavailable bool     `json:"unavailable,omitempty"`
}

// Manual reports whether this entry was created explicitly by the user. A
// discovered application is copied into config only after it is selected for
// a workspace; the full machine catalog remains separate from config.json.
func (a Application) Manual() bool { return a.DiscoveryID == "" }
