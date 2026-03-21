package trading

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/config"
	"github.com/junghoonkye/tossinvest-cli/internal/orderintent"
	"github.com/junghoonkye/tossinvest-cli/internal/permissions"
)

type brokerStub struct {
	placeCalled  bool
	cancelCalled bool
	amendCalled  bool
	lastOrderID  string
	placeResult  MutationResult
	cancelResult MutationResult
	amendResult  MutationResult
}

func (b *brokerStub) PlacePendingOrder(_ context.Context, intent orderintent.PlaceIntent) (MutationResult, error) {
	b.placeCalled = true
	b.lastOrderID = intent.Symbol
	if b.placeResult.Kind == "" {
		b.placeResult = MutationResult{Kind: "place", Status: "accepted_pending", OrderID: intent.Symbol}
	}
	return b.placeResult, nil
}

func (b *brokerStub) GetOrderAvailableActions(_ context.Context, orderID string) (map[string]any, error) {
	b.lastOrderID = orderID
	return map[string]any{"cancelSupported": true}, nil
}

func (b *brokerStub) CancelPendingOrder(_ context.Context, intent orderintent.CancelIntent) (MutationResult, error) {
	b.cancelCalled = true
	b.lastOrderID = intent.OrderID
	if b.cancelResult.Kind == "" {
		b.cancelResult = MutationResult{Kind: "cancel", Status: "canceled", OrderID: intent.OrderID}
	}
	return b.cancelResult, nil
}

func (b *brokerStub) AmendPendingOrder(_ context.Context, intent orderintent.AmendIntent) (MutationResult, error) {
	b.amendCalled = true
	b.lastOrderID = intent.OrderID
	if b.amendResult.Kind == "" {
		b.amendResult = MutationResult{Kind: "amend", Status: "amended_pending", OrderID: intent.OrderID, CurrentOrderID: intent.OrderID}
	}
	return b.amendResult, nil
}

func TestPlaceRequiresExecutionFlagsAndGrant(t *testing.T) {
	dir := t.TempDir()
	permissionService := permissions.NewService(filepath.Join(dir, "permission.json"))
	if _, err := permissionService.Grant(context.Background(), 5*time.Minute); err != nil {
		t.Fatalf("Grant returned error: %v", err)
	}

	service := NewService(permissionService, config.Trading{
		Place:                 true,
		AllowLiveOrderActions: true,
	}, nil)
	intent, err := orderintent.NormalizePlace(orderintent.PlaceInput{
		Symbol:       "TSLL",
		Market:       "us",
		Side:         "buy",
		OrderType:    "limit",
		Quantity:     1,
		Price:        500,
		CurrencyMode: "KRW",
	})
	if err != nil {
		t.Fatalf("NormalizePlace returned error: %v", err)
	}

	if _, err := service.Place(context.Background(), intent, ExecuteOptions{}); !errors.Is(err, ErrExecuteRequired) {
		t.Fatalf("expected ErrExecuteRequired, got %v", err)
	}
	if _, err := service.Place(context.Background(), intent, ExecuteOptions{Execute: true}); !errors.Is(err, ErrDangerousFlagRequired) {
		t.Fatalf("expected ErrDangerousFlagRequired, got %v", err)
	}
	if _, err := service.Place(context.Background(), intent, ExecuteOptions{
		Execute:                    true,
		DangerouslySkipPermissions: true,
		Confirm:                    "badtoken",
	}); !errors.Is(err, ErrConfirmMismatch) {
		t.Fatalf("expected ErrConfirmMismatch, got %v", err)
	}
	if _, err := service.Place(context.Background(), intent, ExecuteOptions{
		Execute:                    true,
		DangerouslySkipPermissions: true,
		Confirm:                    service.PreviewPlace(intent).ConfirmToken,
	}); !errors.Is(err, ErrLiveMutationPending) {
		t.Fatalf("expected ErrLiveMutationPending, got %v", err)
	}
}

