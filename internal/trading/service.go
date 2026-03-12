package trading

import (
	"context"
	"crypto/subtle"
	"fmt"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/config"
	"github.com/junghoonkye/tossinvest-cli/internal/orderintent"
	"github.com/junghoonkye/tossinvest-cli/internal/permissions"
)

type Broker interface {
	PlacePendingOrder(ctx context.Context, intent orderintent.PlaceIntent) error
	GetOrderAvailableActions(ctx context.Context, orderID string) (map[string]any, error)
	CancelPendingOrder(ctx context.Context, orderID string) error
	AmendPendingOrder(ctx context.Context, intent orderintent.AmendIntent) error
	HasPendingOrder(ctx context.Context, orderID string) (bool, error)
}

type Preview struct {
	Kind          string   `json:"kind"`
	Canonical     string   `json:"canonical"`
	ConfirmToken  string   `json:"confirm_token"`
	Warnings      []string `json:"warnings,omitempty"`
	LiveReady     bool     `json:"live_ready"`
	MutationReady bool     `json:"mutation_ready"`
}

type ExecuteOptions struct {
	Execute                    bool
	DangerouslySkipPermissions bool
	Confirm                    string
}

type Service struct {
	permissions *permissions.Service
	policy      config.Trading
	broker      Broker
}

var (
	cancelReconcileAttempts = 8
	cancelReconcileInterval = 250 * time.Millisecond
)

func NewService(permissionService *permissions.Service, policy config.Trading, broker Broker) *Service {
	return &Service{
		permissions: permissionService,
		policy:      policy,
		broker:      broker,
	}
}

func (s *Service) PreviewPlace(intent orderintent.PlaceIntent) Preview {
	canonical := orderintent.CanonicalPlace(intent)
	warnings := []string{
		"Live place currently supports only US buy limit orders in KRW, non-fractional mode.",
		"US orders may still require funding, FX consent, or product-risk acknowledgement before submission.",
	}
	liveReady := placeIntentSupported(intent)
	if !s.policy.Place {
		warnings = append(warnings, "Config currently disables `order place`.")
	}
	if !s.policy.AllowDangerousExecute {
		warnings = append(warnings, "Config currently disables dangerous live execution.")
	}
	return Preview{
		Kind:          "place",
		Canonical:     canonical,
		ConfirmToken:  orderintent.ConfirmToken(canonical),
		Warnings:      warnings,
		LiveReady:     liveReady,
		MutationReady: liveReady && s.policy.Place && s.policy.AllowDangerousExecute,
	}
}

func (s *Service) PreviewCancel(intent orderintent.CancelIntent) Preview {
	canonical := orderintent.CanonicalCancel(intent)
	warnings := []string{"Single-order cancel is wired for same-day pending orders and still reconciles through pending history."}
	if !s.policy.Cancel {
		warnings = append(warnings, "Config currently disables `order cancel`.")
	}
	if !s.policy.AllowDangerousExecute {
		warnings = append(warnings, "Config currently disables dangerous live execution.")
	}
	return Preview{
		Kind:          "cancel",
		Canonical:     canonical,
		ConfirmToken:  orderintent.ConfirmToken(canonical),
		Warnings:      warnings,
		LiveReady:     true,
		MutationReady: s.policy.Cancel && s.policy.AllowDangerousExecute,
	}
}

func (s *Service) PreviewAmend(intent orderintent.AmendIntent) Preview {
	canonical := orderintent.CanonicalAmend(intent)
	warnings := []string{
		"Amend reconciles against the surviving pending order record after mutation.",
		"Request bodies for amend are still under active discovery.",
	}
	if !s.policy.Amend {
		warnings = append(warnings, "Config currently disables `order amend`.")
	}
	if !s.policy.AllowDangerousExecute {
		warnings = append(warnings, "Config currently disables dangerous live execution.")
	}
	return Preview{
		Kind:          "amend",
		Canonical:     canonical,
		ConfirmToken:  orderintent.ConfirmToken(canonical),
		Warnings:      warnings,
		LiveReady:     false,
		MutationReady: false,
	}
}

func (s *Service) Place(ctx context.Context, intent orderintent.PlaceIntent, opts ExecuteOptions) error {
	if err := s.guard(ctx, ActionPlace, s.PreviewPlace(intent), opts); err != nil {
		return err
	}
	if !placeIntentSupported(intent) {
		return ErrPlaceUnsupported
	}
	if s.broker == nil {
		return ErrLiveMutationPending
	}
	return s.broker.PlacePendingOrder(ctx, intent)
}

func (s *Service) Cancel(ctx context.Context, intent orderintent.CancelIntent, opts ExecuteOptions) error {
	if err := s.guard(ctx, ActionCancel, s.PreviewCancel(intent), opts); err != nil {
		return err
	}
	if s.broker == nil {
		return ErrLiveMutationPending
	}

	if _, err := s.broker.GetOrderAvailableActions(ctx, intent.OrderID); err != nil {
		return err
	}
	if err := s.broker.CancelPendingOrder(ctx, intent.OrderID); err != nil {
		return err
	}

	return s.waitForCanceledOrder(ctx, intent.OrderID)
}

func (s *Service) Amend(ctx context.Context, intent orderintent.AmendIntent, opts ExecuteOptions) error {
	if err := s.guard(ctx, ActionAmend, s.PreviewAmend(intent), opts); err != nil {
		return err
	}
	if s.broker == nil {
		return ErrLiveMutationPending
	}
	return s.broker.AmendPendingOrder(ctx, intent)
}

func (s *Service) GrantEnabled() error {
	return s.requireActionEnabled(ActionGrant)
}

func (s *Service) guard(ctx context.Context, action Action, preview Preview, opts ExecuteOptions) error {
	if err := s.requireActionEnabled(action); err != nil {
		return err
	}
	if !opts.Execute {
		return fmt.Errorf("%w; rerun with --execute after reviewing `tossctl order preview`", ErrExecuteRequired)
	}
	if !s.policy.AllowDangerousExecute {
		return ErrDangerousExecuteDisabled
	}
	if !opts.DangerouslySkipPermissions {
		return fmt.Errorf("%w; explicit danger acknowledgement is required", ErrDangerousFlagRequired)
	}
	if err := s.permissions.Require(ctx); err != nil {
		return err
	}
	if subtle.ConstantTimeCompare([]byte(opts.Confirm), []byte(preview.ConfirmToken)) != 1 {
		return ErrConfirmMismatch
	}
	return nil
}

func (s *Service) requireActionEnabled(action Action) error {
	enabled := false
	switch action {
	case ActionGrant:
		enabled = s.policy.Grant
	case ActionPlace:
		enabled = s.policy.Place
	case ActionCancel:
		enabled = s.policy.Cancel
	case ActionAmend:
		enabled = s.policy.Amend
	}
	if enabled {
		return nil
	}
	return &DisabledActionError{Action: action}
}

func placeIntentSupported(intent orderintent.PlaceIntent) bool {
	return intent.Market == "us" &&
		intent.Side == "buy" &&
		intent.OrderType == "limit" &&
		intent.CurrencyMode == "KRW" &&
		!intent.Fractional
}

func (s *Service) waitForCanceledOrder(ctx context.Context, orderID string) error {
	for attempt := 0; attempt < cancelReconcileAttempts; attempt++ {
		stillPending, err := s.broker.HasPendingOrder(ctx, orderID)
		if err != nil {
			return err
		}
		if !stillPending {
			return nil
		}
		if attempt == cancelReconcileAttempts-1 {
			break
		}

		timer := time.NewTimer(cancelReconcileInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}

	return ErrCancelStillPending
}
