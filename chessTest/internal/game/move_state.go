// path: chessTest/internal/game/move_state.go
package game

// MoveState tracks the current move in progress with a step budget system.
// It holds all the temporary state for a piece's actions within a single turn.
type MoveState struct {
	Piece               *Piece
	RemainingSteps      int
	Path                []Square
	Captures            []*Piece
	AbilityData         map[Ability]*AbilityRuntime
	TurnEnded           bool
	LastSegmentCaptured bool
	Promotion           PieceType
	PromotionSet        bool
	Handlers            map[Ability][]AbilityHandler
}

const (
	abilityFlagUsed         = "used"
	abilityFlagWindow       = "window"
	abilityFlagCaptureExtra = "captureExtra"

	abilityCounterFree             = "free"
	abilityCounterCaptures         = "captures"
	abilityCounterCaptureLimit     = "captureLimit"
	abilityCounterCaptureSquare    = "captureSquare"
	abilityCounterCaptureSegment   = "captureSegment"
	abilityCounterCaptureEnPassant = "captureEnPassant"
	abilityCounterResurrectionHold = "resurrectionHold"
)

func initializeMoveState(pc *Piece, start Square, remaining int, handlers map[Ability][]AbilityHandler, promotion PieceType, promotionSet bool) *MoveState {
	move := &MoveState{
		Piece:          pc,
		RemainingSteps: remaining,
		Path:           []Square{start},
		Captures:       make([]*Piece, 0, 2),
		AbilityData:    newAbilityRuntimeMap(pc.Abilities),
		Promotion:      promotion,
		PromotionSet:   promotionSet,
		Handlers:       handlers,
	}
	move.setCaptureLimit(1)
	move.setAbilityCounter(AbilityNone, abilityCounterCaptures, len(move.Captures))
	move.setAbilityCounter(AbilityNone, abilityCounterCaptureSegment, -1)
	move.setAbilityCounter(AbilityNone, abilityCounterCaptureSquare, -1)
	move.setAbilityCounter(AbilityNone, abilityCounterCaptureEnPassant, 0)
	return move
}

func (ms *MoveState) ensureAbilityData() {
	if ms.AbilityData == nil {
		ms.AbilityData = make(map[Ability]*AbilityRuntime)
	}
}

func (ms *MoveState) abilityRuntime(id Ability) *AbilityRuntime {
	if ms == nil {
		return nil
	}
	ms.ensureAbilityData()
	rt, ok := ms.AbilityData[id]
	if !ok {
		rt = &AbilityRuntime{}
		ms.AbilityData[id] = rt
	}
	return rt
}

func (ms *MoveState) abilityFlag(id Ability, key string) bool {
	if ms == nil || len(ms.AbilityData) == 0 {
		return false
	}
	if rt, ok := ms.AbilityData[id]; ok {
		return rt.flag(key)
	}
	return false
}

func (ms *MoveState) setAbilityFlag(id Ability, key string, value bool) {
	if ms == nil {
		return
	}
	ms.abilityRuntime(id).setFlag(key, value)
}

func (ms *MoveState) abilityUsed(id Ability) bool {
	return ms.abilityFlag(id, abilityFlagUsed)
}

func (ms *MoveState) markAbilityUsed(id Ability) {
	ms.setAbilityFlag(id, abilityFlagUsed, true)
}

func (ms *MoveState) clearAbilityUsed(id Ability) {
	ms.setAbilityFlag(id, abilityFlagUsed, false)
}

func (ms *MoveState) abilityCounter(id Ability, key string) int {
	if ms == nil || len(ms.AbilityData) == 0 {
		return 0
	}
	if rt, ok := ms.AbilityData[id]; ok {
		return rt.counter(key)
	}
	return 0
}

func (ms *MoveState) setAbilityCounter(id Ability, key string, value int) {
	if ms == nil {
		return
	}
	ms.abilityRuntime(id).setCounter(key, value)
}

func (ms *MoveState) addAbilityCounter(id Ability, key string, delta int) int {
	if ms == nil {
		return 0
	}
	return ms.abilityRuntime(id).addCounter(key, delta)
}

func (ms *MoveState) handlersFor(id Ability) []AbilityHandler {
	if ms == nil || len(ms.Handlers) == 0 {
		return nil
	}
	return ms.Handlers[id]
}
