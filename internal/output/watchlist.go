package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

func WriteWatchlist(w io.Writer, format Format, items []domain.WatchlistItem) error {
	switch format {
	case FormatJSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(items)
	case FormatCSV:
		writer := csv.NewWriter(w)
		if err := writer.Write([]string{"group", "symbol", "name", "currency", "base", "last"}); err != nil {
			return err
		}
		for _, item := range items {
			if err := writer.Write([]string{
				item.Group,
				item.Symbol,
				item.Name,
				item.Currency,
				formatFloat(item.Base),
				formatFloat(item.Last),
			}); err != nil {
				return err
			}
		}
		writer.Flush()
		return writer.Error()
	case FormatTable:
		for _, item := range items {
			if _, err := fmt.Fprintf(
				w,
				"- [%s] %s %s base=%s last=%s %s\n",
				item.Group,
				item.Symbol,
				item.Name,
				formatFloat(item.Base),
				formatFloat(item.Last),
				item.Currency,
			); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}
