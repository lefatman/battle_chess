// path: chessTest/internal/game/errors.go
package game

import "errors"

var (
	ErrEngineLocked    = errors.New("engine locked")
	ErrInvalidConfig   = errors.New("invalid configuration")
	ErrInvalidMove     = errors.New("invalid move")
	ErrDoOverActivated = errors.New("do-over activated")
	ErrCaptureBlocked  = errors.New("capture blocked")
)
