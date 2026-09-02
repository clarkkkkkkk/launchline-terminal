package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newStartCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "start [workspace]",
		Short: "Launch every application in a workspace",
		Long:  "Launch a named workspace, or the default workspace when no name is supplied.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			reference := ""
			if len(args) == 1 {
				reference = args[0]
			}
			summary, err := deps.Launch.Start(command.Context(), reference)
			if err != nil {
				return fmt.Errorf("could not start workspace: %w. Create or choose one with `launchline workspace`", err)
			}
			fmt.Fprintf(command.OutOrStdout(), "Starting %s\n\n", summary.Workspace.Name)
			for _, result := range summary.Results {
				if result.Err != nil {
					fmt.Fprintf(command.OutOrStdout(), "× %s — %v\n", result.Application.Name, result.Err)
				} else {
					fmt.Fprintf(command.OutOrStdout(), "✓ %s\n", result.Application.Name)
				}
			}
			fmt.Fprintf(command.OutOrStdout(), "\n%d applications launched.\n", summary.Succeeded())
			if summary.Failed() > 0 {
				fmt.Fprintf(command.OutOrStdout(), "%d applications failed.\n", summary.Failed())
				return fmt.Errorf("workspace started with %d failed application(s)", summary.Failed())
			}
			return nil
		},
	}
}
