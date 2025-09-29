// path: chessTest/internal/game/abilities/handler.go
package abilities

import (
	"errors"
	"fmt"
	"sync"

	"battle_chess_poc/internal/game"
)

// HandlerFactory constructs a new AbilityHandler instance.
type HandlerFactory func() game.AbilityHandler

// Re-export runtime context and helper types from the game package so ability
// implementations can continue to reference the original names without pulling
// in the entire engine package from call sites.
type (
	AbilityHandler       = game.AbilityHandler
	StepBudgetContext    = game.StepBudgetContext
	StepBudgetDelta      = game.StepBudgetDelta
	PhaseContext         = game.PhaseContext
	MoveLifecycleContext = game.MoveLifecycleContext
	SegmentMetadata      = game.SegmentMetadata
	SegmentContext       = game.SegmentContext
	CaptureContext       = game.CaptureContext
	TurnEndContext       = game.TurnEndContext
	TurnEndReason        = game.TurnEndReason
)

const (
	TurnEndNatural   = game.TurnEndNatural
	TurnEndForced    = game.TurnEndForced
	TurnEndCancelled = game.TurnEndCancelled
)

var (
	ErrPhaseDenied = game.ErrPhaseDenied
)

// HandlerFuncs provides a convenient adapter that allows handlers to override
// only the hooks they need. Any nil function pointer results in a neutral
// response so the registry can safely skip unused hooks.
type HandlerFuncs struct {
	StepBudgetModifierFunc func(game.StepBudgetContext) (game.StepBudgetDelta, error)
	CanPhaseFunc           func(game.PhaseContext) (bool, error)
	OnMoveStartFunc        func(game.MoveLifecycleContext) error
	OnSegmentStartFunc     func(game.SegmentContext) error
	OnPostSegmentFunc      func(game.PostSegmentContext) error
	PrepareSegmentFunc     func(*game.SegmentPreparationContext) error
	OnSegmentResolvedFunc  func(game.SegmentResolutionContext) error
	OnCaptureFunc          func(game.CaptureContext) error
	OnTurnEndFunc          func(game.TurnEndContext) error
	ResolveCaptureFunc     func(game.CaptureContext) (game.CaptureOutcome, error)
	ResolveTurnEndFunc     func(game.TurnEndContext) (game.TurnEndOutcome, error)
	PlanSpecialMoveFunc    func(*game.SpecialMoveContext) (game.SpecialMovePlan, bool, error)
	FreeContinuationFunc   func(game.FreeContinuationContext) bool
	OnDirectionChangeFunc  func(game.DirectionChangeContext) bool
}

// StepBudgetModifier invokes the configured modifier hook if present.
func (hf HandlerFuncs) StepBudgetModifier(ctx game.StepBudgetContext) (game.StepBudgetDelta, error) {
	if hf.StepBudgetModifierFunc == nil {
		return game.StepBudgetDelta{}, nil
	}
	return hf.StepBudgetModifierFunc(ctx)
}

// CanPhase invokes the configured phasing hook if present.
func (hf HandlerFuncs) CanPhase(ctx game.PhaseContext) (bool, error) {
	if hf.CanPhaseFunc == nil {
		return false, nil
	}
	return hf.CanPhaseFunc(ctx)
}

// OnMoveStart invokes the configured move-start hook if present.
func (hf HandlerFuncs) OnMoveStart(ctx game.MoveLifecycleContext) error {
	if hf.OnMoveStartFunc == nil {
		return nil
	}
	return hf.OnMoveStartFunc(ctx)
}

// OnSegmentStart invokes the configured segment-start hook if present.
func (hf HandlerFuncs) OnSegmentStart(ctx game.SegmentContext) error {
	if hf.OnSegmentStartFunc == nil {
		return nil
	}
	return hf.OnSegmentStartFunc(ctx)
}

// OnPostSegment invokes the configured post-segment hook if present.
func (hf HandlerFuncs) OnPostSegment(ctx game.PostSegmentContext) error {
	if hf.OnPostSegmentFunc == nil {
		return nil
	}
	return hf.OnPostSegmentFunc(ctx)
}

