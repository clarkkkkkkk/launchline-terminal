package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newStopCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "stop [workspace]",
		Short: "Request closure of applications started by Launchline",
		Long:  "Request closure of verified processes started by Launchline in the default or named workspace. This never force-kills applications or matches unrelated processes by name. Save your work first; application shutdown behavior varies.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			reference := ""
			if len(args) == 1 {
				reference = args[0]
			}
			summary, err := deps.Launch.Stop(command.Context(), reference)
			if err != nil {
				return fmt.Errorf("could not stop workspace: %w", err)
			}
			fmt.Fprintf(command.OutOrStdout(), "Stopping %s\n\n", summary.Workspace.Name)
			for _, result := range summary.Results {
				switch {
				case result.Err != nil:
					fmt.Fprintf(command.OutOrStdout(), "× %s — %v\n", result.Application.Name, result.Err)
				case result.Requested > 0:
					fmt.Fprintf(command.OutOrStdout(), "✓ %s — close requested (%d processes)\n", result.Application.Name, result.Requested)
				default:
					fmt.Fprintf(command.OutOrStdout(), "· %s — no tracked process running\n", result.Application.Name)
				}
			}
			fmt.Fprintf(command.OutOrStdout(), "\n%d process close requests sent.\n", summary.Requested())
			fmt.Fprintln(command.OutOrStdout(), "Requests do not confirm exit. Check applications for save prompts; untracked instances remain open.")
			if summary.Failed() > 0 {
				return fmt.Errorf("workspace stop had %d failed application(s)", summary.Failed())
			}
			return nil
		},
	}
}
