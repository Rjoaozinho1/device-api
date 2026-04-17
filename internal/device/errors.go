package device

import "errors"

var (
	ErrNotFound     = errors.New("device not found")
	ErrInUse        = errors.New("device is in use")
	ErrInvalidState = errors.New("invalid device state")
	ErrInvalidInput = errors.New("invalid input")
)
