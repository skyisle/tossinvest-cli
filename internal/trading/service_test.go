package trading

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/orderintent"
	"github.com/junghoonkye/tossinvest-cli/internal/permissions"
)

type brokerStub struct {
	placeCalled  bool
	cancelCalled bool
	amendCalled  bool
	stillPending bool
	lastOrderID  string
	pendingSequence []bool
	pendingChecks   int
}

func (b *brokerStub) PlacePendingOrder(_ context.Context, intent orderintent.PlaceIntent) error {
	b.placeCalled = true
	b.lastOrderID = intent.Symbol
	return nil
}

func (b *brokerStub) GetOrderAvailableActions(_ context.Context, orderID string) (map[string]any, error) {
	b.lastOrderID = orderID
	return map[string]any{"cancelSupported": true}, nil
}

func (b *brokerStub) CancelPendingOrder(_ context.Context, orderID string) error {
	b.cancelCalled = true
	b.lastOrderID = orderID
	return nil
}

func (b *brokerStub) AmendPendingOrder(_ context.Context, intent orderintent.AmendIntent) error {
	b.amendCalled = true
	b.lastOrderID = intent.OrderID
	return nil
}

func (b *brokerStub) HasPendingOrder(context.Context, string) (bool, error) {
	b.pendingChecks++
	if len(b.pendingSequence) > 0 {
		stillPending := b.pendingSequence[0]
		if len(b.pendingSequence) > 1 {
			b.pendingSequence = b.pendingSequence[1:]
		}
		return stillPending, nil
	}
	return b.stillPending, nil
}

