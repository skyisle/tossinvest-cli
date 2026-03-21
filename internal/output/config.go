package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/junghoonkye/tossinvest-cli/internal/config"
)

func WriteConfigStatus(w io.Writer, format Format, status config.Status) error {
	switch format {
	case FormatJSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(status)
	case FormatCSV:
		writer := csv.NewWriter(w)
		if err := writer.Write([]string{
			"config_file", "exists", "schema_version", "source_schema_version", "grant", "place", "sell", "kr", "fractional", "cancel", "amend", "allow_live_order_actions", "complete_trade_auth", "accept_product_ack", "accept_fx_consent", "legacy_fields",
		}); err != nil {
			return err
		}
		if err := writer.Write([]string{
			status.ConfigFile,
			strconv.FormatBool(status.Exists),
			strconv.Itoa(status.SchemaVersion),
			strconv.Itoa(status.SourceSchemaVersion),
			strconv.FormatBool(status.Trading.Grant),
			strconv.FormatBool(status.Trading.Place),
			strconv.FormatBool(status.Trading.Sell),
			strconv.FormatBool(status.Trading.KR),
			strconv.FormatBool(status.Trading.Fractional),
			strconv.FormatBool(status.Trading.Cancel),
			strconv.FormatBool(status.Trading.Amend),
			strconv.FormatBool(status.Trading.AllowLiveOrderActions),
			strconv.FormatBool(status.Trading.DangerousAutomation.CompleteTradeAuth),
			strconv.FormatBool(status.Trading.DangerousAutomation.AcceptProductAck),
			strconv.FormatBool(status.Trading.DangerousAutomation.AcceptFXConsent),
			strings.Join(status.LegacyFields, "|"),
		}); err != nil {
			return err
		}
		writer.Flush()
		return writer.Error()
	case FormatTable:
		if _, err := fmt.Fprintf(
			w,
			"Config File: %s\nExists: %t\nSchema: %s\nSchema Version: %d\n",
			status.ConfigFile,
			status.Exists,
			status.Schema,
			status.SchemaVersion,
		); err != nil {
			return err
		}
		if status.SourceSchemaVersion > 0 && status.SourceSchemaVersion != status.SchemaVersion {
			if _, err := fmt.Fprintf(w, "Source Schema Version: %d\n", status.SourceSchemaVersion); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(
			w,
			"Trading Grant: %t\nTrading Place: %t\nTrading Sell: %t\nTrading KR: %t\nTrading Fractional: %t\nTrading Cancel: %t\nTrading Amend: %t\nAllow Live Order Actions: %t\nDangerous Automation: %s\n",
			status.Trading.Grant,
			status.Trading.Place,
			status.Trading.Sell,
			status.Trading.KR,
			status.Trading.Fractional,
			status.Trading.Cancel,
			status.Trading.Amend,
			status.Trading.AllowLiveOrderActions,
			formatDangerousAutomation(status.Trading.DangerousAutomation),
		); err != nil {
			return err
		}
		if len(status.LegacyFields) == 0 {
			return nil
		}
		_, err := fmt.Fprintf(w, "Legacy Fields: %s\n", strings.Join(status.LegacyFields, ", "))
		return err
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func formatDangerousAutomation(value config.DangerousAutomation) string {
	enabled := value.EnabledActions()
	if len(enabled) == 0 {
		return "none"
	}
	return strings.Join(enabled, ", ")
}
