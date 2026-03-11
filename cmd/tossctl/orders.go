package main

import (
	"github.com/junghoonkye/tossinvest-cli/internal/output"
	"github.com/spf13/cobra"
)

func newOrdersCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "orders",
		Short: "Read order history data",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List read-only order history",
			RunE: func(cmd *cobra.Command, _ []string) error {
				app, err := newAppContext(opts)
				if err != nil {
					return err
				}

				orders, err := app.client.ListPendingOrders(cmd.Context())
				if err != nil {
					return userFacingCommandError(err)
				}

				return output.WriteOrders(cmd.OutOrStdout(), app.format, orders)
			},
		},
	)

	return cmd
}
