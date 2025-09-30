// path: chessTest/internal/game/ability_handlers_test.go
package game

import (
	"errors"
	"reflect"
	"testing"
)

func resetAbilityRegistry(t *testing.T) {
	t.Helper()
	abilityRegistryMu.Lock()
	prev := abilityRegistry
	abilityRegistry = make(map[Ability]abilityHandlerFactory, AbilityCount)
	abilityRegistryMu.Unlock()

	t.Cleanup(func() {
		abilityRegistryMu.Lock()
		abilityRegistry = prev
		abilityRegistryMu.Unlock()
	})
}

func TestRegisterAbilityHandlerAndResolve(t *testing.T) {
	resetAbilityRegistry(t)

	handler := HandlerFuncs{
		OnMoveStartFunc: func(MoveLifecycleContext) error { return nil },
	}

	if err := RegisterAbilityHandler(AbilityScorch, func() AbilityHandler { return handler }); err != nil {
		t.Fatalf("register ability: %v", err)
	}

	instance, err := resolveAbilityHandler(AbilityScorch)
	if err != nil {
		t.Fatalf("resolve ability: %v", err)
	}

	if _, err := instance.StepBudgetModifier(StepBudgetContext{}); err != nil {
		t.Fatalf("step budget modifier: %v", err)
	}
	if err := instance.OnMoveStart(MoveLifecycleContext{}); err != nil {
		t.Fatalf("on move start: %v", err)
	}
}

func TestRegisterAbilityHandlerDuplicate(t *testing.T) {
	resetAbilityRegistry(t)

	ctor := func() AbilityHandler { return HandlerFuncs{} }
	if err := RegisterAbilityHandler(AbilityScorch, ctor); err != nil {
		t.Fatalf("first registration: %v", err)
	}
	if err := RegisterAbilityHandler(AbilityScorch, ctor); !errors.Is(err, ErrDuplicateRegistration) {
		t.Fatalf("expected ErrDuplicateRegistration, got %v", err)
	}
}

func TestRegisterAbilityHandlerNilFactory(t *testing.T) {
	resetAbilityRegistry(t)
	if err := RegisterAbilityHandler(AbilityScorch, nil); !errors.Is(err, ErrNilFactory) {
		t.Fatalf("expected ErrNilFactory, got %v", err)
	}
}

func TestRegisterAbilityHandlerInvalidAbility(t *testing.T) {
	resetAbilityRegistry(t)
	if err := RegisterAbilityHandler(AbilityNone, func() AbilityHandler { return HandlerFuncs{} }); !errors.Is(err, ErrInvalidAbility) {
		t.Fatalf("expected ErrInvalidAbility, got %v", err)
	}
}

func TestResolveAbilityHandlerMissing(t *testing.T) {
	resetAbilityRegistry(t)
	if _, err := resolveAbilityHandler(AbilityScorch); !errors.Is(err, ErrAbilityNotRegistered) {
		t.Fatalf("expected ErrAbilityNotRegistered, got %v", err)
	}
}

func TestResolveAbilityHandlerNilInstance(t *testing.T) {
	resetAbilityRegistry(t)
	if err := RegisterAbilityHandler(AbilityScorch, func() AbilityHandler { return nil }); err != nil {
		t.Fatalf("register ability: %v", err)
	}
	if _, err := resolveAbilityHandler(AbilityScorch); !errors.Is(err, ErrNilHandler) {
		t.Fatalf("expected ErrNilHandler, got %v", err)
	}
}

func TestRegisteredAbilityHandlersIsCopy(t *testing.T) {
	resetAbilityRegistry(t)
	if err := RegisterAbilityHandler(AbilityScorch, func() AbilityHandler { return HandlerFuncs{} }); err != nil {
		t.Fatalf("register ability: %v", err)
	}

	ids := registeredAbilityHandlers()
	if len(ids) != 1 || ids[0] != AbilityScorch {
		t.Fatalf("unexpected ids: %v", ids)
	}

	ids[0] = AbilityDoOver

	abilityRegistryMu.RLock()
	_, exists := abilityRegistry[AbilityScorch]
	_, mutated := abilityRegistry[AbilityDoOver]
	abilityRegistryMu.RUnlock()

	if !exists {
		t.Fatalf("ability scorch missing after mutation")
	}
	if mutated {
		t.Fatalf("registry mutated after slice modification")
	}
}

