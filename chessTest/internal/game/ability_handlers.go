// path: chessTest/internal/game/ability_handlers.go
package game

import (
	"errors"
	"fmt"
	"sync"
)

type abilityHandlerFactory func() AbilityHandler

var (
	abilityRegistryMu sync.RWMutex
	abilityRegistry   map[Ability]abilityHandlerFactory

	ErrDuplicateRegistration = errors.New("game: ability handler already registered")
	ErrNilFactory            = errors.New("game: nil ability handler factory")
	ErrInvalidAbility        = errors.New("game: invalid ability identifier")
	ErrAbilityNotRegistered  = errors.New("game: ability handler not registered")
	ErrNilHandler            = errors.New("game: ability handler factory produced nil handler")
)

// RegisterAbilityHandler associates an ability with a handler factory.
func RegisterAbilityHandler(id Ability, ctor func() AbilityHandler) error {
	if id == AbilityNone {
		return ErrInvalidAbility
	}
	if ctor == nil {
		return ErrNilFactory
	}

	abilityRegistryMu.Lock()
	defer abilityRegistryMu.Unlock()
	if abilityRegistry == nil {
		abilityRegistry = make(map[Ability]abilityHandlerFactory, AbilityCount)
	}
	if _, exists := abilityRegistry[id]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateRegistration, id.String())
	}
	abilityRegistry[id] = ctor
	return nil
}

func resolveAbilityHandler(id Ability) (AbilityHandler, error) {
	abilityRegistryMu.RLock()
	ctor := abilityRegistry[id]
	abilityRegistryMu.RUnlock()

	if ctor == nil {
		return nil, fmt.Errorf("%w: %s", ErrAbilityNotRegistered, id.String())
	}
	handler := ctor()
	if handler == nil {
		return nil, fmt.Errorf("%w: %s", ErrNilHandler, id.String())
	}
	return handler, nil
}

func registeredAbilityHandlers() []Ability {
	abilityRegistryMu.RLock()
	defer abilityRegistryMu.RUnlock()

	out := make([]Ability, 0, len(abilityRegistry))
	for id := range abilityRegistry {
		out = append(out, id)
	}
	return out
}

// HandlerFuncs allows handler implementations to override only the hooks they need.
type HandlerFuncs struct {
	StepBudgetModifierFunc func(StepBudgetContext) (StepBudgetDelta, error)
	CanPhaseFunc           func(PhaseContext) (bool, error)
	OnMoveStartFunc        func(MoveLifecycleContext) error
	OnSegmentStartFunc     func(SegmentContext) error
	OnPostSegmentFunc      func(PostSegmentContext) error
	PrepareSegmentFunc     func(*SegmentPreparationContext) error
	OnSegmentResolvedFunc  func(SegmentResolutionContext) error
	OnCaptureFunc          func(CaptureContext) error
	OnTurnEndFunc          func(TurnEndContext) error
	ResolveCaptureFunc     func(CaptureContext) (CaptureOutcome, error)
	ResolveTurnEndFunc     func(TurnEndContext) (TurnEndOutcome, error)
	PlanSpecialMoveFunc    func(*SpecialMoveContext) (SpecialMovePlan, bool, error)
	FreeContinuationFunc   func(FreeContinuationContext) bool
	OnDirectionChangeFunc  func(DirectionChangeContext) bool
}

func (hf HandlerFuncs) StepBudgetModifier(ctx StepBudgetContext) (StepBudgetDelta, error) {
	if hf.StepBudgetModifierFunc == nil {
		return StepBudgetDelta{}, nil
	}
	return hf.StepBudgetModifierFunc(ctx)
}

func (hf HandlerFuncs) CanPhase(ctx PhaseContext) (bool, error) {
	if hf.CanPhaseFunc == nil {
		return false, nil
	}
	return hf.CanPhaseFunc(ctx)
}

func (hf HandlerFuncs) OnMoveStart(ctx MoveLifecycleContext) error {
	if hf.OnMoveStartFunc == nil {
		return nil
	}
	return hf.OnMoveStartFunc(ctx)
}

func (hf HandlerFuncs) OnSegmentStart(ctx SegmentContext) error {
	if hf.OnSegmentStartFunc == nil {
		return nil
	}
	return hf.OnSegmentStartFunc(ctx)
}

func (hf HandlerFuncs) OnPostSegment(ctx PostSegmentContext) error {
	if hf.OnPostSegmentFunc == nil {
		return nil
	}
	return hf.OnPostSegmentFunc(ctx)
}

func (hf HandlerFuncs) PrepareSegment(ctx *SegmentPreparationContext) error {
	if hf.PrepareSegmentFunc == nil {
		return nil
	}
	return hf.PrepareSegmentFunc(ctx)
}

func (hf HandlerFuncs) OnSegmentResolved(ctx SegmentResolutionContext) error {
	if hf.OnSegmentResolvedFunc == nil {
		return nil
	}
	return hf.OnSegmentResolvedFunc(ctx)
}

func (hf HandlerFuncs) OnCapture(ctx CaptureContext) error {
	if hf.OnCaptureFunc == nil {
		return nil
	}
	return hf.OnCaptureFunc(ctx)
}

func (hf HandlerFuncs) OnTurnEnd(ctx TurnEndContext) error {
	if hf.OnTurnEndFunc == nil {
		return nil
	}
	return hf.OnTurnEndFunc(ctx)
}

func (hf HandlerFuncs) ResolveCapture(ctx CaptureContext) (CaptureOutcome, error) {
	if hf.ResolveCaptureFunc == nil {
		return CaptureOutcome{}, nil
	}
	return hf.ResolveCaptureFunc(ctx)
}

func (hf HandlerFuncs) ResolveTurnEnd(ctx TurnEndContext) (TurnEndOutcome, error) {
	if hf.ResolveTurnEndFunc == nil {
		return TurnEndOutcome{}, nil
	}
	return hf.ResolveTurnEndFunc(ctx)
}

func (hf HandlerFuncs) PlanSpecialMove(ctx *SpecialMoveContext) (SpecialMovePlan, bool, error) {
	if hf.PlanSpecialMoveFunc == nil {
		return SpecialMovePlan{}, false, nil
	}
	return hf.PlanSpecialMoveFunc(ctx)
}

func (hf HandlerFuncs) FreeContinuationAvailable(ctx FreeContinuationContext) bool {
	if hf.FreeContinuationFunc == nil {
		return false
	}
	return hf.FreeContinuationFunc(ctx)
}

func (hf HandlerFuncs) OnDirectionChange(ctx DirectionChangeContext) bool {
	if hf.OnDirectionChangeFunc == nil {
		return false
	}
	return hf.OnDirectionChangeFunc(ctx)
}
