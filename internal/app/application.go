package app

// Application is a locally registered program Launchline can start.
type Application struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Path      string   `json:"path"`
	Arguments []string `json:"arguments,omitempty"`
}
