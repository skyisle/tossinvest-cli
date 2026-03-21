package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
	"github.com/junghoonkye/tossinvest-cli/internal/orderintent"
	"github.com/junghoonkye/tossinvest-cli/internal/orderlineage"
	"github.com/junghoonkye/tossinvest-cli/internal/output"
	"github.com/junghoonkye/tossinvest-cli/internal/trading"
	"github.com/spf13/cobra"
)

type placeFlags struct {
	symbol       string
	market       string
	side         string
	orderType    string
	quantity     float64
	price        float64
	amount       float64
	currencyMode string
	fractional   bool
}

type executeFlags struct {
	execute                    bool
	dangerouslySkipPermissions bool
	confirm                    string
}

type amendFlags struct {
	orderID  string
	quantity float64
	price    float64
}

func newOrderCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "order",
		Short: "Preview, inspect, and manage trading actions",
		Long: "Trading commands are intentionally separate from read-only commands and " +
			"default to a local preview and permission gate before any future live mutation support.",
	}

	cmd.AddCommand(
		newOrderShowCmd(opts),
		newOrderPreviewCmd(opts),
		newOrderPlaceCmd(opts),
		newOrderCancelCmd(opts),
		newOrderAmendCmd(opts),
		newOrderPermissionsCmd(opts),
	)

	return cmd
}

func newOrderShowCmd(opts *rootOptions) *cobra.Command {
	var market string

	cmd := &cobra.Command{
		Use:   "show <order-id>",
		Short: "Show a single order from pending or current-month completed history",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}

			aliases := []string{}
			lineageHintKey := args[0]
			lineageErr := error(nil)
			if app.lineageService != nil {
				if currentOrderID, ok, err := app.lineageService.Resolve(args[0]); err != nil {
					lineageErr = err
				} else if ok {
					aliases = append(aliases, currentOrderID)
					lineageHintKey = currentOrderID
				}
			}

			order, err := app.client.FindOrderWithAliases(cmd.Context(), args[0], market, aliases...)
			if err != nil {
				if recoveredOrder, recovered, recoveryErr := recoverOrderWithLineageHint(cmd.Context(), app, args[0], lineageHintKey, market); recoveryErr != nil {
					if lineageErr != nil {
						return fmt.Errorf("%v; local lineage cache %s could not be read: %v", recoveryErr, app.paths.LineageFile, lineageErr)
					}
					return recoveryErr
				} else if recovered {
					return output.WriteOrder(cmd.OutOrStdout(), app.format, recoveredOrder)
				}
				if lineageErr != nil {
					return fmt.Errorf("%w; local lineage cache %s could not be read: %v", err, app.paths.LineageFile, lineageErr)
				}
				return userFacingCommandError(err)
			}

			return output.WriteOrder(cmd.OutOrStdout(), app.format, order)
		},
	}

	cmd.Flags().StringVar(&market, "market", "all", "Completed-history market filter used during lookup: all, us, kr")
	return cmd
}

func newOrderPreviewCmd(opts *rootOptions) *cobra.Command {
	flags := defaultPlaceFlags()

	cmd := &cobra.Command{
		Use:   "preview",
		Short: "Preview a canonical order intent",
		RunE: func(cmd *cobra.Command, _ []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}

			intent, err := orderintent.NormalizePlace(orderintent.PlaceInput{
				Symbol:       flags.symbol,
				Market:       flags.market,
				Side:         flags.side,
				OrderType:    flags.orderType,
				Quantity:     flags.quantity,
				Price:        flags.price,
				Amount:       flags.amount,
				CurrencyMode: flags.currencyMode,
				Fractional:   flags.fractional,
			})
			if err != nil {
				return err
			}

			return output.WriteTradingPreview(cmd.OutOrStdout(), app.format, app.tradingService.PreviewPlace(intent))
		},
	}

	bindPlaceFlags(cmd, flags)
	return cmd
}

func newOrderPlaceCmd(opts *rootOptions) *cobra.Command {
	place := defaultPlaceFlags()
	exec := &executeFlags{}

	cmd := &cobra.Command{
		Use:   "place",
		Short: "Place a live order with explicit danger approval",
		RunE: func(cmd *cobra.Command, _ []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}

			intent, err := orderintent.NormalizePlace(orderintent.PlaceInput{
				Symbol:       place.symbol,
				Market:       place.market,
				Side:         place.side,
				OrderType:    place.orderType,
				Quantity:     place.quantity,
				Price:        place.price,
				Amount:       place.amount,
				CurrencyMode: place.currencyMode,
				Fractional:   place.fractional,
			})
			if err != nil {
				return err
			}

			result, err := app.tradingService.Place(cmd.Context(), intent, trading.ExecuteOptions{
				Execute:                    exec.execute,
				DangerouslySkipPermissions: exec.dangerouslySkipPermissions,
				Confirm:                    exec.confirm,
			})
			if err != nil {
				return userFacingPlaceError(app.paths, err, place)
			}
			recordMutationLineage(app, &result)

			return output.WriteMutationResult(cmd.OutOrStdout(), app.format, result)
		},
	}

	bindPlaceFlags(cmd, place)
	bindExecuteFlags(cmd, exec)
	return cmd
}

