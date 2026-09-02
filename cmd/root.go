package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/launchline/launchline/internal/app"
	"github.com/launchline/launchline/internal/config"
	platformlauncher "github.com/launchline/launchline/internal/launcher"
	"github.com/launchline/launchline/internal/tui"
	"github.com/spf13/cobra"
)

type Dependencies struct {
	Config *app.Service
	Launch *app.LaunchService
	RunTUI func(*app.Service, *app.LaunchService) error
}

func defaultDependencies() (Dependencies, error) {
	repository, err := config.NewDefaultRepository()
	if err != nil {
		return Dependencies{}, err
	}
	service := app.NewService(repository)
	launch := app.NewLaunchService(service, platformlauncher.New())
	return Dependencies{Config: service, Launch: launch, RunTUI: tui.Run}, nil
}

func NewRootCommand(deps Dependencies, stdout, stderr io.Writer) *cobra.Command {
	root := &cobra.Command{
		Use:           "launchline",
		Short:         "One command. Your entire workspace.",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return deps.RunTUI(deps.Config, deps.Launch)
		},
	}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetFlagErrorFunc(func(command *cobra.Command, err error) error {
		return fmt.Errorf("%w\nRun %q for usage", err, command.CommandPath()+" --help")
	})
	root.AddCommand(newStartCommand(deps), newAddCommand(deps), newAppsCommand(deps), newWorkspaceCommand(deps), newConfigCommand(deps), newVersionCommand())
	return root
}

func Execute() error {
	deps, err := defaultDependencies()
	if err != nil {
		return err
	}
	return NewRootCommand(deps, os.Stdout, os.Stderr).Execute()
}

func findApplication(cfg app.Config, reference string) (app.Application, error) {
	for _, item := range cfg.Applications {
		if item.ID == reference || equalName(item.Name, reference) {
			return item, nil
		}
	}
	return app.Application{}, fmt.Errorf("application %q was not found", reference)
}

func findWorkspace(cfg app.Config, reference string) (app.Workspace, error) {
	for _, item := range cfg.Workspaces {
		if item.ID == reference || equalName(item.Name, reference) {
			return item, nil
		}
	}
	return app.Workspace{}, fmt.Errorf("workspace %q was not found", reference)
}

func equalName(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}
