package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newConfigCommand(deps Dependencies) *cobra.Command {
	command := &cobra.Command{
		Use:   "config",
		Short: "Show local configuration information",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			cfg, err := deps.Config.Load()
			if err != nil {
				return err
			}
			fmt.Fprintf(command.OutOrStdout(), "Path: %s\nSchema: %d\nApplications: %d\nWorkspaces: %d\n", deps.Config.ConfigPath(), cfg.Version, len(cfg.Applications), len(cfg.Workspaces))
			return nil
		},
	}
	command.AddCommand(&cobra.Command{
		Use:   "path",
		Short: "Print the configuration file path",
		Args:  cobra.NoArgs,
		Run: func(command *cobra.Command, _ []string) {
			fmt.Fprintln(command.OutOrStdout(), deps.Config.ConfigPath())
		},
	})
	return command
}
