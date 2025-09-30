// path: chessTest/internal/game/move_state.go
package game

// MoveState tracks the current move in progress with a step budget system.
// It holds all the temporary state for a piece's actions within a single turn.
type MoveState struct {
	Piece               *Piece
	RemainingSteps      int
	Path                []Square
	Captures            []*Piece
	AbilityData         abilityRuntimeTable
	TurnEnded           bool
	LastSegmentCaptured bool
	Promotion           PieceType
	PromotionSet        bool
	Handlers            *abilityHandlerTable
}

func initializeMoveState(pc *Piece, start Square, remaining int, handlers *abilityHandlerTable, promotion PieceType, promotionSet bool) *MoveState {
	abilities := AbilityList(nil)
	if pc != nil {
		abilities = pc.Abilities
	}
	move := &MoveState{
		Piece:          pc,
		RemainingSteps: remaining,
		Path:           []Square{start},
		Captures:       make([]*Piece, 0, 2),
		AbilityData:    newAbilityRuntimeTable(abilities),
		Promotion:      promotion,
		PromotionSet:   promotionSet,
		Handlers:       handlers,
	}
	move.setCaptureLimit(1)
	move.setAbilityCounter(AbilityNone, abilityCounterCaptures, len(move.Captures))
	move.setAbilityCounter(AbilityNone, abilityCounterCaptureSegment, -1)
	move.setAbilityCounter(AbilityNone, abilityCounterCaptureSquare, -1)
	move.setAbilityCounter(AbilityNone, abilityCounterCaptureEnPassant, 0)
	move.setAbilityCounter(AbilityResurrection, abilityCounterResurrectionWindow, 0)
	return move
}

func (ms *MoveState) abilityRuntime(id Ability) *AbilityRuntime {
	if ms == nil {
		return nil
	}
	return ms.AbilityData.ensure(id)
}

func (ms *MoveState) abilityFlag(id Ability, key abilityFlagIndex) bool {
	if ms == nil {
		return false
	}
	if rt, ok := ms.AbilityData.get(id); ok {
		return rt.flag(key)
	}
	return false
}

func (ms *MoveState) setAbilityFlag(id Ability, key abilityFlagIndex, value bool) {
	if ms == nil {
		return
	}
	if rt := ms.abilityRuntime(id); rt != nil {
		rt.setFlag(key, value)
	}
}

func (ms *MoveState) abilityUsed(id Ability) bool {
	return ms.abilityFlag(id, abilityFlagUsed)
}

func (ms *MoveState) markAbilityUsed(id Ability) {
	ms.setAbilityFlag(id, abilityFlagUsed, true)
}

func (ms *MoveState) abilityCounter(id Ability, key abilityCounterIndex) int {
	if ms == nil {
		return 0
	}
	if rt, ok := ms.AbilityData.get(id); ok {
		return rt.counter(key)
	}
	return 0
}

func (ms *MoveState) setAbilityCounter(id Ability, key abilityCounterIndex, value int) {
	if ms == nil {
		return
	}
	if rt := ms.abilityRuntime(id); rt != nil {
		rt.setCounter(key, value)
	}
}

func (ms *MoveState) addAbilityCounter(id Ability, key abilityCounterIndex, delta int) int {
	if ms == nil {
		return 0
	}
	if rt := ms.abilityRuntime(id); rt != nil {
		return rt.addCounter(key, delta)
	}
	return 0
}

func (ms *MoveState) handlersFor(id Ability) []AbilityHandler {
	if ms == nil || ms.Handlers == nil {
		return nil
	}
	return ms.Handlers.handlersFor(id)
}