func TestPlaceCallsBrokerForSupportedIntent(t *testing.T) {
	dir := t.TempDir()
	permissionService := permissions.NewService(filepath.Join(dir, "permission.json"))
	if _, err := permissionService.Grant(context.Background(), 5*time.Minute); err != nil {
		t.Fatalf("Grant returned error: %v", err)
	}

	broker := &brokerStub{}
	service := NewService(permissionService, config.Trading{
		Place:                 true,
		AllowLiveOrderActions: true,
	}, broker)
	intent, err := orderintent.NormalizePlace(orderintent.PlaceInput{
		Symbol:       "TSLL",
		Market:       "us",
		Side:         "buy",
		OrderType:    "limit",
		Quantity:     1,
		Price:        500,
		CurrencyMode: "KRW",
	})
	if err != nil {
		t.Fatalf("NormalizePlace returned error: %v", err)
	}

	result, err := service.Place(context.Background(), intent, ExecuteOptions{
		Execute:                    true,
		DangerouslySkipPermissions: true,
		Confirm:                    service.PreviewPlace(intent).ConfirmToken,
	})
	if err != nil {
		t.Fatalf("Place returned error: %v", err)
	}
	if !broker.placeCalled {
		t.Fatal("expected broker place to be called")
	}
	if result.Status != "accepted_pending" {
		t.Fatalf("expected accepted_pending, got %q", result.Status)
	}
}

func TestCancelExecutesBrokerAndReconciles(t *testing.T) {
	dir := t.TempDir()
	permissionService := permissions.NewService(filepath.Join(dir, "permission.json"))
	if _, err := permissionService.Grant(context.Background(), 5*time.Minute); err != nil {
		t.Fatalf("Grant returned error: %v", err)
	}

	broker := &brokerStub{}
	service := NewService(permissionService, config.Trading{
		Cancel:                true,
		AllowLiveOrderActions: true,
	}, broker)

	intent, err := orderintent.NormalizeCancel("5", "TSLL")
	if err != nil {
		t.Fatalf("NormalizeCancel returned error: %v", err)
	}

	result, err := service.Cancel(context.Background(), intent, ExecuteOptions{
		Execute:                    true,
		DangerouslySkipPermissions: true,
		Confirm:                    service.PreviewCancel(intent).ConfirmToken,
	})
	if err != nil {
		t.Fatalf("Cancel returned error: %v", err)
	}
	if !broker.cancelCalled {
		t.Fatal("expected broker cancel to be called")
	}
	if broker.lastOrderID != "5" {
		t.Fatalf("expected order id 5, got %q", broker.lastOrderID)
	}
	if result.Status != "canceled" {
		t.Fatalf("expected canceled result, got %q", result.Status)
	}
}

func TestAmendCallsBrokerAfterGate(t *testing.T) {
	dir := t.TempDir()
	permissionService := permissions.NewService(filepath.Join(dir, "permission.json"))
	if _, err := permissionService.Grant(context.Background(), 5*time.Minute); err != nil {
		t.Fatalf("Grant returned error: %v", err)
	}

	broker := &brokerStub{}
	service := NewService(permissionService, config.Trading{
		Amend:                 true,
		AllowLiveOrderActions: true,
	}, broker)
	price := 700.0
	intent, err := orderintent.NormalizeAmend("13", nil, &price)
	if err != nil {
		t.Fatalf("NormalizeAmend returned error: %v", err)
	}

	result, err := service.Amend(context.Background(), intent, ExecuteOptions{
		Execute:                    true,
		DangerouslySkipPermissions: true,
		Confirm:                    service.PreviewAmend(intent).ConfirmToken,
	})
	if err != nil {
		t.Fatalf("Amend returned error: %v", err)
	}
	if !broker.amendCalled {
		t.Fatal("expected broker amend to be called")
	}
	if result.Status != "amended_pending" {
		t.Fatalf("expected amended_pending, got %q", result.Status)
	}
}

func TestPlaceFailsWhenActionDisabledInConfig(t *testing.T) {
	dir := t.TempDir()
	permissionService := permissions.NewService(filepath.Join(dir, "permission.json"))
	if _, err := permissionService.Grant(context.Background(), 5*time.Minute); err != nil {
		t.Fatalf("Grant returned error: %v", err)
	}

	service := NewService(permissionService, config.Trading{
		AllowLiveOrderActions: true,
	}, nil)
	intent, err := orderintent.NormalizePlace(orderintent.PlaceInput{
		Symbol:       "TSLL",
		Market:       "us",
		Side:         "buy",
		OrderType:    "limit",
		Quantity:     1,
		Price:        500,
		CurrencyMode: "KRW",
	})
	if err != nil {
		t.Fatalf("NormalizePlace returned error: %v", err)
	}

	_, err = service.Place(context.Background(), intent, ExecuteOptions{
		Execute:                    true,
		DangerouslySkipPermissions: true,
		Confirm:                    service.PreviewPlace(intent).ConfirmToken,
	})
	var disabled *DisabledActionError
	if !errors.As(err, &disabled) || disabled.Action != ActionPlace {
		t.Fatalf("expected place action to be disabled, got %v", err)
	}
}

