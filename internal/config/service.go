package config

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const (
	SchemaVersion    = 2
	DefaultSchemaURL = "https://raw.githubusercontent.com/JungHoonGhae/tossinvest-cli/main/schemas/config.schema.json"
)

type DangerousAutomation struct {
	AcceptFXConsent bool `json:"accept_fx_consent"`
}

type Trading struct {
	Place                 bool                `json:"place"`
	Sell                  bool                `json:"sell"`
	KR                    bool                `json:"kr"`
	Fractional            bool                `json:"fractional"`
	Cancel                bool                `json:"cancel"`
	Amend                 bool                `json:"amend"`
	AllowLiveOrderActions bool                `json:"allow_live_order_actions"`
	DangerousAutomation   DangerousAutomation `json:"dangerous_automation"`
}

func (t Trading) EnabledActions() []string {
	enabled := []string{}
	if t.Place {
		enabled = append(enabled, "place")
	}
	if t.Sell {
		enabled = append(enabled, "sell")
	}
	if t.KR {
		enabled = append(enabled, "kr")
	}
	if t.Fractional {
		enabled = append(enabled, "fractional")
	}
	if t.Cancel {
		enabled = append(enabled, "cancel")
	}
	if t.Amend {
		enabled = append(enabled, "amend")
	}
	return enabled
}

func (d DangerousAutomation) EnabledActions() []string {
	enabled := []string{}
	if d.AcceptFXConsent {
		enabled = append(enabled, "accept_fx_consent")
	}
	return enabled
}

// AnyMutationEnabled reports whether any order-mutation toggle is on.
// Used to decide whether commands like `order permissions grant` are useful
// (vs. being a no-op because no action gate is open).
func (t Trading) AnyMutationEnabled() bool {
	return t.Place || t.Cancel || t.Amend
}

type UpdateCheck struct {
	Enabled bool `json:"enabled"`
}

type File struct {
	Schema        string      `json:"$schema,omitempty"`
	SchemaVersion int         `json:"schema_version"`
	Trading       Trading     `json:"trading"`
	UpdateCheck   UpdateCheck `json:"update_check"`
}

type Status struct {
	ConfigFile          string      `json:"config_file"`
	Exists              bool        `json:"exists"`
	Schema              string      `json:"$schema,omitempty"`
	SchemaVersion       int         `json:"schema_version"`
	SourceSchemaVersion int         `json:"source_schema_version,omitempty"`
	LegacyFields        []string    `json:"legacy_fields,omitempty"`
	Trading             Trading     `json:"trading"`
	UpdateCheck         UpdateCheck `json:"update_check"`
}

type InitResult struct {
	Status  Status `json:"status"`
	Created bool   `json:"created"`
}

type Service struct {
	path string
}

type legacyMetadata struct {
	SourceSchemaVersion int
	LegacyFields        []string
}

type rawTrading struct {
	Grant                 *bool                   `json:"grant"`
	Place                 bool                    `json:"place"`
	Sell                  bool                    `json:"sell"`
	KR                    bool                    `json:"kr"`
	Fractional            bool                    `json:"fractional"`
	Cancel                bool                    `json:"cancel"`
	Amend                 bool                    `json:"amend"`
	AllowLiveOrderActions *bool                   `json:"allow_live_order_actions"`
	AllowDangerousExecute *bool                   `json:"allow_dangerous_execute"`
	DangerousAutomation   *rawDangerousAutomation `json:"dangerous_automation"`
}

type rawDangerousAutomation struct {
	CompleteTradeAuth *bool `json:"complete_trade_auth"`
	AcceptProductAck  *bool `json:"accept_product_ack"`
	AcceptFXConsent   bool  `json:"accept_fx_consent"`
}

type rawUpdateCheck struct {
	Enabled *bool `json:"enabled"`
}

type rawFile struct {
	Schema        string         `json:"$schema,omitempty"`
	SchemaVersion int            `json:"schema_version"`
	Trading       rawTrading     `json:"trading"`
	UpdateCheck   rawUpdateCheck `json:"update_check"`
}

func NewService(path string) *Service {
	return &Service{path: path}
}

func DefaultFile() File {
	return File{
		Schema:        DefaultSchemaURL,
		SchemaVersion: SchemaVersion,
		Trading:       Trading{},
		UpdateCheck:   UpdateCheck{Enabled: true},
	}
}

