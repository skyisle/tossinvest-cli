package main

import (
	"fmt"
	"strings"
	"time"

	tossclient "github.com/junghoonkye/tossinvest-cli/internal/client"
	"github.com/junghoonkye/tossinvest-cli/internal/domain"
	"github.com/junghoonkye/tossinvest-cli/internal/output"
	"github.com/spf13/cobra"
)

func newProfitCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profit",
		Short: "Read realized P&L analysis (수익분석)",
		Long: "Read-only access to Toss Securities' 수익분석 surface. " +
			"Unlike `transactions list`, profit endpoints expose multi-year " +
			"realized P&L without the 365-day server cap.",
	}
	cmd.AddCommand(
		newProfitOverviewCmd(opts),
		newProfitDailyCmd(opts),
		newProfitGroupCmd(opts),
	)
	return cmd
}

type profitWindowFlags struct {
	rangeType string
	fromStr   string
	toStr     string
}

func (f *profitWindowFlags) bind(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.rangeType, "range", "all", "Range type: day|week|month|year|all (UI hint sent alongside dates)")
	cmd.Flags().StringVar(&f.fromStr, "from", "", "Start date YYYY-MM-DD (default: ~10 years ago)")
	cmd.Flags().StringVar(&f.toStr, "to", "", "End date YYYY-MM-DD (default: today, KST)")
}

func (f *profitWindowFlags) resolve() (time.Time, time.Time, error) {
	var from, to time.Time
	if v := strings.TrimSpace(f.toStr); v != "" {
		t, err := time.ParseInLocation("2006-01-02", v, tossclient.KoreaLocation)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid --to: %w", err)
		}
		to = t
	}
	if v := strings.TrimSpace(f.fromStr); v != "" {
		t, err := time.ParseInLocation("2006-01-02", v, tossclient.KoreaLocation)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid --from: %w", err)
		}
		from = t
	}
	return from, to, nil
}

func newProfitOverviewCmd(opts *rootOptions) *cobra.Command {
	winFlags := &profitWindowFlags{}
	cmd := &cobra.Command{
		Use:   "overview",
		Short: "Realized P&L aggregate (sales/dividend/lending/maturity/interest)",
		Long: "Calls /api/v1/profit/overview. Returns lifetime or period totals " +
			"broken down by source. With no flags, this is the multi-year " +
			"aggregate 수익분석 shows in the '전체' period.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}
			from, to, err := winFlags.resolve()
			if err != nil {
				return err
			}
			overview, err := app.client.GetProfitOverview(cmd.Context(), winFlags.rangeType, from, to)
			if err != nil {
				return userFacingCommandError(err)
			}
			return output.WriteProfitOverview(cmd.OutOrStdout(), app.format, overview)
		},
	}
	winFlags.bind(cmd)
	return cmd
}

type profitDailyFlags struct {
	currency  string
	size      int
	page      int
	all       bool
	pageLimit int
}

func newProfitDailyCmd(opts *rootOptions) *cobra.Command {
	winFlags := &profitWindowFlags{}
	flags := &profitDailyFlags{}
	cmd := &cobra.Command{
		Use:   "daily",
		Short: "Per-symbol daily realized P&L (multi-year, all markets)",
		Long: "Calls /api/v1/profit/wts/daily/market. Returns per-symbol per-date " +
			"realized P&L across both KR and US markets. This is the multi-year " +
			"time series that backs 수익분석's '일별' view and does not share the " +
			"365-day cap of `transactions list`.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}
			from, to, err := winFlags.resolve()
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			var items []domain.ProfitDailyItem
			if flags.all {
				items, err = app.client.ListAllProfitDaily(ctx, flags.currency, from, to, flags.size, flags.pageLimit)
				if err != nil {
					return userFacingCommandError(err)
				}
			} else {
				page, err := app.client.ListProfitDaily(ctx, flags.currency, from, to, flags.page, flags.size)
				if err != nil {
					return userFacingCommandError(err)
				}
				items = page.Items
				if page.HasNext && app.format == output.FormatTable {
					fmt.Fprintf(cmd.ErrOrStderr(),
						"(more pages available; pass --all or --page=%d to fetch next)\n",
						page.CurrentPage+1,
					)
				}
			}
			return output.WriteProfitDaily(cmd.OutOrStdout(), app.format, items)
		},
	}
	winFlags.bind(cmd)
	cmd.Flags().StringVar(&flags.currency, "currency", "USD", "Display currency hint: KRW or USD (response carries both)")
	cmd.Flags().IntVar(&flags.size, "size", tossclient.ProfitDefaultDailyPageSize, "Page size")
	cmd.Flags().IntVar(&flags.page, "page", 0, "Page index (0-based); ignored when --all")
	cmd.Flags().BoolVar(&flags.all, "all", false, "Page through until hasNext is false")
	cmd.Flags().IntVar(&flags.pageLimit, "page-limit", tossclient.ProfitDefaultPageLimit, "Max pages when --all is set")
	return cmd
}

type profitGroupFlags struct {
	market     string
	profitType string
	currency   string
	size       int
	page       int
	all        bool
	pageLimit  int
}

func newProfitGroupCmd(opts *rootOptions) *cobra.Command {
	winFlags := &profitWindowFlags{}
	flags := &profitGroupFlags{}
	cmd := &cobra.Command{
		Use:   "group",
		Short: "Per-symbol aggregate (종목별 합계) for a single market",
		Long: "Calls /api/v2/profit/group/market. Returns one symbol per row " +
			"with totals across the window. The endpoint scopes to one market " +
			"and one source type per request — the UI splits KR/US tabs the same way.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}
			from, to, err := winFlags.resolve()
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			var page domain.ProfitGroupPage
			if flags.all {
				page, err = app.client.ListAllProfitGroup(ctx, flags.market, flags.profitType, winFlags.rangeType, flags.currency, from, to, flags.size, flags.pageLimit)
				if err != nil {
					return userFacingCommandError(err)
				}
			} else {
				page, err = app.client.ListProfitGroup(ctx, flags.market, flags.profitType, winFlags.rangeType, flags.currency, from, to, flags.page, flags.size)
				if err != nil {
					return userFacingCommandError(err)
				}
				if !page.IsLast && app.format == output.FormatTable {
					next := flags.page + 1
					if page.NextPage != nil {
						next = page.NextPage.Page
					}
					fmt.Fprintf(cmd.ErrOrStderr(),
						"(more pages available; pass --all or --page=%d to fetch next)\n",
						next,
					)
				}
			}
			return output.WriteProfitGroup(cmd.OutOrStdout(), app.format, page)
		},
	}
	winFlags.bind(cmd)
	cmd.Flags().StringVar(&flags.market, "market", "kr", "Market: kr or us")
	cmd.Flags().StringVar(&flags.profitType, "type", "sales", "Profit type: sales|dividend|account-interest")
	cmd.Flags().StringVar(&flags.currency, "currency", "USD", "Currency hint: KRW or USD (response carries both)")
	cmd.Flags().IntVar(&flags.size, "size", tossclient.ProfitDefaultGroupPageSize, "Page size")
	cmd.Flags().IntVar(&flags.page, "page", 0, "Page index (0-based); ignored when --all")
	cmd.Flags().BoolVar(&flags.all, "all", false, "Page through until isLast is true")
	cmd.Flags().IntVar(&flags.pageLimit, "page-limit", tossclient.ProfitDefaultPageLimit, "Max pages when --all is set")
	return cmd
}