func TestHandlerFuncsLifecycleInvocation(t *testing.T) {
	t.Helper()

	var calls []string
	move := &MoveState{RemainingSteps: 3}
	hf := HandlerFuncs{
		StepBudgetModifierFunc: func(ctx StepBudgetContext) (StepBudgetDelta, error) {
			if ctx.Move != move {
				t.Fatalf("expected move pointer in step budget context")
			}
			calls = append(calls, "stepBudget")
			return StepBudgetDelta{AddSteps: 1, Notes: []string{"grant"}}, nil
		},
		OnMoveStartFunc: func(ctx MoveLifecycleContext) error {
			if ctx.Move != move {
				t.Fatalf("expected move pointer in move start context")
			}
			calls = append(calls, "moveStart")
			return nil
		},
		PrepareSegmentFunc: func(ctx *SegmentPreparationContext) error {
			if ctx.Move != move {
				t.Fatalf("expected move pointer in preparation context")
			}
			if ctx.StepCost == nil || ctx.StepBudget == nil {
				t.Fatalf("expected step cost and budget pointers")
			}
			(*ctx.StepCost)++
			*ctx.StepBudget += 2
			calls = append(calls, "prepareSegment")
			return nil
		},
		OnSegmentStartFunc: func(ctx SegmentContext) error {
			if ctx.Move != move {
				t.Fatalf("expected move pointer in segment start context")
			}
			calls = append(calls, "segmentStart")
			return nil
		},
		OnPostSegmentFunc: func(ctx PostSegmentContext) error {
			if ctx.Move != move {
				t.Fatalf("expected move pointer in post segment context")
			}
			calls = append(calls, "postSegment")
			return nil
		},
		OnSegmentResolvedFunc: func(ctx SegmentResolutionContext) error {
			if ctx.Move != move {
				t.Fatalf("expected move pointer in segment resolution context")
			}
			calls = append(calls, "segmentResolved")
			return nil
		},
		OnCaptureFunc: func(ctx CaptureContext) error {
			if ctx.Move != move {
				t.Fatalf("expected move pointer in capture context")
			}
			calls = append(calls, "capture")
			return nil
		},
		ResolveCaptureFunc: func(ctx CaptureContext) (CaptureOutcome, error) {
			if ctx.Move != move {
				t.Fatalf("expected move pointer in capture resolution context")
			}
			calls = append(calls, "resolveCapture")
			return CaptureOutcome{StepAdjustment: -1, ForceTurnEnd: true}, nil
		},
		OnTurnEndFunc: func(ctx TurnEndContext) error {
			if ctx.Move != move {
				t.Fatalf("expected move pointer in turn end context")
			}
			calls = append(calls, "turnEnd")
			return nil
		},
		ResolveTurnEndFunc: func(ctx TurnEndContext) (TurnEndOutcome, error) {
			if ctx.Move != move {
				t.Fatalf("expected move pointer in turn end resolution context")
			}
			calls = append(calls, "resolveTurnEnd")
			outcome := TurnEndOutcome{}
			outcome.AddSlow(Black, 2)
			outcome.Notes = append(outcome.Notes, "slow note")
			return outcome, nil
		},
		PlanSpecialMoveFunc: func(ctx *SpecialMoveContext) (SpecialMovePlan, bool, error) {
			if ctx.Move != move {
				t.Fatalf("expected move pointer in special move context")
			}
			calls = append(calls, "planSpecialMove")
			return SpecialMovePlan{StepCost: 1, Note: "special move", Ability: AbilitySideStep, MarkAbilityUsed: true}, true, nil
		},
		FreeContinuationFunc: func(ctx FreeContinuationContext) bool {
			if ctx.Move != move {
				t.Fatalf("expected move pointer in free continuation context")
			}
			calls = append(calls, "freeContinuation")
			return true
		},
		OnDirectionChangeFunc: func(ctx DirectionChangeContext) bool {
			if ctx.Move != move {
				t.Fatalf("expected move pointer in direction change context")
			}
			calls = append(calls, "directionChange")
			return true
		},
	}

	if delta, err := hf.StepBudgetModifier(StepBudgetContext{Move: move}); err != nil {
		t.Fatalf("step budget modifier: %v", err)
	} else if delta.AddSteps != 1 || len(delta.Notes) != 1 {
		t.Fatalf("unexpected delta: %+v", delta)
	}

	if err := hf.OnMoveStart(MoveLifecycleContext{Move: move}); err != nil {
		t.Fatalf("move start: %v", err)
	}

	cost := 2
	budget := move.RemainingSteps
	if err := hf.PrepareSegment(&SegmentPreparationContext{Move: move, StepCost: &cost, StepBudget: &budget}); err != nil {
		t.Fatalf("prepare segment: %v", err)
	}
	if cost != 3 {
		t.Fatalf("expected step cost to increase to 3, got %d", cost)
	}
	if budget != move.RemainingSteps+2 {
		t.Fatalf("expected budget to increase by 2, got %d", budget)
	}
	move.RemainingSteps = budget

	if err := hf.OnSegmentStart(SegmentContext{Move: move}); err != nil {
		t.Fatalf("segment start: %v", err)
	}
	if err := hf.OnPostSegment(PostSegmentContext{Move: move}); err != nil {
		t.Fatalf("post segment: %v", err)
	}
	if err := hf.OnSegmentResolved(SegmentResolutionContext{Move: move, StepsConsumed: cost}); err != nil {
		t.Fatalf("segment resolved: %v", err)
	}
	if err := hf.OnCapture(CaptureContext{Move: move}); err != nil {
		t.Fatalf("capture: %v", err)
	}
	if outcome, err := hf.ResolveCapture(CaptureContext{Move: move}); err != nil {
		t.Fatalf("resolve capture: %v", err)
	} else if outcome.StepAdjustment != -1 || !outcome.ForceTurnEnd {
		t.Fatalf("unexpected capture outcome: %+v", outcome)
	}
	if err := hf.OnTurnEnd(TurnEndContext{Move: move}); err != nil {
		t.Fatalf("turn end: %v", err)
	}
	if outcome, err := hf.ResolveTurnEnd(TurnEndContext{Move: move}); err != nil {
		t.Fatalf("resolve turn end: %v", err)
	} else {
		if got := outcome.Slow[Black]; got != 2 {
			t.Fatalf("expected slow of 2 on black, got %d", got)
		}
		if len(outcome.Notes) != 1 || outcome.Notes[0] != "slow note" {
			t.Fatalf("unexpected turn end notes: %+v", outcome.Notes)
		}
	}
	if plan, handled, err := hf.PlanSpecialMove(&SpecialMoveContext{Move: move}); err != nil {
		t.Fatalf("plan special move: %v", err)
	} else {
		if !handled {
			t.Fatalf("expected special move to be handled")
		}
		if plan.StepCost != 1 || plan.Note != "special move" || !plan.MarkAbilityUsed || plan.Ability != AbilitySideStep {
			t.Fatalf("unexpected special move plan: %+v", plan)
		}
	}
	if !hf.FreeContinuationAvailable(FreeContinuationContext{Move: move}) {
		t.Fatalf("expected free continuation to be available")
	}
	if !hf.OnDirectionChange(DirectionChangeContext{Move: move}) {
		t.Fatalf("expected direction change handler to claim event")
	}

	expected := []string{"stepBudget", "moveStart", "prepareSegment", "segmentStart", "postSegment", "segmentResolved", "capture", "resolveCapture", "turnEnd", "resolveTurnEnd", "planSpecialMove", "freeContinuation", "directionChange"}
	if !reflect.DeepEqual(calls, expected) {
		t.Fatalf("unexpected call order: %v", calls)
	}
}

