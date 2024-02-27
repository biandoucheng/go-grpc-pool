package gogrpcpool

import "errors"

var (
	ErrConnTooManyReference = errors.New("connection too many reference")
	ErrConnIsClosing        = errors.New("connection is closing")
	ErrTargetNotAvailable   = errors.New("target not available")
	ErrNoConnAvailable      = errors.New("no connection available")
	ErrWaitConnReadyTimeout = errors.New("wait connection ready timeout")
)
