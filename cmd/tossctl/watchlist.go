package main

import (
	"github.com/junghoonkye/tossinvest-cli/internal/output"
	"github.com/spf13/cobra"
)

func newWatchlistCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watchlist",
		Short: "Read watchlist data",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List watchlist entries",
			RunE: func(cmd *cobra.Command, _ []string) error {
				app, err := newAppContext(opts)
				if err != nil {
					return err
				}

				items, err := app.client.ListWatchlist(cmd.Context())
				if err != nil {
					return userFacingCommandError(err)
				}

				return output.WriteWatchlist(cmd.OutOrStdout(), app.format, items)
			},
		},
	)

	return cmd
}
