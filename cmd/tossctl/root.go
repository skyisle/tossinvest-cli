package main

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/junghoonkye/tossinvest-cli/internal/auth"
	tossclient "github.com/junghoonkye/tossinvest-cli/internal/client"
	"github.com/junghoonkye/tossinvest-cli/internal/config"
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
	permissionService *permissions.Service
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
			_, err := output.ParseFormat(opts.outputFormat)
			return err
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
		newWatchlistCmd(opts),
		newQuoteCmd(opts),
		newOrderCmd(opts),
		newExportCmd(),
	)

	return cmd
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
	client := tossclient.New(tossclient.Config{Session: sess})
	configService := config.NewService(paths.ConfigFile)
	cfg, err := configService.Load(context.Background())
	if err != nil {
		return nil, err
	}
	permissionService := permissions.NewService(paths.PermissionFile)

	return &appContext{
		format:        format,
		paths:         paths,
		config:        cfg,
		configService: configService,
		loginConfig:   loginConfig,
		authService: auth.NewService(store, paths.SessionFile, auth.Options{
			LoginConfig: loginConfig,
			Validator:   client,
		}),
		client:            client,
		permissionService: permissionService,
		tradingService:    trading.NewService(permissionService, cfg.Trading, client),
	}, nil
}
