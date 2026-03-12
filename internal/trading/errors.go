package trading

import "errors"

var (
	ErrExecuteRequired          = errors.New("live trading requires --execute")
	ErrDangerousFlagRequired    = errors.New("live trading requires --dangerously-skip-permissions")
	ErrDangerousExecuteDisabled = errors.New("dangerous trading execution is disabled in config")
	ErrConfirmMismatch          = errors.New("confirmation token mismatch")
	ErrLiveMutationPending      = errors.New("live trading mutation is not implemented yet")
	ErrPlaceUnsupported         = errors.New("live place supports only a narrow subset of orders")
	ErrPlaceNotReconciled       = errors.New("placed order was not found in pending reconciliation")
	ErrCancelStillPending       = errors.New("pending order still present after cancel reconciliation")
	ErrInteractiveAuthRequired  = errors.New("broker requires interactive trade authentication")
)

type Action string

const (
	ActionGrant  Action = "grant"
	ActionPlace  Action = "place"
	ActionCancel Action = "cancel"
	ActionAmend  Action = "amend"
)

type DisabledActionError struct {
	Action Action
}

func (e *DisabledActionError) Error() string {
	return "trading action is disabled in config: " + string(e.Action)
}
