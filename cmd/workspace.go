package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/launchline/launchline/internal/app"
	"github.com/spf13/cobra"
)

func newWorkspaceCommand(deps Dependencies) *cobra.Command {
	command := &cobra.Command{
		Use:     "workspace",
		Aliases: []string{"workspaces"},
		Short:   "List and manage workspaces",
		Args:    cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			cfg, err := deps.Config.Load()
			if err != nil {
				return err
			}
			if len(cfg.Workspaces) == 0 {
				fmt.Fprintln(command.OutOrStdout(), "No workspaces yet.\n\nCreate one with `launchline workspace create --name NAME`.")
				return nil
			}
			for _, item := range cfg.Workspaces {
				marker := " "
				if item.ID == cfg.DefaultWorkspaceID {
					marker = "*"
				}
				fmt.Fprintf(command.OutOrStdout(), "%s %s (%d applications)\n", marker, item.Name, len(item.Applications))
			}
			return nil
		},
	}
	command.AddCommand(newWorkspaceCreateCommand(deps), newWorkspaceEditCommand(deps), newWorkspaceDeleteCommand(deps), newWorkspaceDefaultCommand(deps))
	return command
}

func resolveAppReferences(deps Dependencies, cfg app.Config, references []string) ([]string, error) {
	ids := make([]string, 0, len(references))
	seen := map[string]bool{}
	for _, reference := range references {
		item, err := findApplication(cfg, strings.TrimSpace(reference))
		if err != nil && deps.Discovery != nil {
			catalog, catalogErr := deps.Discovery.Load()
			if catalogErr != nil {
				return nil, catalogErr
			}
			for _, discovered := range catalog.Applications {
				if discovered.ID == strings.TrimSpace(reference) || equalName(discovered.Name, reference) {
					item, err = deps.Config.LinkDiscoveredApplication(app.Application{Name: discovered.Name, Path: discovered.Target, Arguments: discovered.Arguments, Kind: discovered.Kind, DiscoveryID: discovered.ID, Source: discovered.Source})
					if err == nil {
						cfg.Applications = append(cfg.Applications, item)
					}
					break
				}
			}
		}
		if err != nil {
			return nil, err
		}
		if seen[item.ID] {
			return nil, fmt.Errorf("application %q was selected more than once", item.Name)
		}
		seen[item.ID] = true
		ids = append(ids, item.ID)
	}
	return ids, nil
}

func newWorkspaceCreateCommand(deps Dependencies) *cobra.Command {
	var name string
	var applications []string
	var makeDefault bool
	command := &cobra.Command{
		Use:     "create",
		Short:   "Create a workspace",
		Example: "  launchline workspace create --name Development --app Cursor --app Terminal --default",
		Args:    cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			if strings.TrimSpace(name) == "" {
				return errors.New("--name is required")
			}
			cfg, err := deps.Config.Load()
			if err != nil {
				return err
			}
			ids, err := resolveAppReferences(deps, cfg, applications)
			if err != nil {
				return err
			}
			created, err := deps.Config.CreateWorkspace(app.Workspace{Name: name, Applications: ids})
			if err != nil {
				return err
			}
			if makeDefault {
				if err := deps.Config.SetDefaultWorkspace(created.ID); err != nil {
					return err
				}
			}
			fmt.Fprintf(command.OutOrStdout(), "Created workspace %s with %d applications.\n", created.Name, len(created.Applications))
			return nil
		},
	}
	command.Flags().StringVarP(&name, "name", "n", "", "workspace name (required)")
	command.Flags().StringArrayVarP(&applications, "app", "a", nil, "application name or ID; repeat to select multiple")
	command.Flags().BoolVar(&makeDefault, "default", false, "make this the default workspace")
	return command
}

func newWorkspaceEditCommand(deps Dependencies) *cobra.Command {
	var name string
	var applications []string
	command := &cobra.Command{
		Use:   "edit <workspace>",
		Short: "Rename a workspace or replace its application selection",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			cfg, err := deps.Config.Load()
			if err != nil {
				return err
			}
			workspace, err := findWorkspace(cfg, args[0])
			if err != nil {
				return err
			}
			changed := false
			if command.Flags().Changed("name") {
				workspace.Name, changed = name, true
			}
			if command.Flags().Changed("app") {
				workspace.Applications, err = resolveAppReferences(deps, cfg, applications)
				if err != nil {
					return err
				}
				changed = true
			}
			if !changed {
				return errors.New("nothing to change; provide --name or --app")
			}
			updated, err := deps.Config.UpdateWorkspace(workspace.ID, workspace)
			if err != nil {
				return err
			}
			fmt.Fprintf(command.OutOrStdout(), "Updated workspace %s.\n", updated.Name)
			return nil
		},
	}
	command.Flags().StringVarP(&name, "name", "n", "", "new workspace name")
	command.Flags().StringArrayVarP(&applications, "app", "a", nil, "replacement application name or ID; repeat to select multiple")
	return command
}

func newWorkspaceDeleteCommand(deps Dependencies) *cobra.Command {
	var yes bool
	command := &cobra.Command{
		Use:   "delete <workspace>",
		Short: "Delete a workspace from Launchline",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			if !yes {
				return errors.New("deletion requires --yes; this removes only the Launchline workspace configuration")
			}
			cfg, err := deps.Config.Load()
			if err != nil {
				return err
			}
			workspace, err := findWorkspace(cfg, args[0])
			if err != nil {
				return err
			}
			if err := deps.Config.DeleteWorkspace(workspace.ID); err != nil {
				return err
			}
			fmt.Fprintf(command.OutOrStdout(), "Deleted workspace %s. Registered applications were not changed.\n", workspace.Name)
			return nil
		},
	}
	command.Flags().BoolVar(&yes, "yes", false, "confirm workspace deletion")
	return command
}

func newWorkspaceDefaultCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "default <workspace>",
		Short: "Choose the default workspace used by `launchline start`",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			cfg, err := deps.Config.Load()
			if err != nil {
				return err
			}
			workspace, err := findWorkspace(cfg, args[0])
			if err != nil {
				return err
			}
			if err := deps.Config.SetDefaultWorkspace(workspace.ID); err != nil {
				return err
			}
			fmt.Fprintf(command.OutOrStdout(), "%s is now the default workspace.\n", workspace.Name)
			return nil
		},
	}
}
