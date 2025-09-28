package game

import "errors"

var (
	ErrDoOverActivated            = errors.New("do over activated")
	ErrBlockPathDirectionRequired = errors.New("block path direction required")
	ErrCaptureBlocked             = errors.New("capture blocked by block path")
)
