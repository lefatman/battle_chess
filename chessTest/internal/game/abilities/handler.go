package abilities

import (
	"errors"
	"fmt"
	"sync"

	"battle_chess_poc/internal/game"
	"battle_chess_poc/internal/shared"
)

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

// HandlerFactory constructs a new AbilityHandler instance.
type HandlerFactory func() AbilityHandler

// StepBudgetContext carries the data required to adjust the initial step budget
// for a move.
type StepBudgetContext struct {
	Engine *game.Engine
	Piece  *game.Piece
	Move   *game.MoveState // nil before the MoveState is created
}

// StepBudgetDelta encapsulates modifications to the starting step budget.
type StepBudgetDelta struct {
	AddSteps int
	Notes    []string
}

// PhaseContext supplies phasing calculations with the origin and destination
// squares under consideration.
type PhaseContext struct {
	Engine *game.Engine
	Piece  *game.Piece
	From   shared.Square
	To     shared.Square
}

// MoveLifecycleContext provides data for hooks that run when a new move begins.
type MoveLifecycleContext struct {
	Engine  *game.Engine
	Move    *game.MoveState
	Request game.MoveRequest
	Segment SegmentMetadata
}

// SegmentMetadata mirrors the runtime data captured for a move segment.
type SegmentMetadata struct {
	Capture       *game.Piece
	CaptureSquare shared.Square
	EnPassant     bool
}

// SegmentContext tracks the state for a particular move segment within a turn.
type SegmentContext struct {
	Engine      *game.Engine
	Move        *game.MoveState
	From        shared.Square
	To          shared.Square
	Segment     SegmentMetadata
	SegmentStep int // zero-based index within the turn
}

// CaptureContext mirrors the data supplied when a capture occurs during a
// segment.
type CaptureContext struct {
	Engine        *game.Engine
	Move          *game.MoveState
	Attacker      *game.Piece
	Victim        *game.Piece
	CaptureSquare shared.Square
	SegmentStep   int
}

// TurnEndContext communicates the reason a turn is ending along with the
// active runtime state.
type TurnEndContext struct {
	Engine *game.Engine
	Move   *game.MoveState
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

// HandlerFuncs provides a convenient adapter that allows handlers to override
// only the hooks they need. Any nil function pointer results in a neutral
// response so the registry can safely skip unused hooks.
type HandlerFuncs struct {
	StepBudgetModifierFunc func(StepBudgetContext) (StepBudgetDelta, error)
	CanPhaseFunc           func(PhaseContext) (bool, error)
	OnMoveStartFunc        func(MoveLifecycleContext) error
	OnSegmentStartFunc     func(SegmentContext) error
	OnCaptureFunc          func(CaptureContext) error
	OnTurnEndFunc          func(TurnEndContext) error
}

// StepBudgetModifier invokes the configured modifier hook if present.
func (hf HandlerFuncs) StepBudgetModifier(ctx StepBudgetContext) (StepBudgetDelta, error) {
	if hf.StepBudgetModifierFunc == nil {
		return StepBudgetDelta{}, nil
	}
	return hf.StepBudgetModifierFunc(ctx)
}

// CanPhase invokes the configured phasing hook if present.
func (hf HandlerFuncs) CanPhase(ctx PhaseContext) (bool, error) {
	if hf.CanPhaseFunc == nil {
		return false, nil
	}
	return hf.CanPhaseFunc(ctx)
}

// OnMoveStart invokes the configured move-start hook if present.
func (hf HandlerFuncs) OnMoveStart(ctx MoveLifecycleContext) error {
	if hf.OnMoveStartFunc == nil {
		return nil
	}
	return hf.OnMoveStartFunc(ctx)
}

// OnSegmentStart invokes the configured segment-start hook if present.
func (hf HandlerFuncs) OnSegmentStart(ctx SegmentContext) error {
	if hf.OnSegmentStartFunc == nil {
		return nil
	}
	return hf.OnSegmentStartFunc(ctx)
}

// OnCapture invokes the configured capture hook if present.
func (hf HandlerFuncs) OnCapture(ctx CaptureContext) error {
	if hf.OnCaptureFunc == nil {
		return nil
	}
	return hf.OnCaptureFunc(ctx)
}

// OnTurnEnd invokes the configured turn-end hook if present.
func (hf HandlerFuncs) OnTurnEnd(ctx TurnEndContext) error {
	if hf.OnTurnEndFunc == nil {
		return nil
	}
	return hf.OnTurnEndFunc(ctx)
}

var (
	registryMu sync.RWMutex
	registry   map[shared.Ability]HandlerFactory

	// ErrDuplicateRegistration indicates an ability already has a handler factory.
	ErrDuplicateRegistration = errors.New("abilities: handler already registered")
	// ErrNilFactory indicates a registration attempt provided a nil constructor.
	ErrNilFactory = errors.New("abilities: nil handler factory")
	// ErrInvalidAbility indicates the ability identifier is not valid for registration.
	ErrInvalidAbility = errors.New("abilities: invalid ability identifier")
	// ErrUnknownAbility indicates no handler factory has been registered for the ability.
	ErrUnknownAbility = errors.New("abilities: handler not registered")
	// ErrNilHandler indicates a factory returned a nil handler instance.
	ErrNilHandler = errors.New("abilities: factory produced nil handler")
)

// Register associates an ability with a handler factory. The function is safe
// for concurrent use.
func Register(id shared.Ability, ctor HandlerFactory) error {
	if id == shared.AbilityNone {
		return ErrInvalidAbility
	}
	if ctor == nil {
		return ErrNilFactory
	}

	registryMu.Lock()
	defer registryMu.Unlock()
	if registry == nil {
		registry = make(map[shared.Ability]HandlerFactory)
	}
	if _, exists := registry[id]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateRegistration, id.String())
	}
	registry[id] = ctor
	return nil
}

// New creates a handler instance for the requested ability using the registered
// factory.
func New(id shared.Ability) (AbilityHandler, error) {
	registryMu.RLock()
	ctor := registry[id]
	registryMu.RUnlock()

	if ctor == nil {
		return nil, fmt.Errorf("%w: %s", ErrUnknownAbility, id.String())
	}

	handler := ctor()
	if handler == nil {
		return nil, fmt.Errorf("%w: %s", ErrNilHandler, id.String())
	}
	return handler, nil
}

// registeredAbilities returns a copy of the registered ability identifiers. It
// is primarily intended for debugging and tests.
func registeredAbilities() []shared.Ability {
	registryMu.RLock()
	defer registryMu.RUnlock()

	out := make([]shared.Ability, 0, len(registry))
	for id := range registry {
		out = append(out, id)
	}
	return out
}
