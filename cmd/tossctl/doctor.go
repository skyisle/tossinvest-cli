package main

import (
	"encoding/json"

	"github.com/junghoonkye/tossinvest-cli/internal/doctor"
	"github.com/junghoonkye/tossinvest-cli/internal/output"
	"github.com/spf13/cobra"
)

func newDoctorCmd(opts *rootOptions) *cobra.Command {
	var reportMode bool

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check whether tossctl is ready on this machine",
		Long: `Check whether tossctl is ready on this machine.

Use --report to emit a JSON diagnostic bundle (endpoint-family probes, file
permissions, orphan cache files). Paths are redacted to '~' so the output
can be pasted into bug reports without leaking the local username.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}
			configStatus, err := app.configService.Status(cmd.Context())
			if err != nil {
				return err
			}

			svc := doctor.NewService(
				app.paths,
				configStatus,
				app.loginConfig,
				app.authService,
				app.permissionService,
			)

			if reportMode {
				report, err := svc.RunReport(cmd.Context(), app.client)
				if err != nil {
					return err
				}
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(report)
			}

			report, err := svc.Run(cmd.Context())
			if err != nil {
				return err
			}

			return output.WriteDoctorReport(cmd.OutOrStdout(), app.format, report)
		},
	}

	cmd.Flags().BoolVar(&reportMode, "report", false, "Emit a JSON diagnostic bundle suitable for bug reports (runs endpoint probes, redacts paths)")

	return cmd
}