func newOrderCancelCmd(opts *rootOptions) *cobra.Command {
	exec := &executeFlags{}
	var orderID string
	var symbol string

	cmd := &cobra.Command{
		Use:   "cancel",
		Short: "Cancel a live pending order",
		RunE: func(cmd *cobra.Command, _ []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}

			intent, err := orderintent.NormalizeCancel(orderID, symbol)
			if err != nil {
				return err
			}

			preview := app.tradingService.PreviewCancel(intent)
			if !exec.execute {
				return output.WriteTradingPreview(cmd.OutOrStdout(), app.format, preview)
			}

			result, err := app.tradingService.Cancel(cmd.Context(), intent, trading.ExecuteOptions{
				Execute:                    exec.execute,
				DangerouslySkipPermissions: exec.dangerouslySkipPermissions,
				Confirm:                    exec.confirm,
			})
			if err != nil {
				return userFacingTradingError(app.paths, err)
			}
			recordMutationLineage(app, &result)

			return output.WriteMutationResult(cmd.OutOrStdout(), app.format, result)
		},
	}

	cmd.Flags().StringVar(&orderID, "order-id", "", "Pending order identifier")
	cmd.Flags().StringVar(&symbol, "symbol", "", "Trading symbol for the pending order")
	if err := cmd.MarkFlagRequired("order-id"); err != nil {
		panic(err)
	}
	if err := cmd.MarkFlagRequired("symbol"); err != nil {
		panic(err)
	}
	bindExecuteFlags(cmd, exec)
	return cmd
}

func newOrderAmendCmd(opts *rootOptions) *cobra.Command {
	flags := &amendFlags{}
	exec := &executeFlags{}

	cmd := &cobra.Command{
		Use:   "amend",
		Short: "Amend a live pending order",
		RunE: func(cmd *cobra.Command, _ []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}

			intent, err := orderintent.NormalizeAmend(flags.orderID, optionalFloat64(cmd, "quantity", flags.quantity), optionalFloat64(cmd, "price", flags.price))
			if err != nil {
				return err
			}

			preview := app.tradingService.PreviewAmend(intent)
			if !exec.execute {
				return output.WriteTradingPreview(cmd.OutOrStdout(), app.format, preview)
			}

			result, err := app.tradingService.Amend(cmd.Context(), intent, trading.ExecuteOptions{
				Execute:                    exec.execute,
				DangerouslySkipPermissions: exec.dangerouslySkipPermissions,
				Confirm:                    exec.confirm,
			})
			if err != nil {
				return userFacingTradingError(app.paths, err)
			}
			recordMutationLineage(app, &result)
			return output.WriteMutationResult(cmd.OutOrStdout(), app.format, result)
		},
	}

	cmd.Flags().StringVar(&flags.orderID, "order-id", "", "Pending order identifier")
	cmd.Flags().Float64Var(&flags.quantity, "quantity", 0, "Updated quantity")
	cmd.Flags().Float64Var(&flags.price, "price", 0, "Updated limit price")
	if err := cmd.MarkFlagRequired("order-id"); err != nil {
		panic(err)
	}
	bindExecuteFlags(cmd, exec)
	return cmd
}

func newOrderPermissionsCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "permissions",
		Short: "Manage temporary trading execution permissions",
	}

	var ttlSeconds int
	grantCmd := &cobra.Command{
		Use:   "grant",
		Short: "Grant a short-lived trading permission",
		RunE: func(cmd *cobra.Command, _ []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}
			if ttlSeconds <= 0 {
				return fmt.Errorf("ttl must be greater than zero seconds")
			}
			if err := app.tradingService.GrantEnabled(); err != nil {
				return userFacingTradingError(app.paths, err)
			}

			status, err := app.permissionService.Grant(cmd.Context(), time.Duration(ttlSeconds)*time.Second)
			if err != nil {
				return userFacingTradingError(app.paths, err)
			}

			return output.WritePermissionStatus(cmd.OutOrStdout(), app.format, status)
		},
	}
	grantCmd.Flags().IntVar(&ttlSeconds, "ttl", 300, "Permission TTL in seconds")

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Inspect the current trading permission state",
		RunE: func(cmd *cobra.Command, _ []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}

			status, err := app.permissionService.Status(cmd.Context())
			if err != nil {
				return err
			}

			return output.WritePermissionStatus(cmd.OutOrStdout(), app.format, status)
		},
	}

	revokeCmd := &cobra.Command{
		Use:   "revoke",
		Short: "Revoke any active trading permission",
		RunE: func(cmd *cobra.Command, _ []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}

			cleared, err := app.permissionService.Revoke(cmd.Context())
			if err != nil {
				return err
			}
			if app.format == output.FormatJSON {
				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetIndent("", "  ")
				return encoder.Encode(map[string]any{
					"active":          false,
					"expired":         false,
					"permission_file": app.paths.PermissionFile,
					"cleared":         cleared,
				})
			}
			if cleared {
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "Revoked trading permission: %s\n", app.paths.PermissionFile)
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "No active trading permission found at: %s\n", app.paths.PermissionFile)
			return err
		},
	}

	cmd.AddCommand(grantCmd, statusCmd, revokeCmd)
	return cmd
}

