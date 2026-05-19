package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

// WriteProfitOverview renders the lifetime/period aggregate from
// /api/v1/profit/overview. Table mode prints KRW totals because the WTS UI
// defaults to KRW for the aggregate cards; JSON/CSV preserve both KRW and USD.
func WriteProfitOverview(w io.Writer, format Format, overview domain.ProfitOverview) error {
	switch format {
	case FormatJSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(overview)
	case FormatCSV:
		writer := csv.NewWriter(w)
		if err := writer.Write([]string{"section", "metric", "krw", "usd"}); err != nil {
			return err
		}
		rows := [][]string{
			{"total", "asset_amount", f(overview.TotalAssetAmount.KRW), f(overview.TotalAssetAmount.USD)},
			{"total", "earning_amount", f(overview.EarningAmount.KRW), f(overview.EarningAmount.USD)},
		}
		legs := []struct {
			name string
			leg  domain.ProfitOverviewLeg
		}{
			{"sales", overview.Sales},
			{"lending", overview.Lending},
			{"dividend", overview.Dividend},
			{"maturity", overview.Maturity},
		}
		for _, l := range legs {
			rows = append(rows,
				[]string{l.name, "amount", f(l.leg.Amount.KRW), f(l.leg.Amount.USD)},
				[]string{l.name, "earning_rate", f(l.leg.EarningRate.KRW), f(l.leg.EarningRate.USD)},
			)
			if l.leg.PurchaseAmount != nil {
				rows = append(rows, []string{l.name, "purchase_amount", f(l.leg.PurchaseAmount.KRW), f(l.leg.PurchaseAmount.USD)})
			}
		}
		rows = append(rows, []string{"interest", "amount", f(overview.Interest), ""})
		for _, r := range rows {
			if err := writer.Write(r); err != nil {
				return err
			}
		}
		writer.Flush()
		return writer.Error()
	case FormatTable:
		fmt.Fprintf(w, "Range: %s  %s ~ %s\n", overview.RangeType, overview.StartDate, overview.EndDate)
		fmt.Fprintf(w, "Total earning: %s\n", formatKRW(overview.EarningAmount.KRW))
		fmt.Fprintln(w)

		headers := []string{"항목", "금액", "수익률", "원금"}
		legs := []struct {
			name string
			leg  domain.ProfitOverviewLeg
		}{
			{"판매수익", overview.Sales},
			{"배당", overview.Dividend},
			{"대여", overview.Lending},
			{"만기", overview.Maturity},
		}
		rows := make([][]string, 0, len(legs)+1)
		for _, l := range legs {
			rate := fmt.Sprintf("%.2f%%", l.leg.EarningRate.KRW)
			principal := "-"
			if l.leg.PurchaseAmount != nil {
				principal = formatKRW(l.leg.PurchaseAmount.KRW)
			}
			rows = append(rows, []string{
				l.name,
				formatKRW(l.leg.Amount.KRW),
				rate,
				principal,
			})
		}
		rows = append(rows, []string{"이자", formatKRW(overview.Interest), "-", "-"})
		return renderTable(w, headers, rows)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// WriteProfitDaily renders the per-symbol daily realized P&L list.
func WriteProfitDaily(w io.Writer, format Format, items []domain.ProfitDailyItem) error {
	switch format {
	case FormatJSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(items)
	case FormatCSV:
		writer := csv.NewWriter(w)
		if err := writer.Write([]string{
			"date", "market", "symbol", "name", "quantity",
			"profit_loss_krw", "profit_loss_usd", "profit_rate",
			"sell_amount_krw", "sell_amount_usd",
			"buy_amount_krw", "buy_amount_usd",
			"commission_krw", "commission_usd",
			"tax_krw", "tax_usd",
			"exchange_profit_loss",
		}); err != nil {
			return err
		}
		for _, it := range items {
			sym := it.Symbol
			if sym == "" {
				sym = it.ProductCode
			}
			fxPnL := ""
			if it.ExchangeProfitLossAmount != nil {
				fxPnL = f(*it.ExchangeProfitLossAmount)
			}
			if err := writer.Write([]string{
				it.BaseDate,
				strings.ToUpper(it.MarketType),
				sym,
				it.ProductName,
				strconv.FormatFloat(it.Quantity, 'f', -1, 64),
				f(it.ProfitLossAmount.KRW), f(it.ProfitLossAmount.USD),
				strconv.FormatFloat(it.ProfitRate, 'f', -1, 64),
				f(it.SellAmount.KRW), f(it.SellAmount.USD),
				f(it.BuyAmount.KRW), f(it.BuyAmount.USD),
				f(it.Commission.KRW), f(it.Commission.USD),
				f(it.Tax.KRW), f(it.Tax.USD),
				fxPnL,
			}); err != nil {
				return err
			}
		}
		writer.Flush()
		return writer.Error()
	case FormatTable:
		if len(items) == 0 {
			_, err := fmt.Fprintln(w, "No realized profit in range")
			return err
		}
		if _, err := fmt.Fprintf(w, "Realized P&L items: %d\n", len(items)); err != nil {
			return err
		}
		headers := []string{"일자", "시장", "종목", "수량", "손익", "수익률", "판매금액", "구매금액"}
		rows := make([][]string, 0, len(items))
		for _, it := range items {
			rows = append(rows, []string{
				it.BaseDate,
				strings.ToUpper(it.MarketType),
				truncateName(it.ProductName, 16),
				formatQty(it.Quantity),
				pickPrimaryAmount(it.MarketType, it.ProfitLossAmount.KRW, it.ProfitLossAmount.USD),
				fmt.Sprintf("%.2f%%", it.ProfitRate),
				pickPrimaryAmount(it.MarketType, it.SellAmount.KRW, it.SellAmount.USD),
				pickPrimaryAmount(it.MarketType, it.BuyAmount.KRW, it.BuyAmount.USD),
			})
		}
		return renderTable(w, headers, rows)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// WriteProfitGroup renders the per-symbol aggregate for one market.
func WriteProfitGroup(w io.Writer, format Format, page domain.ProfitGroupPage) error {
	switch format {
	case FormatJSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(page)
	case FormatCSV:
		writer := csv.NewWriter(w)
		if err := writer.Write([]string{
			"market", "profit_type", "symbol", "product_code", "name", "quantity",
			"earning_amount_krw", "earning_amount_usd", "earning_rate_krw",
			"purchase_amount_krw", "purchase_amount_usd",
			"commission_krw", "tax_krw",
		}); err != nil {
			return err
		}
		for _, it := range page.Items {
			sym := it.Symbol
			if sym == "" {
				sym = it.ProductCode
			}
			if err := writer.Write([]string{
				page.Market,
				it.ProfitType,
				sym,
				it.ProductCode,
				it.Name,
				strconv.FormatFloat(it.Quantity, 'f', -1, 64),
				f(it.EarningAmount.KRW), f(it.EarningAmount.USD),
				f(it.EarningRate.KRW),
				f(it.PurchaseAmount.KRW), f(it.PurchaseAmount.USD),
				f(it.Commission.KRW),
				f(it.Tax.KRW),
			}); err != nil {
				return err
			}
		}
		writer.Flush()
		return writer.Error()
	case FormatTable:
		if _, err := fmt.Fprintf(w,
			"Market: %s  Type: %s  Items: %d\n",
			strings.ToUpper(page.Market), page.ProfitType, len(page.Items)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w,
			"Total earning: %s (%.2f%%)\n\n",
			pickPrimaryAmount(page.Market, page.TotalEarningAmount.KRW, page.TotalEarningAmount.USD),
			page.TotalEarningRate.KRW,
		); err != nil {
			return err
		}
		if len(page.Items) == 0 {
			_, err := fmt.Fprintln(w, "No items.")
			return err
		}
		headers := []string{"종목", "수량", "손익", "수익률", "원금"}
		rows := make([][]string, 0, len(page.Items))
		for _, it := range page.Items {
			rows = append(rows, []string{
				truncateName(it.Name, 20),
				formatQty(it.Quantity),
				pickPrimaryAmount(page.Market, it.EarningAmount.KRW, it.EarningAmount.USD),
				fmt.Sprintf("%.2f%%", it.EarningRate.KRW),
				pickPrimaryAmount(page.Market, it.PurchaseAmount.KRW, it.PurchaseAmount.USD),
			})
		}
		return renderTable(w, headers, rows)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// f formats a float for CSV cells with a trimmed fractional part.
func f(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