func (s *Service) Load(context.Context) (File, error) {
	cfg, _, _, err := s.load()
	return cfg, err
}

func (s *Service) Status(context.Context) (Status, error) {
	cfg, exists, meta, err := s.load()
	if err != nil {
		return Status{}, err
	}
	return Status{
		ConfigFile:          s.path,
		Exists:              exists,
		Schema:              cfg.Schema,
		SchemaVersion:       cfg.SchemaVersion,
		SourceSchemaVersion: meta.SourceSchemaVersion,
		LegacyFields:        meta.LegacyFields,
		Trading:             cfg.Trading,
		UpdateCheck:         cfg.UpdateCheck,
	}, nil
}

func (s *Service) Init(context.Context) (InitResult, error) {
	if _, err := os.Stat(s.path); err == nil {
		status, err := s.Status(context.Background())
		if err != nil {
			return InitResult{}, err
		}
		return InitResult{Status: status, Created: false}, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return InitResult{}, err
	}

	cfg := DefaultFile()
	if err := s.save(cfg); err != nil {
		return InitResult{}, err
	}
	status, err := s.Status(context.Background())
	if err != nil {
		return InitResult{}, err
	}
	return InitResult{Status: status, Created: true}, nil
}

func (s *Service) load() (File, bool, legacyMetadata, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultFile(), false, legacyMetadata{}, nil
		}
		return File{}, false, legacyMetadata{}, err
	}

	var raw rawFile
	if err := json.Unmarshal(data, &raw); err != nil {
		return File{}, true, legacyMetadata{}, err
	}
	cfg := DefaultFile()
	meta := legacyMetadata{}

	if raw.Schema != "" {
		cfg.Schema = raw.Schema
	}
	sourceSchemaVersion := raw.SchemaVersion
	if sourceSchemaVersion == 0 {
		sourceSchemaVersion = SchemaVersion
	}
	meta.SourceSchemaVersion = sourceSchemaVersion

	cfg.Trading.Place = raw.Trading.Place
	cfg.Trading.Sell = raw.Trading.Sell
	cfg.Trading.KR = raw.Trading.KR
	cfg.Trading.Fractional = raw.Trading.Fractional
	cfg.Trading.Cancel = raw.Trading.Cancel
	cfg.Trading.Amend = raw.Trading.Amend

	// trading.grant was removed in v0.4.3 — it gated nothing that the other
	// per-action toggles + allow_live_order_actions didn't already gate. We
	// still parse it so an old config with `grant` present doesn't fail to
	// load, and surface it in LegacyFields so the doctor can flag it.
	if raw.Trading.Grant != nil {
		meta.LegacyFields = append(meta.LegacyFields, "trading.grant")
	}

	switch {
	case raw.Trading.AllowLiveOrderActions != nil:
		cfg.Trading.AllowLiveOrderActions = *raw.Trading.AllowLiveOrderActions
	case raw.Trading.AllowDangerousExecute != nil:
		cfg.Trading.AllowLiveOrderActions = *raw.Trading.AllowDangerousExecute
		meta.LegacyFields = append(meta.LegacyFields, "trading.allow_dangerous_execute")
	}

	if raw.Trading.DangerousAutomation != nil {
		cfg.Trading.DangerousAutomation.AcceptFXConsent = raw.Trading.DangerousAutomation.AcceptFXConsent
		// complete_trade_auth / accept_product_ack were removed in v0.4.3
		// — never wired to any behavior. Legacy key detection only.
		if raw.Trading.DangerousAutomation.CompleteTradeAuth != nil {
			meta.LegacyFields = append(meta.LegacyFields, "trading.dangerous_automation.complete_trade_auth")
		}
		if raw.Trading.DangerousAutomation.AcceptProductAck != nil {
			meta.LegacyFields = append(meta.LegacyFields, "trading.dangerous_automation.accept_product_ack")
		}
	}

	if cfg.Schema == "" {
		cfg.Schema = DefaultSchemaURL
	}
	cfg.SchemaVersion = SchemaVersion

	if raw.UpdateCheck.Enabled != nil {
		cfg.UpdateCheck.Enabled = *raw.UpdateCheck.Enabled
	}

	return cfg, true, meta, nil
}

func (s *Service) save(cfg File) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmpPath, s.path)
}
