package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

func WriteQuote(w io.Writer, format Format, quote domain.Quote) error {
	switch format {
	case FormatTable:
		return writeQuoteTable(w, quote)
	case FormatJSON:
		return writeQuoteJSON(w, quote)
	case FormatCSV:
		return writeQuoteCSV(w, quote)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func writeQuoteJSON(w io.Writer, quote domain.Quote) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(quote)
}

func writeQuoteCSV(w io.Writer, quote domain.Quote) error {
	writer := csv.NewWriter(w)
	if err := writer.Write([]string{
		"product_code",
		"symbol",
		"name",
		"market_code",
		"market",
		"currency",
		"reference_price",
		"last",
		"change",
		"change_rate",
		"volume",
		"status",
		"badge_count",
		"notice_count",
		"fetched_at",
	}); err != nil {
		return err
	}

	if err := writer.Write([]string{
		quote.ProductCode,
		quote.Symbol,
		quote.Name,
		quote.MarketCode,
		quote.Market,
		quote.Currency,
		formatFloat(quote.ReferencePrice),
		formatFloat(quote.Last),
		formatFloat(quote.Change),
		formatFloat(quote.ChangeRate),
		formatFloat(quote.Volume),
		quote.Status,
		strconv.Itoa(quote.BadgeCount),
		strconv.Itoa(quote.NoticeCount),
		quote.FetchedAt.Format("2006-01-02T15:04:05Z07:00"),
	}); err != nil {
		return err
	}

	writer.Flush()
	return writer.Error()
}

func writeQuoteTable(w io.Writer, quote domain.Quote) error {
	_, err := fmt.Fprintf(
		w,
		"Product Code: %s\nSymbol: %s\nName: %s\nMarket: %s (%s)\nCurrency: %s\nReference Price: %s\nLast: %s\nChange: %s\nChange Rate: %.2f%%\nVolume: %s\nStatus: %s\nBadges: %d\nNotices: %d\nFetched At: %s\n",
		quote.ProductCode,
		quote.Symbol,
		quote.Name,
		quote.Market,
		quote.MarketCode,
		quote.Currency,
		formatFloat(quote.ReferencePrice),
		formatFloat(quote.Last),
		formatFloat(quote.Change),
		quote.ChangeRate*100,
		formatFloat(quote.Volume),
		quote.Status,
		quote.BadgeCount,
		quote.NoticeCount,
		quote.FetchedAt.Format("2006-01-02 15:04:05Z07:00"),
	)
	return err
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
