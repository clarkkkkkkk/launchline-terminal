package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/launchline/launchline/internal/app"
	"github.com/spf13/cobra"
)

func newAddCommand(deps Dependencies) *cobra.Command {
	var name, path string
	var arguments []string
	command := &cobra.Command{
		Use:   "add",
		Short: "Register an application",
		Example: "  launchline add --name Cursor --path /usr/bin/cursor\n" +
			"  launchline add --name Chrome --path /path/to/chrome --arg=--profile-directory --arg=Work",
		Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			if strings.TrimSpace(name) == "" || strings.TrimSpace(path) == "" {
				return errors.New("both --name and --path are required; run `launchline add --help` for an example")
			}
			created, err := deps.Config.AddApplication(app.Application{Name: name, Path: path, Arguments: arguments})
			if err != nil {
				return err
			}
			fmt.Fprintf(command.OutOrStdout(), "Added %s.\n", created.Name)
			return nil
		},
	}
	command.Flags().StringVarP(&name, "name", "n", "", "display name (required)")
	command.Flags().StringVarP(&path, "path", "p", "", "executable, application, or supported target path (required)")
	command.Flags().StringArrayVarP(&arguments, "arg", "a", nil, "launch argument; repeat for multiple arguments")
	return command
}
