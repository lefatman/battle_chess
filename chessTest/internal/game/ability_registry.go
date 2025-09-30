// path: chessTest/internal/game/ability_registry.go
package game

var (
	fallbackHandlers [AbilityCount][]AbilityHandler
)

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