func TestPlaceRequiresExecutionFlagsAndGrant(t *testing.T) {
	dir := t.TempDir()
	permissionService := permissions.NewService(filepath.Join(dir, "permission.json"))
	if _, err := permissionService.Grant(context.Background(), 5*time.Minute); err != nil {
		t.Fatalf("Grant returned error: %v", err)
	}

	service := NewService(permissionService, nil)
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

	if err := service.Place(context.Background(), intent, ExecuteOptions{}); !errors.Is(err, ErrExecuteRequired) {
		t.Fatalf("expected ErrExecuteRequired, got %v", err)
	}
	if err := service.Place(context.Background(), intent, ExecuteOptions{Execute: true}); !errors.Is(err, ErrDangerousFlagRequired) {
		t.Fatalf("expected ErrDangerousFlagRequired, got %v", err)
	}
	if err := service.Place(context.Background(), intent, ExecuteOptions{
		Execute:                    true,
		DangerouslySkipPermissions: true,
		Confirm:                    "badtoken",
	}); !errors.Is(err, ErrConfirmMismatch) {
		t.Fatalf("expected ErrConfirmMismatch, got %v", err)
	}
	if err := service.Place(context.Background(), intent, ExecuteOptions{
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
	service := NewService(permissionService, broker)
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

	if err := service.Place(context.Background(), intent, ExecuteOptions{
		Execute:                    true,
		DangerouslySkipPermissions: true,
		Confirm:                    service.PreviewPlace(intent).ConfirmToken,
	}); err != nil {
		t.Fatalf("Place returned error: %v", err)
	}
	if !broker.placeCalled {
		t.Fatal("expected broker place to be called")
	}
}

func TestCancelExecutesBrokerAndReconciles(t *testing.T) {
	dir := t.TempDir()
	permissionService := permissions.NewService(filepath.Join(dir, "permission.json"))
	if _, err := permissionService.Grant(context.Background(), 5*time.Minute); err != nil {
		t.Fatalf("Grant returned error: %v", err)
	}

	broker := &brokerStub{}
	service := NewService(permissionService, broker)

	intent, err := orderintent.NormalizeCancel("5", "TSLL")
	if err != nil {
		t.Fatalf("NormalizeCancel returned error: %v", err)
	}

	if err := service.Cancel(context.Background(), intent, ExecuteOptions{
		Execute:                    true,
		DangerouslySkipPermissions: true,
		Confirm:                    service.PreviewCancel(intent).ConfirmToken,
	}); err != nil {
		t.Fatalf("Cancel returned error: %v", err)
	}
	if !broker.cancelCalled {
		t.Fatal("expected broker cancel to be called")
	}
	if broker.lastOrderID != "5" {
		t.Fatalf("expected order id 5, got %q", broker.lastOrderID)
	}
}

func TestCancelWaitsForPendingOrderToDisappear(t *testing.T) {
	dir := t.TempDir()
	permissionService := permissions.NewService(filepath.Join(dir, "permission.json"))
	if _, err := permissionService.Grant(context.Background(), 5*time.Minute); err != nil {
		t.Fatalf("Grant returned error: %v", err)
	}

	previousAttempts := cancelReconcileAttempts
	previousInterval := cancelReconcileInterval
	cancelReconcileAttempts = 4
	cancelReconcileInterval = time.Millisecond
	defer func() {
		cancelReconcileAttempts = previousAttempts
		cancelReconcileInterval = previousInterval
	}()

	broker := &brokerStub{pendingSequence: []bool{true, true, false}}
	service := NewService(permissionService, broker)
	intent, err := orderintent.NormalizeCancel("5", "TSLL")
	if err != nil {
		t.Fatalf("NormalizeCancel returned error: %v", err)
	}

	err = service.Cancel(context.Background(), intent, ExecuteOptions{
		Execute:                    true,
		DangerouslySkipPermissions: true,
		Confirm:                    service.PreviewCancel(intent).ConfirmToken,
	})
	if err != nil {
		t.Fatalf("Cancel returned error: %v", err)
	}
	if broker.pendingChecks != 3 {
		t.Fatalf("expected 3 pending checks, got %d", broker.pendingChecks)
	}
}

func TestCancelFailsWhenOrderStillPending(t *testing.T) {
	dir := t.TempDir()
	permissionService := permissions.NewService(filepath.Join(dir, "permission.json"))
	if _, err := permissionService.Grant(context.Background(), 5*time.Minute); err != nil {
		t.Fatalf("Grant returned error: %v", err)
	}

	broker := &brokerStub{stillPending: true}
	service := NewService(permissionService, broker)
	previousAttempts := cancelReconcileAttempts
	previousInterval := cancelReconcileInterval
	cancelReconcileAttempts = 3
	cancelReconcileInterval = time.Millisecond
	defer func() {
		cancelReconcileAttempts = previousAttempts
		cancelReconcileInterval = previousInterval
	}()
	intent, err := orderintent.NormalizeCancel("5", "TSLL")
	if err != nil {
		t.Fatalf("NormalizeCancel returned error: %v", err)
	}

	err = service.Cancel(context.Background(), intent, ExecuteOptions{
		Execute:                    true,
		DangerouslySkipPermissions: true,
		Confirm:                    service.PreviewCancel(intent).ConfirmToken,
	})
	if !errors.Is(err, ErrCancelStillPending) {
		t.Fatalf("expected ErrCancelStillPending, got %v", err)
	}
	if broker.pendingChecks != 3 {
		t.Fatalf("expected 3 pending checks, got %d", broker.pendingChecks)
	}
}

func TestAmendCallsBrokerAfterGate(t *testing.T) {
	dir := t.TempDir()
	permissionService := permissions.NewService(filepath.Join(dir, "permission.json"))
	if _, err := permissionService.Grant(context.Background(), 5*time.Minute); err != nil {
		t.Fatalf("Grant returned error: %v", err)
	}

	broker := &brokerStub{}
	service := NewService(permissionService, broker)
	price := 700.0
	intent, err := orderintent.NormalizeAmend("13", nil, &price)
	if err != nil {
		t.Fatalf("NormalizeAmend returned error: %v", err)
	}

	err = service.Amend(context.Background(), intent, ExecuteOptions{
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
}
