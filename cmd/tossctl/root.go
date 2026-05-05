package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/auth"
	tossclient "github.com/junghoonkye/tossinvest-cli/internal/client"
	"github.com/junghoonkye/tossinvest-cli/internal/config"
	"github.com/junghoonkye/tossinvest-cli/internal/orderlineage"
	"github.com/junghoonkye/tossinvest-cli/internal/output"
	"github.com/junghoonkye/tossinvest-cli/internal/permissions"
	"github.com/junghoonkye/tossinvest-cli/internal/session"
	"github.com/junghoonkye/tossinvest-cli/internal/trading"
	"github.com/spf13/cobra"
)

type rootOptions struct {
	outputFormat string
	configDir    string
	sessionFile  string
}

type appContext struct {
	format            output.Format
	paths             config.Paths
	config            config.File
	configService     *config.Service
	loginConfig       auth.LoginConfig
	authService       *auth.Service
	client            *tossclient.Client
	session           *session.Session
	permissionService *permissions.Service
	lineageService    *orderlineage.Service
	tradingService    *trading.Service
}

func newRootCmd() *cobra.Command {
	opts := &rootOptions{}

	cmd := &cobra.Command{
		Use:   "tossctl",
		Short: "CLI for Toss Securities web data and trading experiments",
		Long: "tossctl is the CLI binary for tossinvest-cli, an unofficial Toss Securities " +
			"web client with browser-assisted login and a narrow trading beta surface.",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			format, err := output.ParseFormat(opts.outputFormat)
			if err != nil {
				return err
			}
			store := session.NewFileStore(resolveSessionFile(opts))
			sess, _ := store.Load(cmd.Context())
			writeExpiryWarningIfNeeded(cmd.ErrOrStderr(), sess, deepestCommandName(cmd), format, time.Now())
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(
		&opts.outputFormat,
		"output",
		string(output.FormatTable),
		"Output format: table, json, csv",
	)
	cmd.PersistentFlags().StringVar(
		&opts.configDir,
		"config-dir",
		"",
		"Override the config directory",
	)
	cmd.PersistentFlags().StringVar(
		&opts.sessionFile,
		"session-file",
		"",
		"Override the session file path",
	)

	cmd.AddCommand(
		newVersionCmd(opts),
		newDoctorCmd(opts),
		newConfigCmd(opts),
		newAuthCmd(opts),
		newAccountCmd(opts),
		newPortfolioCmd(opts),
		newOrdersCmd(opts),
		newTransactionsCmd(opts),
		newWatchlistCmd(opts),
		newQuoteCmd(opts),
		newOrderCmd(opts),
		newExportCmd(opts),
		newPushCmd(opts),
	)

	return cmd
}

// resolveSessionFile mirrors the resolution done in newAppContext but without
// requiring the full app context — PersistentPreRunE runs before subcommands
// have built theirs.
func resolveSessionFile(opts *rootOptions) string {
	if opts.sessionFile != "" {
		return opts.sessionFile
	}
	if opts.configDir != "" {
		return filepath.Join(opts.configDir, "session.json")
	}
	paths, err := config.DefaultPaths()
	if err != nil {
		return ""
	}
	return paths.SessionFile
}

func deepestCommandName(cmd *cobra.Command) string {
	if cmd == nil {
		return ""
	}
	return cmd.Name()
}

var expiryWarningSkipCommands = map[string]struct{}{
	"extend":                  {},
	"login":                   {},
	"logout":                  {},
	"status":                  {},
	"import-playwright-state": {},
	"version":                 {},
	"help":                    {},
}

func writeExpiryWarningIfNeeded(w io.Writer, sess *session.Session, cmdName string, format output.Format, now time.Time) {
	if sess == nil || sess.ServerExpiresAt == nil {
		return
	}
	if format == output.FormatJSON {
		return
	}
	if _, skip := expiryWarningSkipCommands[cmdName]; skip {
		return
	}
	remaining := sess.ServerExpiresAt.Sub(now)
	if remaining <= 0 || remaining >= 24*time.Hour {
		return
	}
	fmt.Fprintf(w, "⚠ session expires in ~%s; run `tossctl auth extend` to renew\n", humanizeDuration(remaining))
}

func humanizeDuration(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}
	hours := int(d.Hours())
	if hours >= 1 {
		minutes := int(d.Minutes()) % 60
		if minutes == 0 {
			return fmt.Sprintf("%dh", hours)
		}
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	minutes := int(d.Minutes())
	if minutes >= 1 {
		return fmt.Sprintf("%dm", minutes)
	}
	return fmt.Sprintf("%ds", int(d.Seconds()))
}

func newAppContext(opts *rootOptions) (*appContext, error) {
	format, err := output.ParseFormat(opts.outputFormat)
	if err != nil {
		return nil, err
	}

	paths, err := config.DefaultPaths()
	if err != nil {
		return nil, err
	}

	if opts.configDir != "" {
		paths.ConfigDir = opts.configDir
		paths.ConfigFile = filepath.Join(opts.configDir, "config.json")
		paths.SessionFile = filepath.Join(opts.configDir, "session.json")
		paths.PermissionFile = filepath.Join(opts.configDir, "trading-permission.json")
		paths.LineageFile = filepath.Join(opts.configDir, "trading-lineage.json")
	}

	if opts.sessionFile != "" {
		paths.SessionFile = opts.sessionFile
	}

	store := session.NewFileStore(paths.SessionFile)
	sess, err := store.Load(context.Background())
	if err != nil && !errors.Is(err, session.ErrNoSession) {
		return nil, err
	}

	loginConfig := auth.DefaultLoginConfig(paths.CacheDir)
	configService := config.NewService(paths.ConfigFile)
	cfg, err := configService.Load(context.Background())
	if err != nil {
		return nil, err
	}
	client := tossclient.New(tossclient.Config{
		Session:       sess,
		TradingPolicy: cfg.Trading,
	})
	permissionService := permissions.NewService(paths.PermissionFile)

	return &appContext{
		format:        format,
		paths:         paths,
		config:        cfg,
		configService: configService,
		loginConfig:   loginConfig,
		authService: auth.NewService(store, paths.SessionFile, auth.Options{
			LoginConfig:     loginConfig,
			Validator:       client,
			ExtensionRunner: client,
		}),
		client:            client,
		session:           sess,
		permissionService: permissionService,
		lineageService:    orderlineage.NewService(paths.LineageFile),
		tradingService:    trading.NewService(permissionService, cfg.Trading, client),
	}, nil
}
