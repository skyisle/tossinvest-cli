package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestStatusFallsBackToDefaultWhenConfigIsMissing(t *testing.T) {
	service := NewService(filepath.Join(t.TempDir(), "config.json"))

	status, err := service.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if status.Exists {
		t.Fatal("expected config to be absent")
	}
	if status.SchemaVersion != SchemaVersion {
		t.Fatalf("expected schema version %d, got %d", SchemaVersion, status.SchemaVersion)
	}
	if status.Trading.Place {
		t.Fatal("expected place to be disabled by default")
	}
	if status.Trading.AllowLiveOrderActions {
		t.Fatal("expected live order actions to be disabled by default")
	}
}

func TestInitCreatesDefaultConfig(t *testing.T) {
	service := NewService(filepath.Join(t.TempDir(), "config.json"))

	result, err := service.Init(context.Background())
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	if !result.Created {
		t.Fatal("expected config file to be created")
	}
	if !result.Status.Exists {
		t.Fatal("expected config file to exist after init")
	}
	if result.Status.Schema != DefaultSchemaURL {
		t.Fatalf("expected schema url %q, got %q", DefaultSchemaURL, result.Status.Schema)
	}
	if result.Status.Trading.AllowLiveOrderActions {
		t.Fatal("expected live order actions to be disabled by default")
	}
}

func TestLoadTranslatesLegacyAllowDangerousExecute(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	data := []byte(`{
  "$schema": "https://raw.githubusercontent.com/JungHoonGhae/tossinvest-cli/main/schemas/config.schema.json",
  "schema_version": 1,
  "trading": {
    "grant": true,
    "place": true,
    "cancel": false,
    "amend": false,
    "allow_dangerous_execute": true
  }
}`)
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	service := NewService(configPath)

	status, err := service.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if status.SchemaVersion != SchemaVersion {
		t.Fatalf("expected effective schema version %d, got %d", SchemaVersion, status.SchemaVersion)
	}
	if status.SourceSchemaVersion != 1 {
		t.Fatalf("expected source schema version 1, got %d", status.SourceSchemaVersion)
	}
	if !status.Trading.AllowLiveOrderActions {
		t.Fatal("expected legacy allow_dangerous_execute to translate into allow_live_order_actions")
	}
	if len(status.LegacyFields) != 1 || status.LegacyFields[0] != "trading.allow_dangerous_execute" {
		t.Fatalf("unexpected legacy fields: %#v", status.LegacyFields)
	}
}

func TestLoadDefaultsSellToFalseWhenFieldMissing(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	data := []byte(`{
  "schema_version": 2,
  "trading": {
    "grant": true,
    "place": true,
    "cancel": false,
    "amend": false,
    "allow_live_order_actions": true,
    "dangerous_automation": {
      "complete_trade_auth": false,
      "accept_product_ack": false,
      "accept_fx_consent": false
    }
  }
}`)
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	service := NewService(configPath)
	cfg, err := service.Load(context.Background())
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Trading.Sell {
		t.Fatal("expected Sell to default to false when field is missing from config")
	}
	if !cfg.Trading.Place {
		t.Fatal("expected Place to be true")
	}
}

func TestLoadParsesSellTrue(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	data := []byte(`{
  "schema_version": 2,
  "trading": {
    "grant": true,
    "place": true,
    "sell": true,
    "cancel": false,
    "amend": false,
    "allow_live_order_actions": true,
    "dangerous_automation": {
      "complete_trade_auth": false,
      "accept_product_ack": false,
      "accept_fx_consent": false
    }
  }
}`)
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	service := NewService(configPath)
	cfg, err := service.Load(context.Background())
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if !cfg.Trading.Sell {
		t.Fatal("expected Sell to be true")
	}
}

func TestEnabledActionsIncludesSell(t *testing.T) {
	trading := Trading{
		Grant: true,
		Place: true,
		Sell:  true,
	}
	enabled := trading.EnabledActions()
	found := false
	for _, action := range enabled {
		if action == "sell" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected EnabledActions to include 'sell', got %v", enabled)
	}
}

func TestEnabledActionsExcludesSellWhenFalse(t *testing.T) {
	trading := Trading{
		Grant: true,
		Place: true,
		Sell:  false,
	}
	enabled := trading.EnabledActions()
	for _, action := range enabled {
		if action == "sell" {
			t.Fatalf("expected EnabledActions to exclude 'sell' when false, got %v", enabled)
		}
	}
}

func TestLoadDefaultsKRToFalseWhenFieldMissing(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	data := []byte(`{
  "schema_version": 2,
  "trading": {
    "grant": true,
    "place": true,
    "sell": true,
    "cancel": false,
    "amend": false,
    "allow_live_order_actions": true,
    "dangerous_automation": {
      "complete_trade_auth": false,
      "accept_product_ack": false,
      "accept_fx_consent": false
    }
  }
}`)
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	service := NewService(configPath)
	cfg, err := service.Load(context.Background())
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Trading.KR {
		t.Fatal("expected KR to default to false when field is missing from config")
	}
}

func TestLoadParsesKRTrue(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	data := []byte(`{
  "schema_version": 2,
  "trading": {
    "grant": true,
    "place": true,
    "sell": true,
    "kr": true,
    "cancel": false,
    "amend": false,
    "allow_live_order_actions": true,
    "dangerous_automation": {
      "complete_trade_auth": false,
      "accept_product_ack": false,
      "accept_fx_consent": false
    }
  }
}`)
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	service := NewService(configPath)
	cfg, err := service.Load(context.Background())
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if !cfg.Trading.KR {
		t.Fatal("expected KR to be true")
	}
}

func TestEnabledActionsIncludesKR(t *testing.T) {
	trading := Trading{Place: true, KR: true}
	enabled := trading.EnabledActions()
	found := false
	for _, action := range enabled {
		if action == "kr" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected EnabledActions to include 'kr', got %v", enabled)
	}
}

func TestInitCreatesDangerousAutomationDefaults(t *testing.T) {
	service := NewService(filepath.Join(t.TempDir(), "config.json"))

	result, err := service.Init(context.Background())
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	if result.Status.Trading.DangerousAutomation.CompleteTradeAuth {
		t.Fatal("expected complete_trade_auth to be disabled by default")
	}
	if result.Status.Trading.DangerousAutomation.AcceptProductAck {
		t.Fatal("expected accept_product_ack to be disabled by default")
	}
	if result.Status.Trading.DangerousAutomation.AcceptFXConsent {
		t.Fatal("expected accept_fx_consent to be disabled by default")
	}
}
