package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

func WritePositions(w io.Writer, format Format, positions []domain.Position) error {
	switch format {
	case FormatJSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(positions)
	case FormatCSV:
		writer := csv.NewWriter(w)
		if err := writer.Write([]string{"product_code", "symbol", "name", "market_type", "market_code", "quantity", "average_price", "current_price", "market_value", "unrealized_pnl", "profit_rate", "daily_profit_loss", "daily_profit_rate"}); err != nil {
			return err
		}
		for _, position := range positions {
			if err := writer.Write([]string{
				position.ProductCode,
				position.Symbol,
				position.Name,
				position.MarketType,
				position.MarketCode,
				formatFloat(position.Quantity),
				formatFloat(position.AveragePrice),
				formatFloat(position.CurrentPrice),
				formatFloat(position.MarketValue),
				formatFloat(position.UnrealizedPnL),
				formatFloat(position.ProfitRate),
				formatFloat(position.DailyProfitLoss),
				formatFloat(position.DailyProfitRate),
			}); err != nil {
				return err
			}
		}
		writer.Flush()
		return writer.Error()
	case FormatTable:
		for _, position := range positions {
			if _, err := fmt.Fprintf(
				w,
				"- %s (%s) qty=%s avg=%s last=%s value=%s pnl=%s rate=%.2f%% day_pnl=%s day_rate=%.2f%%\n",
				position.Name,
				position.Symbol,
				formatFloat(position.Quantity),
				formatFloat(position.AveragePrice),
				formatFloat(position.CurrentPrice),
				formatFloat(position.MarketValue),
				formatFloat(position.UnrealizedPnL),
				position.ProfitRate*100,
				formatFloat(position.DailyProfitLoss),
				position.DailyProfitRate*100,
			); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func WriteAllocation(w io.Writer, format Format, markets map[string]domain.AccountMarketSummary) error {
	switch format {
	case FormatJSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(markets)
	case FormatCSV:
		writer := csv.NewWriter(w)
		if err := writer.Write([]string{"market", "total_asset_amount", "principal_amount", "evaluated_profit_amount", "profit_rate"}); err != nil {
			return err
		}
		keys := sortedMarketKeys(markets)
		for _, key := range keys {
			market := markets[key]
			if err := writer.Write([]string{
				key,
				formatFloat(market.TotalAssetAmount),
				formatFloat(market.PrincipalAmount),
				formatFloat(market.EvaluatedProfitAmount),
				formatFloat(market.ProfitRate),
			}); err != nil {
				return err
			}
		}
		writer.Flush()
		return writer.Error()
	case FormatTable:
		keys := sortedMarketKeys(markets)
		for _, key := range keys {
			market := markets[key]
			if _, err := fmt.Fprintf(
				w,
				"- %s: total=%s principal=%s profit=%s rate=%.2f%%\n",
				key,
				formatFloat(market.TotalAssetAmount),
				formatFloat(market.PrincipalAmount),
				formatFloat(market.EvaluatedProfitAmount),
				market.ProfitRate*100,
			); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func sortedMarketKeys(markets map[string]domain.AccountMarketSummary) []string {
	keys := make([]string, 0, len(markets))
	for key := range markets {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
