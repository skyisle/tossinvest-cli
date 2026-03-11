package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

func WriteOrders(w io.Writer, format Format, orders []domain.Order) error {
	switch format {
	case FormatJSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(orders)
	case FormatCSV:
		writer := csv.NewWriter(w)
		if err := writer.Write([]string{"id", "symbol", "side", "status", "quantity", "price", "submitted_at"}); err != nil {
			return err
		}
		for _, order := range orders {
			submittedAt := ""
			if order.SubmittedAt != nil {
				submittedAt = order.SubmittedAt.Format("2006-01-02T15:04:05Z07:00")
			}
			if err := writer.Write([]string{
				order.ID,
				order.Symbol,
				order.Side,
				order.Status,
				strconv.FormatFloat(order.Quantity, 'f', -1, 64),
				strconv.FormatFloat(order.Price, 'f', -1, 64),
				submittedAt,
			}); err != nil {
				return err
			}
		}
		writer.Flush()
		return writer.Error()
	case FormatTable:
		if len(orders) == 0 {
			_, err := fmt.Fprintln(w, "No pending orders")
			return err
		}
		if _, err := fmt.Fprintf(w, "Pending Orders: %d\n", len(orders)); err != nil {
			return err
		}
		for _, order := range orders {
			if _, err := fmt.Fprintf(
				w,
				"- %s %s %s qty=%s price=%s id=%s\n",
				order.Symbol,
				order.Side,
				order.Status,
				strconv.FormatFloat(order.Quantity, 'f', -1, 64),
				strconv.FormatFloat(order.Price, 'f', -1, 64),
				order.ID,
			); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}
