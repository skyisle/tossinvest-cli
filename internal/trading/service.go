package trading

import (
	"context"
	"crypto/subtle"
	"fmt"

	"github.com/junghoonkye/tossinvest-cli/internal/config"
	"github.com/junghoonkye/tossinvest-cli/internal/orderintent"
	"github.com/junghoonkye/tossinvest-cli/internal/permissions"
)

type Broker interface {
	PlacePendingOrder(ctx context.Context, intent orderintent.PlaceIntent) (MutationResult, error)
	GetOrderAvailableActions(ctx context.Context, orderID string) (map[string]any, error)
	CancelPendingOrder(ctx context.Context, intent orderintent.CancelIntent) (MutationResult, error)
	AmendPendingOrder(ctx context.Context, intent orderintent.AmendIntent) (MutationResult, error)
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
		"Live place supports US/KR buy/sell limit orders and US fractional (market) orders in KRW.",
		"US orders may still require funding, FX consent, or product-risk acknowledgement before submission.",
	}
	liveReady := placeIntentSupported(intent)
	if !s.policy.Place {
		warnings = append(warnings, "Config currently disables `order place`.")
	}
	if intent.Market == "kr" && !s.policy.KR {
		warnings = append(warnings, "Config currently disables `order place --market kr`.")
	}
	if intent.Side == "sell" && !s.policy.Sell {
		warnings = append(warnings, "Config currently disables `order place --side sell`.")
	}
	if intent.Fractional && !s.policy.Fractional {
		warnings = append(warnings, "Config currently disables `order place --fractional`.")
	}
	if !s.policy.AllowLiveOrderActions {
		warnings = append(warnings, "Config currently disables live order actions.")
	}
	mutationReady := liveReady && s.policy.Place && s.policy.AllowLiveOrderActions
	if intent.Market == "kr" {
		mutationReady = mutationReady && s.policy.KR
	}
	if intent.Side == "sell" {
		mutationReady = mutationReady && s.policy.Sell
	}
	if intent.Fractional {
		mutationReady = mutationReady && s.policy.Fractional
	}
	return Preview{
		Kind:          "place",
		Canonical:     canonical,
		ConfirmToken:  orderintent.ConfirmToken(canonical),
		Warnings:      warnings,
		LiveReady:     liveReady,
		MutationReady: mutationReady,
	}
}

func (s *Service) PreviewCancel(intent orderintent.CancelIntent) Preview {
	canonical := orderintent.CanonicalCancel(intent)
	warnings := []string{"Single-order cancel is wired for same-day pending orders and still reconciles through pending history."}
	if !s.policy.Cancel {
		warnings = append(warnings, "Config currently disables `order cancel`.")
	}
	if !s.policy.AllowLiveOrderActions {
		warnings = append(warnings, "Config currently disables live order actions.")
	}
	return Preview{
		Kind:          "cancel",
		Canonical:     canonical,
		ConfirmToken:  orderintent.ConfirmToken(canonical),
		Warnings:      warnings,
		LiveReady:     true,
		MutationReady: s.policy.Cancel && s.policy.AllowLiveOrderActions,
	}
}

func (s *Service) PreviewAmend(intent orderintent.AmendIntent) Preview {
	canonical := orderintent.CanonicalAmend(intent)
	warnings := []string{
		"Amend reconciles against the surviving pending order record after mutation.",
		"Amend wiring exists, but the current beta slice still needs more live verification.",
	}
	if !s.policy.Amend {
		warnings = append(warnings, "Config currently disables `order amend`.")
	}
	if !s.policy.AllowLiveOrderActions {
		warnings = append(warnings, "Config currently disables live order actions.")
	}
	return Preview{
		Kind:          "amend",
		Canonical:     canonical,
		ConfirmToken:  orderintent.ConfirmToken(canonical),
		Warnings:      warnings,
		LiveReady:     true,
		MutationReady: s.policy.Amend && s.policy.AllowLiveOrderActions,
	}
}

func (s *Service) Place(ctx context.Context, intent orderintent.PlaceIntent, opts ExecuteOptions) (MutationResult, error) {
	// 1. capability check
	if !placeIntentSupported(intent) {
		return MutationResult{}, ErrPlaceUnsupported
	}
	// 2. market policy
	if intent.Market == "kr" && !s.policy.KR {
		return MutationResult{}, &DisabledActionError{Action: "kr"}
	}
	// 3. side policy
	if intent.Side == "sell" && !s.policy.Sell {
		return MutationResult{}, &DisabledActionError{Action: "sell"}
	}
	// 4. fractional policy
	if intent.Fractional && !s.policy.Fractional {
		return MutationResult{}, &DisabledActionError{Action: "fractional"}
	}
	// 5. execution guard (--execute, permissions, confirm)
	if err := s.guard(ctx, ActionPlace, s.PreviewPlace(intent), opts); err != nil {
		return MutationResult{}, err
	}
	// 5. broker call
	if s.broker == nil {
		return MutationResult{}, ErrLiveMutationPending
	}
	return s.broker.PlacePendingOrder(ctx, intent)
}

func (s *Service) Cancel(ctx context.Context, intent orderintent.CancelIntent, opts ExecuteOptions) (MutationResult, error) {
	if err := s.guard(ctx, ActionCancel, s.PreviewCancel(intent), opts); err != nil {
		return MutationResult{}, err
	}
	if s.broker == nil {
		return MutationResult{}, ErrLiveMutationPending
	}

	if _, err := s.broker.GetOrderAvailableActions(ctx, intent.OrderID); err != nil {
		return MutationResult{}, err
	}
	return s.broker.CancelPendingOrder(ctx, intent)
}

func (s *Service) Amend(ctx context.Context, intent orderintent.AmendIntent, opts ExecuteOptions) (MutationResult, error) {
	if err := s.guard(ctx, ActionAmend, s.PreviewAmend(intent), opts); err != nil {
		return MutationResult{}, err
	}
	if s.broker == nil {
		return MutationResult{}, ErrLiveMutationPending
	}
	if _, err := s.broker.GetOrderAvailableActions(ctx, intent.OrderID); err != nil {
		return MutationResult{}, err
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
	if !s.policy.AllowLiveOrderActions {
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
	if intent.Market != "us" && intent.Market != "kr" {
		return false
	}
	if intent.Fractional {
		// fractional orders: US market + market order, KRW or USD
		return intent.Market == "us" && intent.OrderType == "market" &&
			(intent.CurrencyMode == "KRW" || intent.CurrencyMode == "USD")
	}
	// non-fractional: KRW only, limit only
	if intent.CurrencyMode != "KRW" {
		return false
	}
	return intent.OrderType == "limit"
}
