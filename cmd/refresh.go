package cmd

import (
	"errors"
	"fmt"

	"github.com/launchline/launchline/internal/app"
	"github.com/launchline/launchline/internal/discovery"
	"github.com/spf13/cobra"
)

func newRefreshCommand(deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "refresh",
		Short: "Refresh the local installed-application catalog",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			if deps.Discovery == nil {
				return errors.New("application discovery is unavailable")
			}
			result, err := deps.Discovery.Refresh(command.Context())
			if reconcileErr := deps.Config.ReconcileDiscoveredCatalog(catalogApplications(result)); reconcileErr != nil {
				return reconcileErr
			}
			fmt.Fprintf(command.OutOrStdout(), "Application catalog refreshed: %d applications (%d new, %d no longer detected).\n", len(result.Catalog.Applications), result.Added, result.Removed)
			if err != nil {
				fmt.Fprintf(command.OutOrStdout(), "Refresh completed with warnings; cached applications were retained: %v\n", err)
			}
			return nil
		},
	}
}

func catalogApplications(result discovery.RefreshResult) map[string]app.Application {
	available := make(map[string]app.Application, len(result.Catalog.Applications))
	for _, item := range result.Catalog.Applications {
		available[item.ID] = app.Application{Name: item.Name, Path: item.Target, Arguments: item.Arguments, Kind: item.Kind, DiscoveryID: item.ID, Source: item.Source}
	}
	return available
}