func defaultPlaceFlags() *placeFlags {
	return &placeFlags{
		market:       "us",
		orderType:    "limit",
		currencyMode: "KRW",
	}
}

func recoverOrderWithLineageHint(ctx context.Context, app *appContext, requestedOrderID, lineageHintKey, market string) (domain.Order, bool, error) {
	if app == nil || app.lineageService == nil {
		return domain.Order{}, false, nil
	}

	entry, ok, err := app.lineageService.Lookup(lineageHintKey)
	if err != nil {
		return domain.Order{}, false, err
	}
	if !ok {
		return domain.Order{}, false, nil
	}

	order, recovered, err := app.client.FindCompletedOrderFromLineageHint(ctx, requestedOrderID, market, entry)
	if err != nil || !recovered {
		return order, recovered, err
	}

	if err := app.lineageService.Record(lineageHintKey, orderlineage.Entry{
		CurrentOrderID: order.ID,
		Kind:           entry.Kind,
		Symbol:         entry.Symbol,
		Market:         entry.Market,
		Quantity:       entry.Quantity,
		Price:          entry.Price,
		OrderDate:      entry.OrderDate,
	}); err != nil {
		return order, true, nil
	}

	return order, true, nil
}

func bindPlaceFlags(cmd *cobra.Command, flags *placeFlags) {
	cmd.Flags().StringVar(&flags.symbol, "symbol", "", "Trading symbol")
	cmd.Flags().StringVar(&flags.market, "market", flags.market, "Market identifier")
	cmd.Flags().StringVar(&flags.side, "side", "", "Order side: buy or sell")
	cmd.Flags().StringVar(&flags.orderType, "type", flags.orderType, "Order type: limit or market")
	cmd.Flags().Float64Var(&flags.quantity, "qty", 0, "Order quantity")
	cmd.Flags().Float64Var(&flags.price, "price", 0, "Order price for limit orders")
	cmd.Flags().Float64Var(&flags.amount, "amount", 0, "Order amount in KRW for fractional orders")
	cmd.Flags().StringVar(&flags.currencyMode, "currency-mode", flags.currencyMode, "Currency mode")
	cmd.Flags().BoolVar(&flags.fractional, "fractional", false, "Whether the order is fractional (US market order, amount-based)")
	if err := cmd.MarkFlagRequired("symbol"); err != nil {
		panic(err)
	}
	if err := cmd.MarkFlagRequired("side"); err != nil {
		panic(err)
	}
	if err := cmd.MarkFlagRequired("qty"); err != nil {
		panic(err)
	}
}

func bindExecuteFlags(cmd *cobra.Command, flags *executeFlags) {
	cmd.Flags().BoolVar(&flags.execute, "execute", false, "Attempt the live mutation instead of local preview only")
	cmd.Flags().BoolVar(&flags.dangerouslySkipPermissions, "dangerously-skip-permissions", false, "Acknowledge that this would execute a live trading mutation")
	cmd.Flags().StringVar(&flags.confirm, "confirm", "", "Confirmation token from a canonical preview")
}

func optionalFloat64(cmd *cobra.Command, name string, value float64) *float64 {
	if !cmd.Flags().Changed(name) {
		return nil
	}
	return &value
}

func recordMutationLineage(app *appContext, result *trading.MutationResult) {
	if app == nil || app.lineageService == nil || result == nil {
		return
	}

	originalOrderID := strings.TrimSpace(result.OriginalOrderID)
	if originalOrderID == "" {
		return
	}

	entry := orderlineage.Entry{
		CurrentOrderID: strings.TrimSpace(result.CurrentOrderID),
		Kind:           strings.TrimSpace(result.Kind),
		Symbol:         strings.TrimSpace(result.Symbol),
		Market:         strings.TrimSpace(result.Market),
		Quantity:       result.Quantity,
		Price:          result.Price,
		OrderDate:      strings.TrimSpace(result.OrderDate),
	}
	if err := app.lineageService.Record(originalOrderID, entry); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Could not update local lineage cache at %s: %v", app.paths.LineageFile, err))
	}
}
