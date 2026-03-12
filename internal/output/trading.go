package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/junghoonkye/tossinvest-cli/internal/permissions"
	"github.com/junghoonkye/tossinvest-cli/internal/trading"
)

func WriteTradingPreview(w io.Writer, format Format, preview trading.Preview) error {
	switch format {
	case FormatJSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(preview)
	case FormatCSV:
		writer := csv.NewWriter(w)
		if err := writer.Write([]string{"kind", "confirm_token", "canonical", "warnings"}); err != nil {
			return err
		}
		if err := writer.Write([]string{
			preview.Kind,
			preview.ConfirmToken,
			preview.Canonical,
			strconv.Quote(fmt.Sprintf("%v", preview.Warnings)),
		}); err != nil {
			return err
		}
		writer.Flush()
		return writer.Error()
	case FormatTable:
		if _, err := fmt.Fprintf(
			w,
			"Kind: %s\nConfirm Token: %s\nCanonical: %s\nLive Ready: %t\nMutation Ready: %t\n",
			preview.Kind,
			preview.ConfirmToken,
			preview.Canonical,
			preview.LiveReady,
			preview.MutationReady,
		); err != nil {
			return err
		}
		if len(preview.Warnings) == 0 {
			return nil
		}
		if _, err := fmt.Fprintln(w, "Warnings:"); err != nil {
			return err
		}
		for _, warning := range preview.Warnings {
			if _, err := fmt.Fprintf(w, "- %s\n", warning); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func WritePermissionStatus(w io.Writer, format Format, status permissions.Status) error {
	switch format {
	case FormatJSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(status)
	case FormatCSV:
		writer := csv.NewWriter(w)
		if err := writer.Write([]string{"active", "expired", "remaining_seconds", "permission_file"}); err != nil {
			return err
		}
		if err := writer.Write([]string{
			strconv.FormatBool(status.Active),
			strconv.FormatBool(status.Expired),
			strconv.FormatInt(status.Remaining, 10),
			status.PermissionFile,
		}); err != nil {
			return err
		}
		writer.Flush()
		return writer.Error()
	case FormatTable:
		state := "inactive"
		if status.Active {
			state = "active"
		} else if status.Expired {
			state = "expired"
		}
		if _, err := fmt.Fprintf(w, "Trading Permission: %s\nPermission File: %s\n", state, status.PermissionFile); err != nil {
			return err
		}
		if status.ExpiresAt != nil {
			if _, err := fmt.Fprintf(w, "Expires At: %s\n", status.ExpiresAt.Format("2006-01-02 15:04:05Z07:00")); err != nil {
				return err
			}
		}
		if status.Active {
			_, err := fmt.Fprintf(w, "Remaining Seconds: %d\n", status.Remaining)
			return err
		}
		return nil
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}