// PrepareSegment invokes the configured segment-preparation hook if present.
func (hf HandlerFuncs) PrepareSegment(ctx *game.SegmentPreparationContext) error {
	if hf.PrepareSegmentFunc == nil {
		return nil
	}
	return hf.PrepareSegmentFunc(ctx)
}

// OnSegmentResolved invokes the configured post-segment hook if present.
func (hf HandlerFuncs) OnSegmentResolved(ctx game.SegmentResolutionContext) error {
	if hf.OnSegmentResolvedFunc == nil {
		return nil
	}
	return hf.OnSegmentResolvedFunc(ctx)
}

// OnCapture invokes the configured capture hook if present.
func (hf HandlerFuncs) OnCapture(ctx game.CaptureContext) error {
	if hf.OnCaptureFunc == nil {
		return nil
	}
	return hf.OnCaptureFunc(ctx)
}

// OnTurnEnd invokes the configured turn-end hook if present.
func (hf HandlerFuncs) OnTurnEnd(ctx game.TurnEndContext) error {
	if hf.OnTurnEndFunc == nil {
		return nil
	}
	return hf.OnTurnEndFunc(ctx)
}

// ResolveCapture invokes the configured capture-resolution hook if present.
func (hf HandlerFuncs) ResolveCapture(ctx game.CaptureContext) (game.CaptureOutcome, error) {
	if hf.ResolveCaptureFunc == nil {
		return game.CaptureOutcome{}, nil
	}
	return hf.ResolveCaptureFunc(ctx)
}

// ResolveTurnEnd invokes the configured turn-end resolution hook if present.
func (hf HandlerFuncs) ResolveTurnEnd(ctx game.TurnEndContext) (game.TurnEndOutcome, error) {
	if hf.ResolveTurnEndFunc == nil {
		return game.TurnEndOutcome{}, nil
	}
	return hf.ResolveTurnEndFunc(ctx)
}

// PlanSpecialMove invokes the configured special-move planner if present.
func (hf HandlerFuncs) PlanSpecialMove(ctx *game.SpecialMoveContext) (game.SpecialMovePlan, bool, error) {
	if hf.PlanSpecialMoveFunc == nil {
		return game.SpecialMovePlan{}, false, nil
	}
	return hf.PlanSpecialMoveFunc(ctx)
}

// FreeContinuationAvailable invokes the configured continuation hook if present.
func (hf HandlerFuncs) FreeContinuationAvailable(ctx game.FreeContinuationContext) bool {
	if hf.FreeContinuationFunc == nil {
		return false
	}
	return hf.FreeContinuationFunc(ctx)
}

// OnDirectionChange invokes the configured direction-change hook if present.
func (hf HandlerFuncs) OnDirectionChange(ctx game.DirectionChangeContext) bool {
	if hf.OnDirectionChangeFunc == nil {
		return false
	}
	return hf.OnDirectionChangeFunc(ctx)
}

var (
	registryMu sync.RWMutex
	registry   map[game.Ability]HandlerFactory

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
func Register(id game.Ability, ctor HandlerFactory) error {
	if id == game.AbilityNone {
		return ErrInvalidAbility
	}
	if ctor == nil {
		return ErrNilFactory
	}

	registryMu.Lock()
	defer registryMu.Unlock()
	if registry == nil {
		registry = make(map[game.Ability]HandlerFactory)
	}
	if _, exists := registry[id]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateRegistration, id.String())
	}
	registry[id] = ctor
	return nil
}

// New creates a handler instance for the requested ability using the registered
// factory.
func New(id game.Ability) (game.AbilityHandler, error) {
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
func registeredAbilities() []game.Ability {
	registryMu.RLock()
	defer registryMu.RUnlock()

	out := make([]game.Ability, 0, len(registry))
	for id := range registry {
		out = append(out, id)
	}
	return out
}

func init() {
	game.RegisterAbilityFactory(func(id game.Ability) (game.AbilityHandler, error) {
		handler, err := New(id)
		if err != nil {
			if errors.Is(err, ErrUnknownAbility) {
				return nil, game.ErrAbilityNotRegistered
			}
			return nil, err
		}
		return handler, nil
	})
}
