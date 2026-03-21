package main

import (
	"github.com/junghoonkye/tossinvest-cli/internal/domain"
	"github.com/junghoonkye/tossinvest-cli/internal/output"
	"github.com/spf13/cobra"
)

func newQuoteCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quote",
		Short: "Read quote data",
	}

	getCmd := &cobra.Command{
		Use:   "get <symbol>",
		Short: "Fetch quote data for a symbol",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}

			quote, err := app.client.GetQuote(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			return output.WriteQuote(cmd.OutOrStdout(), app.format, quote)
		},
	}

	batchCmd := &cobra.Command{
		Use:   "batch <symbol> [symbol...]",
		Short: "Fetch quotes for multiple symbols at once",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}

			var quotes []domain.Quote
			for _, symbol := range args {
				quote, err := app.client.GetQuote(cmd.Context(), symbol)
				if err != nil {
					return err
				}
				quotes = append(quotes, quote)
			}

			return output.WriteQuotes(cmd.OutOrStdout(), app.format, quotes)
		},
	}

	cmd.AddCommand(getCmd, batchCmd)

	return cmd
}
