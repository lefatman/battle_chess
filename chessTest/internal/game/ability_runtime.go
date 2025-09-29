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
	OnPostSegment(ctx PostSegmentContext) error
	OnCapture(ctx CaptureContext) error
	OnTurnEnd(ctx TurnEndContext) error
}

// SegmentPreparationHandler allows abilities to inspect and modify a segment's
// pending cost or remaining step budget before standard validation occurs.
// Implementers may adjust the pointed values in the context to charge
// additional costs, grant discounts, or even consume steps outright. Returning
// an error vetoes the segment and aborts the move continuation.
type SegmentPreparationHandler interface {
	PrepareSegment(ctx *SegmentPreparationContext) error
}

// SegmentResolutionHandler runs after a segment successfully resolves but
// before capture handlers or continuation checks execute. It mirrors the
// existing post-segment bookkeeping performed by inline ability logic so
// handlers can toggle once-per-turn flags, publish notes, or otherwise update
// their runtime state.
type SegmentResolutionHandler interface {
	OnSegmentResolved(ctx SegmentResolutionContext) error
}

// FreeContinuationHandler exposes whether an ability can grant a free
// continuation after the current segment resolves.
type FreeContinuationHandler interface {
	FreeContinuationAvailable(ctx FreeContinuationContext) bool
}

// DirectionChangeHandler allows abilities to inspect direction changes after
// a segment and optionally consume the default note logging.
type DirectionChangeHandler interface {
	OnDirectionChange(ctx DirectionChangeContext) bool
}

// SpecialMoveHandler lets abilities claim ownership of special continuation
// inputs (such as Side Step or Quantum Step). Handlers return a plan that
// describes the step cost, execution action, and any post-resolution
// bookkeeping required. A "false" handled flag instructs the engine to fall
// back to its default implementation.
type SpecialMoveHandler interface {
	PlanSpecialMove(ctx *SpecialMoveContext) (SpecialMovePlan, bool, error)
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

// SegmentPreparationContext supplies handlers with mutable step-budget
// pointers so they can apply surcharges or discounts before the engine checks
// affordability. Handlers may freely mutate the pointed values; the engine will
// read the adjusted totals after all handlers run.
type SegmentPreparationContext struct {
	Engine      *Engine
	Move        *MoveState
	From        shared.Square
	To          shared.Square
	Segment     SegmentMetadata
	SegmentStep int
	StepCost    *int
	StepBudget  *int
}

// SegmentResolutionContext mirrors the segment metadata made available to
// standard hooks while also communicating how many steps the engine deducted for
// the segment. Handlers can inspect the remaining budget via ctx.Move.
type SegmentResolutionContext struct {
	Engine        *Engine
	Move          *MoveState
	From          shared.Square
	To            shared.Square
	Segment       SegmentMetadata
	SegmentStep   int
	StepsConsumed int
}

// PostSegmentContext captures the data required for handlers to react to the
// completion of a segment before capture aftermath logic runs.
type PostSegmentContext struct {
	Engine      *Engine
	Move        *MoveState
	Piece       *Piece
	From        shared.Square
	To          shared.Square
	Segment     SegmentMetadata
	SegmentStep int
}

// SpecialMoveAction enumerates the execution strategies supported by
// SpecialMovePlan.
type SpecialMoveAction int

const (
	SpecialMoveActionNone SpecialMoveAction = iota
	SpecialMoveActionMove
	SpecialMoveActionSwap
)

// SpecialMoveContext conveys the attempted displacement to ability handlers and
// exposes mutable planning fields. The SegmentStep mirrors the zero-based index
// used throughout the move lifecycle.
type SpecialMoveContext struct {
	Engine      *Engine
	Move        *MoveState
	Piece       *Piece
	From        shared.Square
	To          shared.Square
	Ability     Ability
	SegmentStep int
}

// SpecialMovePlan instructs the engine how to execute a handler-claimed special
// move. The engine applies the requested cost, performs the action, and runs the
// shared post-segment pipeline after execution.
type SpecialMovePlan struct {
	StepCost          int
	Action            SpecialMoveAction
	SwapWith          *Piece
	Metadata          SegmentMetadata
	Note              string
	Ability           Ability
	MarkAbilityUsed   bool
	ResetResurrection bool
	ClampRemaining    bool
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

// FreeContinuationContext communicates the data necessary to determine whether
// an ability can grant a free continuation after the current segment.
type FreeContinuationContext struct {
	Engine  *Engine
	Move    *MoveState
	Piece   *Piece
	Ability Ability
}

// DirectionChangeContext provides the squares and directions involved in a
// post-segment direction change so handlers can react appropriately.
type DirectionChangeContext struct {
	Engine            *Engine
	Move              *MoveState
	Piece             *Piece
	PreviousStart     shared.Square
	PreviousEnd       shared.Square
	CurrentEnd        shared.Square
	PreviousDirection Direction
	CurrentDirection  Direction
	SegmentStep       int
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
	stepBudget      StepBudgetContext
	phase           PhaseContext
	move            MoveLifecycleContext
	segment         SegmentContext
	segmentPrep     SegmentPreparationContext
	segmentResolved SegmentResolutionContext
	postSegment     PostSegmentContext
	continuation    FreeContinuationContext
	direction       DirectionChangeContext
	capture         CaptureContext
	turnEnd         TurnEndContext
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
	if c.segmentPrep.Engine != nil || c.segmentPrep.Move != nil || c.segmentPrep.StepCost != nil {
		usage["segmentPrep"] = true
	}
	if c.segmentResolved.Engine != nil || c.segmentResolved.Move != nil || c.segmentResolved.StepsConsumed != 0 {
		usage["segmentResolved"] = true
	}
	if c.postSegment.Engine != nil || c.postSegment.Move != nil || c.postSegment.Piece != nil {
		usage["postSegment"] = true
	}
	if c.continuation.Engine != nil || c.continuation.Move != nil || c.continuation.Piece != nil || c.continuation.Ability != AbilityNone {
		usage["continuation"] = true
	}
	if c.direction.Engine != nil || c.direction.Move != nil || c.direction.Piece != nil || c.direction.PreviousDirection != 0 || c.direction.CurrentDirection != 0 {
		usage["direction"] = true
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