func TestPlaceIntentSupportedAcceptsSell(t *testing.T) {
	intent, err := orderintent.NormalizePlace(orderintent.PlaceInput{
		Symbol:       "TSLL",
		Market:       "us",
		Side:         "sell",
		OrderType:    "limit",
		Quantity:     1,
		Price:        500,
		CurrencyMode: "KRW",
	})
	if err != nil {
		t.Fatalf("NormalizePlace returned error: %v", err)
	}
	if !placeIntentSupported(intent) {
		t.Fatal("expected placeIntentSupported to return true for sell intent")
	}
}

func TestSellPlaceFailsWhenSellDisabledInConfig(t *testing.T) {
	dir := t.TempDir()
	permissionService := permissions.NewService(filepath.Join(dir, "permission.json"))
	if _, err := permissionService.Grant(context.Background(), 5*time.Minute); err != nil {
		t.Fatalf("Grant returned error: %v", err)
	}

	broker := &brokerStub{}
	service := NewService(permissionService, config.Trading{
		Place:                 true,
		Sell:                  false,
		AllowLiveOrderActions: true,
	}, broker)
	intent, err := orderintent.NormalizePlace(orderintent.PlaceInput{
		Symbol:       "TSLL",
		Market:       "us",
		Side:         "sell",
		OrderType:    "limit",
		Quantity:     1,
		Price:        500,
		CurrencyMode: "KRW",
	})
	if err != nil {
		t.Fatalf("NormalizePlace returned error: %v", err)
	}

	_, err = service.Place(context.Background(), intent, ExecuteOptions{
		Execute:                    true,
		DangerouslySkipPermissions: true,
		Confirm:                    service.PreviewPlace(intent).ConfirmToken,
	})
	var disabled *DisabledActionError
	if !errors.As(err, &disabled) || disabled.Action != "sell" {
		t.Fatalf("expected sell action to be disabled, got %v", err)
	}
	if broker.placeCalled {
		t.Fatal("broker should not have been called when sell is disabled")
	}
}

func TestSellPlaceCallsBrokerWhenSellEnabled(t *testing.T) {
	dir := t.TempDir()
	permissionService := permissions.NewService(filepath.Join(dir, "permission.json"))
	if _, err := permissionService.Grant(context.Background(), 5*time.Minute); err != nil {
		t.Fatalf("Grant returned error: %v", err)
	}

	broker := &brokerStub{}
	service := NewService(permissionService, config.Trading{
		Place:                 true,
		Sell:                  true,
		AllowLiveOrderActions: true,
	}, broker)
	intent, err := orderintent.NormalizePlace(orderintent.PlaceInput{
		Symbol:       "TSLL",
		Market:       "us",
		Side:         "sell",
		OrderType:    "limit",
		Quantity:     1,
		Price:        500,
		CurrencyMode: "KRW",
	})
	if err != nil {
		t.Fatalf("NormalizePlace returned error: %v", err)
	}

	result, err := service.Place(context.Background(), intent, ExecuteOptions{
		Execute:                    true,
		DangerouslySkipPermissions: true,
		Confirm:                    service.PreviewPlace(intent).ConfirmToken,
	})
	if err != nil {
		t.Fatalf("Place returned error: %v", err)
	}
	if !broker.placeCalled {
		t.Fatal("expected broker place to be called for sell")
	}
	if result.Status != "accepted_pending" {
		t.Fatalf("expected accepted_pending, got %q", result.Status)
	}
}

