package game

import "battle_chess_poc/internal/shared"

// AbilityHandler represents the lifecycle hooks that an ability can implement
// to integrate with the engine. Handlers may implement any subset of the
// methods; unused hooks should return default values.
type AbilityHandler interface {
	StepBudgetModifier(ctx StepBudgetContext) (StepBudgetDelta, error)
	CanPhase(ctx PhaseContext) (bool, error)
	OnMoveStart(ctx MoveLifecycleContext) error
	OnSegmentStart(ctx SegmentContext) error
	OnCapture(ctx CaptureContext) error
	OnTurnEnd(ctx TurnEndContext) error
}

// StepBudgetContext carries the data required to adjust the initial step budget
// for a move.
type StepBudgetContext struct {
	Engine *Engine
	Piece  *Piece
	Move   *MoveState // nil before the MoveState is created
}

// StepBudgetDelta encapsulates modifications to the starting step budget.
type StepBudgetDelta struct {
	AddSteps int
	Notes    []string
}

// PhaseContext supplies phasing calculations with the origin and destination
// squares under consideration.
type PhaseContext struct {
	Engine *Engine
	Piece  *Piece
	From   shared.Square
	To     shared.Square
}

// MoveLifecycleContext provides data for hooks that run when a new move begins.
type MoveLifecycleContext struct {
	Engine  *Engine
	Move    *MoveState
	Request MoveRequest
	Segment SegmentMetadata
}

// SegmentMetadata mirrors the runtime data captured for a move segment.
type SegmentMetadata struct {
	Capture       *Piece
	CaptureSquare shared.Square
	EnPassant     bool
}

// SegmentContext tracks the state for a particular move segment within a turn.
type SegmentContext struct {
	Engine      *Engine
	Move        *MoveState
	From        shared.Square
	To          shared.Square
	Segment     SegmentMetadata
	SegmentStep int // zero-based index within the turn
}

// CaptureContext mirrors the data supplied when a capture occurs during a
// segment.
type CaptureContext struct {
	Engine        *Engine
	Move          *MoveState
	Attacker      *Piece
	Victim        *Piece
	CaptureSquare shared.Square
	SegmentStep   int
}

// TurnEndContext communicates the reason a turn is ending along with the
// active runtime state.
type TurnEndContext struct {
	Engine *Engine
	Move   *MoveState
	Reason TurnEndReason
}

// TurnEndReason describes why a turn is finishing.
type TurnEndReason int

const (
	// TurnEndNatural indicates the player completed their turn normally.
	TurnEndNatural TurnEndReason = iota
	// TurnEndForced signals that an effect or rule forced the turn to stop.
	TurnEndForced
	// TurnEndCancelled denotes that the turn was aborted due to an error or veto.
	TurnEndCancelled
)

// abilityContextCache stores reusable context structs for ability handler
// invocations within a single Move() call. The structs are reused to avoid
// repeated allocations while ensuring callers reset the cache between moves.
type abilityContextCache struct {
	stepBudget StepBudgetContext
	phase      PhaseContext
	move       MoveLifecycleContext
	segment    SegmentContext
	capture    CaptureContext
	turnEnd    TurnEndContext
}

// clear zeroes the cached contexts so subsequent moves cannot observe stale
// pointers or metadata from earlier invocations.
func (c *abilityContextCache) clear() {
	*c = abilityContextCache{}
}

// usage reports which cached contexts currently hold non-nil references.
func (c abilityContextCache) usage() map[string]bool {
	usage := make(map[string]bool)
	if c.stepBudget.Engine != nil || c.stepBudget.Piece != nil || c.stepBudget.Move != nil {
		usage["stepBudget"] = true
	}
	if c.phase.Engine != nil || c.phase.Piece != nil {
		usage["phase"] = true
	}
	if c.move.Engine != nil || c.move.Move != nil || (c.move.Request != (MoveRequest{})) {
		usage["move"] = true
	}
	if c.segment.Engine != nil || c.segment.Move != nil || c.segment.Segment.Capture != nil {
		usage["segment"] = true
	}
	if c.capture.Engine != nil || c.capture.Attacker != nil || c.capture.Victim != nil {
		usage["capture"] = true
	}
	if c.turnEnd.Engine != nil || c.turnEnd.Move != nil || c.turnEnd.Reason != 0 {
		usage["turnEnd"] = true
	}
	if len(usage) == 0 {
		return nil
	}
	return usage
}