func TestHandlerFuncsCaptureLimit(t *testing.T) {
	t.Helper()

	var captures int
	errLimit := errors.New("max captures reached")
	hf := HandlerFuncs{
		OnCaptureFunc: func(CaptureContext) error {
			captures++
			if captures > 2 {
				return errLimit
			}
			return nil
		},
	}

	ctx := CaptureContext{}
	if err := hf.OnCapture(ctx); err != nil {
		t.Fatalf("first capture unexpected error: %v", err)
	}
	if err := hf.OnCapture(ctx); err != nil {
		t.Fatalf("second capture unexpected error: %v", err)
	}
	if err := hf.OnCapture(ctx); !errors.Is(err, errLimit) {
		t.Fatalf("expected capture limit error, got %v", err)
	}
}

func TestHandlerFuncsSlowApplication(t *testing.T) {
	t.Helper()

	hf := HandlerFuncs{
		ResolveTurnEndFunc: func(TurnEndContext) (TurnEndOutcome, error) {
			outcome := TurnEndOutcome{}
			outcome.AddSlow(Black, 3)
			outcome.Notes = append(outcome.Notes, "slow applied")
			return outcome, nil
		},
	}

	outcome, err := hf.ResolveTurnEnd(TurnEndContext{Move: &MoveState{}})
	if err != nil {
		t.Fatalf("resolve turn end: %v", err)
	}
	if got := outcome.Slow[Black]; got != 3 {
		t.Fatalf("expected slow of 3 on black, got %d", got)
	}
	if len(outcome.Notes) != 1 || outcome.Notes[0] != "slow applied" {
		t.Fatalf("unexpected notes: %+v", outcome.Notes)
	}
}

func TestContextTypes(t *testing.T) {
	t.Helper()
	_ = StepBudgetContext{Engine: (*Engine)(nil), Piece: (*Piece)(nil), Move: (*MoveState)(nil)}
	_ = PhaseContext{Engine: (*Engine)(nil), Piece: (*Piece)(nil)}
	_ = MoveLifecycleContext{Engine: (*Engine)(nil), Move: (*MoveState)(nil), Request: MoveRequest{}, Segment: SegmentMetadata{}}
	_ = SegmentContext{Engine: (*Engine)(nil), Move: (*MoveState)(nil), Segment: SegmentMetadata{}}
	_ = CaptureContext{Engine: (*Engine)(nil), Move: (*MoveState)(nil), Attacker: (*Piece)(nil), Victim: (*Piece)(nil)}
	_ = TurnEndContext{Engine: (*Engine)(nil), Move: (*MoveState)(nil), Reason: TurnEndForced}
}
