package main

import (
	"github.com/junghoonkye/tossinvest-cli/internal/output"
	"github.com/spf13/cobra"
)

func newPortfolioCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "portfolio",
		Short: "Read portfolio and holdings data",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "positions",
			Short: "List current positions",
			RunE: func(cmd *cobra.Command, _ []string) error {
				app, err := newAppContext(opts)
				if err != nil {
					return err
				}

				positions, err := app.client.ListPositions(cmd.Context())
				if err != nil {
					return userFacingCommandError(err)
				}

				return output.WritePositions(cmd.OutOrStdout(), app.format, positions)
			},
		},
		&cobra.Command{
			Use:   "allocation",
			Short: "Show portfolio allocation",
			RunE: func(cmd *cobra.Command, _ []string) error {
				app, err := newAppContext(opts)
				if err != nil {
					return err
				}

				summary, err := app.client.GetAccountSummary(cmd.Context())
				if err != nil {
					return userFacingCommandError(err)
				}

				return output.WriteAllocation(cmd.OutOrStdout(), app.format, summary.Markets)
			},
		},
	)

	return cmd
}
