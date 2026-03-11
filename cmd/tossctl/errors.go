package main

import (
	"errors"
	"fmt"

	tossclient "github.com/junghoonkye/tossinvest-cli/internal/client"
	"github.com/junghoonkye/tossinvest-cli/internal/permissions"
	"github.com/junghoonkye/tossinvest-cli/internal/trading"
)

func userFacingCommandError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, tossclient.ErrNoSession) {
		return fmt.Errorf("no active session; run `tossctl auth login`")
	}

	if tossclient.IsAuthError(err) {
		return fmt.Errorf("stored session is no longer valid; run `tossctl auth login`")
	}
	if errors.Is(err, permissions.ErrNoGrant) || errors.Is(err, permissions.ErrExpiredGrant) {
		return fmt.Errorf("no active trading permission grant; run `tossctl order permissions grant --ttl 300`")
	}
	if errors.Is(err, trading.ErrExecuteRequired) {
		return fmt.Errorf("live trading is blocked by default; rerun with `--execute` after reviewing `tossctl order preview`")
	}
	if errors.Is(err, trading.ErrDangerousFlagRequired) {
		return fmt.Errorf("live trading requires explicit danger acknowledgement via `--dangerously-skip-permissions`")
	}
	if errors.Is(err, trading.ErrConfirmMismatch) {
		return fmt.Errorf("confirmation token mismatch; rerun `tossctl order preview` and pass the new `--confirm` token")
	}
	if errors.Is(err, trading.ErrLiveMutationPending) {
		return fmt.Errorf("permission gate passed, but live trading mutation wiring is not implemented yet")
	}
	if errors.Is(err, trading.ErrPlaceUnsupported) {
		return fmt.Errorf("live place currently supports only `--market us --side buy --type limit --currency-mode KRW` without `--fractional`")
	}
	if errors.Is(err, trading.ErrPlaceNotReconciled) {
		return fmt.Errorf("place mutation returned, but the new order was not found in pending reconciliation; check `tossctl orders list` and completed history before retrying")
	}
	if errors.Is(err, trading.ErrCancelStillPending) {
		return fmt.Errorf("cancel mutation returned, but the order is still pending; reconcile with `tossctl orders list` before retrying")
	}
	if errors.Is(err, trading.ErrInteractiveAuthRequired) {
		return fmt.Errorf("broker requested interactive trade authentication; complete the cancel in the web app and keep the browser session open")
	}

	return err
}
