package trading

import "errors"

var (
	ErrExecuteRequired       = errors.New("live trading requires --execute")
	ErrDangerousFlagRequired = errors.New("live trading requires --dangerously-skip-permissions")
	ErrConfirmMismatch       = errors.New("confirmation token mismatch")
	ErrLiveMutationPending   = errors.New("live trading mutation is not implemented yet")
	ErrPlaceUnsupported      = errors.New("live place supports only a narrow subset of orders")
	ErrPlaceNotReconciled    = errors.New("placed order was not found in pending reconciliation")
	ErrCancelStillPending    = errors.New("pending order still present after cancel reconciliation")
	ErrInteractiveAuthRequired = errors.New("broker requires interactive trade authentication")
)
