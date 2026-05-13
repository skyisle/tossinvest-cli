package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/junghoonkye/tossinvest-cli/internal/monitor"
	"github.com/spf13/cobra"
)

const monitorWebhookEnv = "TOSSCTL_MONITOR_WEBHOOK"

func newMonitorCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "monitor",
		Short: "Periodic health checks against Toss read-only endpoints",
	}

	var webhookURL string
	var quiet bool

	apiCmd := &cobra.Command{
		Use:   "api",
		Short: "Run schema-invariant probes; exit 1 on any failure",
		Long: `Run schema-invariant probes against the read-only Toss endpoints the
CLI depends on. Designed for cron / launchd:

  • Exits 0 when every probe passes, 1 if any probe fails.
  • Optional Discord webhook (--webhook URL or TOSSCTL_MONITOR_WEBHOOK env)
    receives a one-line summary per failed probe; passing probes stay quiet
    so the channel only pings on real regressions.
  • No account numbers, cookies, dollar amounts, or response-body content
    are sent to the webhook — only probe name, HTTP method+path, status
    code, and a schema-diagnosis message (e.g. "result.sections is empty").
  • The webhook URL is never defaulted in code; each user sets their own.
    The tool runs entirely on your machine against your own session — it
    does not collect anyone else's data.

Use this to catch server-side changes (like the body-contract change in
#29) before users do.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app, err := newAppContext(opts)
			if err != nil {
				return err
			}
			if app.session == nil {
				return errors.New("no active session; run `tossctl auth login` first")
			}

			if webhookURL == "" {
				webhookURL = os.Getenv(monitorWebhookEnv)
			}

			results := monitor.Run(cmd.Context(), app.session)
			printResults(cmd.OutOrStdout(), cmd.OutOrStderr(), results, quiet)

			if err := monitor.PostDiscord(context.Background(), webhookURL, results); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "discord webhook: %v\n", err)
				// Don't fail the command on webhook errors — the probe result
				// is the authoritative signal, webhook is informational.
			}

			for _, r := range results {
				if !r.OK {
					os.Exit(1)
				}
			}
			return nil
		},
	}
	apiCmd.Flags().StringVar(&webhookURL, "webhook", "", "Discord webhook URL for failure alerts (or set "+monitorWebhookEnv+")")
	apiCmd.Flags().BoolVar(&quiet, "quiet", false, "Only print failed probes")

	cmd.AddCommand(apiCmd)
	return cmd
}

func printResults(stdout, stderr interface {
	Write([]byte) (int, error)
}, results []monitor.Result, quiet bool) {
	pass, fail := 0, 0
	for _, r := range results {
		if r.OK {
			pass++
		} else {
			fail++
		}
	}
	if !quiet {
		for _, r := range results {
			if r.OK {
				fmt.Fprintf(stdout, "  ✓ %s — status=%d (%dms)\n", r.Probe.Name, r.Status, r.Duration.Milliseconds())
			}
		}
	}
	for _, r := range results {
		if !r.OK {
			fmt.Fprintf(stderr, "  ✗ %s — status=%d: %s\n", r.Probe.Name, r.Status, r.Detail)
		}
	}
	fmt.Fprintf(stdout, "\n%d passed, %d failed\n", pass, fail)
}
