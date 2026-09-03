package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/launchline/launchline/internal/app"
	"github.com/spf13/cobra"
)

func newAppsCommand(deps Dependencies) *cobra.Command {
	command := &cobra.Command{
		Use:     "apps",
		Aliases: []string{"applications"},
		Short:   "List and manage registered applications",
		Args:    cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			cfg, err := deps.Config.Load()
			if err != nil {
				return err
			}
			catalogCount := 0
			if deps.Discovery != nil {
				catalog, catalogErr := deps.Discovery.Load()
				if catalogErr == nil {
					catalogCount = len(catalog.Applications)
					for _, item := range catalog.Applications {
						fmt.Fprintf(command.OutOrStdout(), "%s\n  %s  [discovered: %s]\n", item.Name, item.Target, item.Source)
					}
				} else {
					fmt.Fprintf(command.OutOrStdout(), "Application catalog warning: %v\n\n", catalogErr)
				}
			}
			if len(cfg.Applications) == 0 && catalogCount == 0 {
				fmt.Fprintln(command.OutOrStdout(), "No applications found.\n\nRun `launchline refresh`, or add a custom app with `launchline add --name NAME --path PATH`.")
				return nil
			}
			for _, item := range cfg.Applications {
				if item.DiscoveryID != "" {
					continue
				}
				args := app.FormatArguments(item.Arguments)
				if args != "" {
					args = " " + args
				}
				fmt.Fprintf(command.OutOrStdout(), "%s\n  %s%s  [manual]\n", item.Name, item.Path, args)
			}
			return nil
		},
	}
	command.AddCommand(newAppsEditCommand(deps), newAppsDeleteCommand(deps))
	return command
}

func newAppsEditCommand(deps Dependencies) *cobra.Command {
	var name, path string
	var arguments []string
	command := &cobra.Command{
		Use:   "edit <application>",
		Short: "Edit a registered application without changing its identity",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			cfg, err := deps.Config.Load()
			if err != nil {
				return err
			}
			item, err := findApplication(cfg, args[0])
			if err != nil {
				return err
			}
			changed := false
			if command.Flags().Changed("name") {
				item.Name, changed = name, true
			}
			if command.Flags().Changed("path") {
				item.Path, changed = path, true
			}
			if command.Flags().Changed("arg") {
				item.Arguments, changed = arguments, true
			}
			if !changed {
				return errors.New("nothing to change; provide --name, --path, or --arg")
			}
			updated, err := deps.Config.UpdateApplication(item.ID, item)
			if err != nil {
				return err
			}
			fmt.Fprintf(command.OutOrStdout(), "Updated %s.\n", updated.Name)
			return nil
		},
	}
	command.Flags().StringVarP(&name, "name", "n", "", "new display name")
	command.Flags().StringVarP(&path, "path", "p", "", "new executable/application path")
	command.Flags().StringArrayVarP(&arguments, "arg", "a", nil, "replacement launch argument; repeat for multiple")
	return command
}

func newAppsDeleteCommand(deps Dependencies) *cobra.Command {
	var yes bool
	command := &cobra.Command{
		Use:   "delete <application>",
		Short: "Remove an application from Launchline configuration",
		Long:  "Remove an application entry and its workspace references. This never uninstalls or deletes the actual program.",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			if !yes {
				return errors.New("deletion requires --yes; this removes only Launchline configuration and never uninstalls the program")
			}
			cfg, err := deps.Config.Load()
			if err != nil {
				return err
			}
			item, err := findApplication(cfg, strings.TrimSpace(args[0]))
			if err != nil {
				return err
			}
			if err := deps.Config.DeleteApplication(item.ID); err != nil {
				return err
			}
			fmt.Fprintf(command.OutOrStdout(), "Removed %s from Launchline. The installed application was not changed.\n", item.Name)
			return nil
		},
	}
	command.Flags().BoolVar(&yes, "yes", false, "confirm removal from Launchline configuration")
	return command
}
