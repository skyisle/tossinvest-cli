package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

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
			"config_file", "exists", "schema_version", "grant", "place", "cancel", "amend", "allow_dangerous_execute",
		}); err != nil {
			return err
		}
		if err := writer.Write([]string{
			status.ConfigFile,
			strconv.FormatBool(status.Exists),
			strconv.Itoa(status.SchemaVersion),
			strconv.FormatBool(status.Trading.Grant),
			strconv.FormatBool(status.Trading.Place),
			strconv.FormatBool(status.Trading.Cancel),
			strconv.FormatBool(status.Trading.Amend),
			strconv.FormatBool(status.Trading.AllowDangerousExecute),
		}); err != nil {
			return err
		}
		writer.Flush()
		return writer.Error()
	case FormatTable:
		_, err := fmt.Fprintf(
			w,
			"Config File: %s\nExists: %t\nSchema: %s\nSchema Version: %d\nTrading Grant: %t\nTrading Place: %t\nTrading Cancel: %t\nTrading Amend: %t\nAllow Dangerous Execute: %t\n",
			status.ConfigFile,
			status.Exists,
			status.Schema,
			status.SchemaVersion,
			status.Trading.Grant,
			status.Trading.Place,
			status.Trading.Cancel,
			status.Trading.Amend,
			status.Trading.AllowDangerousExecute,
		)
		return err
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}
