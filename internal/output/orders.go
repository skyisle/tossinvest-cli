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
	return writeOrderList(w, format, "Pending Orders", "No pending orders", orders)
}

func WriteCompletedOrders(w io.Writer, format Format, orders []domain.Order) error {
	return writeOrderList(w, format, "Completed Orders", "No completed orders", orders)
}

func WriteOrder(w io.Writer, format Format, order domain.Order) error {
	switch format {
	case FormatJSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(order)
	case FormatCSV:
		writer := csv.NewWriter(w)
		if err := writer.Write([]string{"id", "resolved_from_id", "symbol", "name", "market", "side", "status", "quantity", "filled_quantity", "price", "average_execution_price", "order_date", "submitted_at"}); err != nil {
			return err
		}
		submittedAt := ""
		if order.SubmittedAt != nil {
			submittedAt = order.SubmittedAt.Format("2006-01-02T15:04:05Z07:00")
		}
		if err := writer.Write([]string{
			order.ID,
			order.ResolvedFromID,
			order.Symbol,
			order.Name,
			order.Market,
			order.Side,
			order.Status,
			strconv.FormatFloat(order.Quantity, 'f', -1, 64),
			strconv.FormatFloat(order.FilledQuantity, 'f', -1, 64),
			strconv.FormatFloat(order.Price, 'f', -1, 64),
			strconv.FormatFloat(order.AverageExecutionPrice, 'f', -1, 64),
			order.OrderDate,
			submittedAt,
		}); err != nil {
			return err
		}
		writer.Flush()
		return writer.Error()
	case FormatTable:
		if _, err := fmt.Fprintf(w, "Order ID: %s\n", order.ID); err != nil {
			return err
		}
		if order.ResolvedFromID != "" {
			if _, err := fmt.Fprintf(w, "Resolved From: %s\n", order.ResolvedFromID); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(w, "Symbol: %s\n", order.Symbol); err != nil {
			return err
		}
		if order.Name != "" {
			if _, err := fmt.Fprintf(w, "Name: %s\n", order.Name); err != nil {
				return err
			}
		}
		if order.Market != "" {
			if _, err := fmt.Fprintf(w, "Market: %s\n", order.Market); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(w, "Side: %s\nStatus: %s\n", order.Side, order.Status); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "Quantity: %s\n", strconv.FormatFloat(order.Quantity, 'f', -1, 64)); err != nil {
			return err
		}
		if order.FilledQuantity > 0 {
			if _, err := fmt.Fprintf(w, "Filled Quantity: %s\n", strconv.FormatFloat(order.FilledQuantity, 'f', -1, 64)); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(w, "Price: %s\n", strconv.FormatFloat(order.Price, 'f', -1, 64)); err != nil {
			return err
		}
		if order.AverageExecutionPrice > 0 {
			if _, err := fmt.Fprintf(w, "Average Execution Price: %s\n", strconv.FormatFloat(order.AverageExecutionPrice, 'f', -1, 64)); err != nil {
				return err
			}
		}
		if order.OrderDate != "" {
			if _, err := fmt.Fprintf(w, "Order Date: %s\n", order.OrderDate); err != nil {
				return err
			}
		}
		if order.SubmittedAt != nil {
			if _, err := fmt.Fprintf(w, "Submitted At: %s\n", order.SubmittedAt.Format("2006-01-02 15:04:05Z07:00")); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func writeOrderList(w io.Writer, format Format, title, emptyMessage string, orders []domain.Order) error {
	switch format {
	case FormatJSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(orders)
	case FormatCSV:
		writer := csv.NewWriter(w)
		if err := writer.Write([]string{"id", "symbol", "name", "market", "side", "status", "quantity", "filled_quantity", "price", "average_execution_price", "order_date", "submitted_at"}); err != nil {
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
				order.Name,
				order.Market,
				order.Side,
				order.Status,
				strconv.FormatFloat(order.Quantity, 'f', -1, 64),
				strconv.FormatFloat(order.FilledQuantity, 'f', -1, 64),
				strconv.FormatFloat(order.Price, 'f', -1, 64),
				strconv.FormatFloat(order.AverageExecutionPrice, 'f', -1, 64),
				order.OrderDate,
				submittedAt,
			}); err != nil {
				return err
			}
		}
		writer.Flush()
		return writer.Error()
	case FormatTable:
		if len(orders) == 0 {
			_, err := fmt.Fprintln(w, emptyMessage)
			return err
		}
		if _, err := fmt.Fprintf(w, "%s: %d\n", title, len(orders)); err != nil {
			return err
		}
		for _, order := range orders {
			if _, err := fmt.Fprintf(
				w,
				"- %s %s %s qty=%s filled=%s price=%s id=%s\n",
				order.Symbol,
				order.Side,
				order.Status,
				strconv.FormatFloat(order.Quantity, 'f', -1, 64),
				strconv.FormatFloat(order.FilledQuantity, 'f', -1, 64),
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