func TestPreviewPlaceSellDisabled(t *testing.T) {
	dir := t.TempDir()
	permissionService := permissions.NewService(filepath.Join(dir, "permission.json"))

	service := NewService(permissionService, config.Trading{
		Place:                 true,
		Sell:                  false,
		AllowLiveOrderActions: true,
	}, nil)
	intent, err := orderintent.NormalizePlace(orderintent.PlaceInput{
		Symbol:       "TSLL",
		Market:       "us",
		Side:         "sell",
		OrderType:    "limit",
		Quantity:     1,
		Price:        500,
		CurrencyMode: "KRW",
	})
	if err != nil {
		t.Fatalf("NormalizePlace returned error: %v", err)
	}

	preview := service.PreviewPlace(intent)
	if preview.MutationReady {
		t.Fatal("expected MutationReady to be false when sell is disabled")
	}
	found := false
	for _, w := range preview.Warnings {
		if w == "Config currently disables `order place --side sell`." {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected sell-disabled warning in preview, got %v", preview.Warnings)
	}
}

func TestPreviewPlaceSellEnabled(t *testing.T) {
	dir := t.TempDir()
	permissionService := permissions.NewService(filepath.Join(dir, "permission.json"))

	service := NewService(permissionService, config.Trading{
		Place:                 true,
		Sell:                  true,
		AllowLiveOrderActions: true,
	}, nil)
	intent, err := orderintent.NormalizePlace(orderintent.PlaceInput{
		Symbol:       "TSLL",
		Market:       "us",
		Side:         "sell",
		OrderType:    "limit",
		Quantity:     1,
		Price:        500,
		CurrencyMode: "KRW",
	})
	if err != nil {
		t.Fatalf("NormalizePlace returned error: %v", err)
	}

	preview := service.PreviewPlace(intent)
	if !preview.LiveReady {
		t.Fatal("expected LiveReady to be true for sell")
	}
	if !preview.MutationReady {
		t.Fatal("expected MutationReady to be true when sell is enabled")
	}
}

func TestPlaceIntentSupportedAcceptsFractionalMarketUS(t *testing.T) {
	intent := orderintent.PlaceIntent{Symbol: "TSLL", Market: "us", Side: "buy", OrderType: "market", Amount: 18000, CurrencyMode: "KRW", Fractional: true}
	if !placeIntentSupported(intent) {
		t.Fatal("expected true for US fractional market order")
	}
}

func TestPlaceIntentSupportedRejectsFractionalLimit(t *testing.T) {
	intent := orderintent.PlaceIntent{Symbol: "TSLL", Market: "us", Side: "buy", OrderType: "limit", Price: 500, CurrencyMode: "KRW", Fractional: true}
	if placeIntentSupported(intent) {
		t.Fatal("expected false for fractional limit order")
	}
}

func TestPlaceIntentSupportedRejectsFractionalKR(t *testing.T) {
	intent := orderintent.PlaceIntent{Symbol: "290080", Market: "kr", Side: "buy", OrderType: "market", Amount: 8000, CurrencyMode: "KRW", Fractional: true}
	if placeIntentSupported(intent) {
		t.Fatal("expected false for KR fractional order")
	}
}

func TestFractionalPlaceFailsWhenDisabled(t *testing.T) {
	dir := t.TempDir()
	ps := permissions.NewService(filepath.Join(dir, "p.json"))
	ps.Grant(context.Background(), 5*time.Minute)
	svc := NewService(ps, config.Trading{Place: true, Fractional: false, AllowLiveOrderActions: true}, nil)
	intent := orderintent.PlaceIntent{Symbol: "TSLL", Market: "us", Side: "buy", OrderType: "market", Amount: 18000, CurrencyMode: "KRW", Fractional: true}
	_, err := svc.Place(context.Background(), intent, ExecuteOptions{Execute: true, DangerouslySkipPermissions: true, Confirm: svc.PreviewPlace(intent).ConfirmToken})
	var disabled *DisabledActionError
	if !errors.As(err, &disabled) || disabled.Action != "fractional" {
		t.Fatalf("expected fractional disabled, got %v", err)
	}
}

func TestFractionalPlaceCallsBroker(t *testing.T) {
	dir := t.TempDir()
	ps := permissions.NewService(filepath.Join(dir, "p.json"))
	ps.Grant(context.Background(), 5*time.Minute)
	broker := &brokerStub{}
	svc := NewService(ps, config.Trading{Place: true, Fractional: true, AllowLiveOrderActions: true}, broker)
	intent := orderintent.PlaceIntent{Symbol: "TSLL", Market: "us", Side: "buy", OrderType: "market", Amount: 18000, CurrencyMode: "KRW", Fractional: true}
	result, err := svc.Place(context.Background(), intent, ExecuteOptions{Execute: true, DangerouslySkipPermissions: true, Confirm: svc.PreviewPlace(intent).ConfirmToken})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !broker.placeCalled {
		t.Fatal("expected broker called")
	}
	if result.Status != "accepted_pending" {
		t.Fatalf("expected accepted_pending, got %q", result.Status)
	}
}

func TestPreviewPlaceFractionalDisabled(t *testing.T) {
	dir := t.TempDir()
	ps := permissions.NewService(filepath.Join(dir, "p.json"))
	svc := NewService(ps, config.Trading{Place: true, Fractional: false, AllowLiveOrderActions: true}, nil)
	intent := orderintent.PlaceIntent{Symbol: "TSLL", Market: "us", Side: "buy", OrderType: "market", Amount: 18000, CurrencyMode: "KRW", Fractional: true}
	preview := svc.PreviewPlace(intent)
	if preview.MutationReady {
		t.Fatal("expected MutationReady false")
	}
}

func TestPlaceIntentSupportedAcceptsKR(t *testing.T) {
	intent, err := orderintent.NormalizePlace(orderintent.PlaceInput{
		Symbol:       "290080",
		Market:       "kr",
		Side:         "buy",
		OrderType:    "limit",
		Quantity:     1,
		Price:        8000,
		CurrencyMode: "KRW",
	})
	if err != nil {
		t.Fatalf("NormalizePlace returned error: %v", err)
	}
	if !placeIntentSupported(intent) {
		t.Fatal("expected placeIntentSupported to return true for kr intent")
	}
}

func TestKRPlaceFailsWhenKRDisabledInConfig(t *testing.T) {
	dir := t.TempDir()
	permissionService := permissions.NewService(filepath.Join(dir, "permission.json"))
	if _, err := permissionService.Grant(context.Background(), 5*time.Minute); err != nil {
		t.Fatalf("Grant returned error: %v", err)
	}

	broker := &brokerStub{}
	service := NewService(permissionService, config.Trading{
		Place:                 true,
		KR:                    false,
		AllowLiveOrderActions: true,
	}, broker)
	intent, err := orderintent.NormalizePlace(orderintent.PlaceInput{
		Symbol:       "290080",
		Market:       "kr",
		Side:         "buy",
		OrderType:    "limit",
		Quantity:     1,
		Price:        8000,
		CurrencyMode: "KRW",
	})
	if err != nil {
		t.Fatalf("NormalizePlace returned error: %v", err)
	}

	_, err = service.Place(context.Background(), intent, ExecuteOptions{
		Execute:                    true,
		DangerouslySkipPermissions: true,
		Confirm:                    service.PreviewPlace(intent).ConfirmToken,
	})
	var disabled *DisabledActionError
	if !errors.As(err, &disabled) || disabled.Action != "kr" {
		t.Fatalf("expected kr action to be disabled, got %v", err)
	}
	if broker.placeCalled {
		t.Fatal("broker should not have been called when kr is disabled")
	}
}

func TestKRPlaceCallsBrokerWhenKREnabled(t *testing.T) {
	dir := t.TempDir()
	permissionService := permissions.NewService(filepath.Join(dir, "permission.json"))
	if _, err := permissionService.Grant(context.Background(), 5*time.Minute); err != nil {
		t.Fatalf("Grant returned error: %v", err)
	}

	broker := &brokerStub{}
	service := NewService(permissionService, config.Trading{
		Place:                 true,
		KR:                    true,
		AllowLiveOrderActions: true,
	}, broker)
	intent, err := orderintent.NormalizePlace(orderintent.PlaceInput{
		Symbol:       "290080",
		Market:       "kr",
		Side:         "buy",
		OrderType:    "limit",
		Quantity:     1,
		Price:        8000,
		CurrencyMode: "KRW",
	})
	if err != nil {
		t.Fatalf("NormalizePlace returned error: %v", err)
	}

	result, err := service.Place(context.Background(), intent, ExecuteOptions{
		Execute:                    true,
		DangerouslySkipPermissions: true,
		Confirm:                    service.PreviewPlace(intent).ConfirmToken,
	})
	if err != nil {
		t.Fatalf("Place returned error: %v", err)
	}
	if !broker.placeCalled {
		t.Fatal("expected broker place to be called for kr")
	}
	if result.Status != "accepted_pending" {
		t.Fatalf("expected accepted_pending, got %q", result.Status)
	}
}

func TestPlacePolicyChecksBeforeGuard(t *testing.T) {
	dir := t.TempDir()
	permissionService := permissions.NewService(filepath.Join(dir, "permission.json"))
	// NO grant — guard() would fail with ErrExecuteRequired etc.
	// But policy check should catch it first.

	service := NewService(permissionService, config.Trading{
		Place:                 true,
		KR:                    false,
		AllowLiveOrderActions: true,
	}, nil)
	intent, err := orderintent.NormalizePlace(orderintent.PlaceInput{
		Symbol:       "290080",
		Market:       "kr",
		Side:         "buy",
		OrderType:    "limit",
		Quantity:     1,
		Price:        8000,
		CurrencyMode: "KRW",
	})
	if err != nil {
		t.Fatalf("NormalizePlace returned error: %v", err)
	}

	// Without --execute, guard would return ErrExecuteRequired.
	// But kr policy should be checked BEFORE guard, so we get DisabledActionError.
	_, err = service.Place(context.Background(), intent, ExecuteOptions{})
	var disabled *DisabledActionError
	if !errors.As(err, &disabled) || disabled.Action != "kr" {
		t.Fatalf("expected kr disabled error BEFORE guard, got %v", err)
	}
}

func TestPreviewPlaceKRDisabled(t *testing.T) {
	dir := t.TempDir()
	permissionService := permissions.NewService(filepath.Join(dir, "permission.json"))

	service := NewService(permissionService, config.Trading{
		Place:                 true,
		KR:                    false,
		AllowLiveOrderActions: true,
	}, nil)
	intent, err := orderintent.NormalizePlace(orderintent.PlaceInput{
		Symbol:       "290080",
		Market:       "kr",
		Side:         "buy",
		OrderType:    "limit",
		Quantity:     1,
		Price:        8000,
		CurrencyMode: "KRW",
	})
	if err != nil {
		t.Fatalf("NormalizePlace returned error: %v", err)
	}

	preview := service.PreviewPlace(intent)
	if preview.MutationReady {
		t.Fatal("expected MutationReady to be false when kr is disabled")
	}
	found := false
	for _, w := range preview.Warnings {
		if w == "Config currently disables `order place --market kr`." {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected kr-disabled warning in preview, got %v", preview.Warnings)
	}
}

func TestPreviewPlaceKREnabled(t *testing.T) {
	dir := t.TempDir()
	permissionService := permissions.NewService(filepath.Join(dir, "permission.json"))

	service := NewService(permissionService, config.Trading{
		Place:                 true,
		KR:                    true,
		AllowLiveOrderActions: true,
	}, nil)
	intent, err := orderintent.NormalizePlace(orderintent.PlaceInput{
		Symbol:       "290080",
		Market:       "kr",
		Side:         "buy",
		OrderType:    "limit",
		Quantity:     1,
		Price:        8000,
		CurrencyMode: "KRW",
	})
	if err != nil {
		t.Fatalf("NormalizePlace returned error: %v", err)
	}

	preview := service.PreviewPlace(intent)
	if !preview.LiveReady {
		t.Fatal("expected LiveReady to be true for kr")
	}
	if !preview.MutationReady {
		t.Fatal("expected MutationReady to be true when kr is enabled")
	}
}

func TestPlaceFailsWhenDangerousExecuteIsDisabledInConfig(t *testing.T) {
	dir := t.TempDir()
	permissionService := permissions.NewService(filepath.Join(dir, "permission.json"))
	if _, err := permissionService.Grant(context.Background(), 5*time.Minute); err != nil {
		t.Fatalf("Grant returned error: %v", err)
	}

	service := NewService(permissionService, config.Trading{
		Place: true,
	}, nil)
	intent, err := orderintent.NormalizePlace(orderintent.PlaceInput{
		Symbol:       "TSLL",
		Market:       "us",
		Side:         "buy",
		OrderType:    "limit",
		Quantity:     1,
		Price:        500,
		CurrencyMode: "KRW",
	})
	if err != nil {
		t.Fatalf("NormalizePlace returned error: %v", err)
	}

	_, err = service.Place(context.Background(), intent, ExecuteOptions{
		Execute:                    true,
		DangerouslySkipPermissions: true,
		Confirm:                    service.PreviewPlace(intent).ConfirmToken,
	})
	if !errors.Is(err, ErrDangerousExecuteDisabled) {
		t.Fatalf("expected ErrDangerousExecuteDisabled, got %v", err)
	}
}
