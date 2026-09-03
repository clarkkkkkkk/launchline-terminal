package command

type Action string

const (
	ActionHelp         Action = "help"
	ActionExit         Action = "exit"
	ActionApplications Action = "applications"
	ActionWorkspaces   Action = "workspaces"
	ActionWorkspace    Action = "workspace"
	ActionStart        Action = "start"
	ActionAdd          Action = "add"
	ActionRefresh      Action = "refresh"
	ActionSettings     Action = "settings"
	ActionVersion      Action = "version"
	ActionClear        Action = "clear"
)

type Definition struct {
	Name        string
	Aliases     []string
	Usage       string
	Description string
	Action      Action
	MinArgs     int
	MaxArgs     int
}

type Invocation struct {
	Definition Definition
	Arguments  []string
	Raw        string
}
