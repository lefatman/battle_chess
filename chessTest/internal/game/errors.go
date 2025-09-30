// path: chessTest/internal/game/errors.go
package game

import "errors"

type AbilityConfigError string

func (e AbilityConfigError) Error() string { return string(e) }

var (
	ErrEngineLocked                             = errors.New("engine locked")
	ErrInvalidConfig                            = errors.New("invalid configuration")
	ErrInvalidMove                              = errors.New("invalid move")
	ErrDoOverActivated                          = errors.New("do-over activated")
	ErrCaptureBlocked                           = errors.New("capture blocked")
	ErrConflictingAugmentors AbilityConfigError = "conflicting augmentors"
	ErrInvalidOverload       AbilityConfigError = "invalid overload loadout"
)
