// path: chessTest/internal/game/ability_registry.go
package game

import "errors"

var (
	abilityFactory   func(Ability) (AbilityHandler, error)
	fallbackHandlers [AbilityCount][]AbilityHandler

	// ErrAbilityFactoryNotConfigured indicates no resolver has been registered.
	ErrAbilityFactoryNotConfigured = errors.New("game: ability factory not configured")
	// ErrAbilityNotRegistered indicates the resolver does not have a handler for the ability.
	ErrAbilityNotRegistered = errors.New("game: ability handler not registered")
)

// RegisterAbilityFactory installs the resolver used to construct ability handlers at runtime.
func RegisterAbilityFactory(factory func(Ability) (AbilityHandler, error)) {
	abilityFactory = factory
}

func resolveAbilityHandler(id Ability) (AbilityHandler, error) {
	if abilityFactory == nil {
		return nil, ErrAbilityFactoryNotConfigured
	}
	return abilityFactory(id)
}

func registerAbilityFallback(id Ability, handler AbilityHandler) {
	if handler == nil {
		return
	}
	idx := abilityIndex(id)
	if idx < 0 {
		return
	}
	fallbackHandlers[idx] = append(fallbackHandlers[idx], handler)
}

func fallbackHandlersFor(id Ability) []AbilityHandler {
	idx := abilityIndex(id)
	if idx < 0 {
		return nil
	}
	return fallbackHandlers[idx]
}
