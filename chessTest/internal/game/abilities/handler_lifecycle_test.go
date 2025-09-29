package abilities

import (
	"errors"
	"reflect"
	"testing"

	"battle_chess_poc/internal/game"
)

func TestHandlerFuncsLifecycleInvocation(t *testing.T) {
	t.Helper()

	var calls []string
	move := &game.MoveState{RemainingSteps: 3}
	hf := HandlerFuncs{
		StepBudgetModifierFunc: func(ctx game.StepBudgetContext) (game.StepBudgetDelta, error) {
			if ctx.Move != move {
				t.Fatalf("expected move pointer in step budget context")
			}
			calls = append(calls, "stepBudget")
			return game.StepBudgetDelta{AddSteps: 1, Notes: []string{"grant"}}, nil
		},
		OnMoveStartFunc: func(ctx game.MoveLifecycleContext) error {
			if ctx.Move != move {
				t.Fatalf("expected move pointer in move start context")
			}
			calls = append(calls, "moveStart")
			return nil
		},
		PrepareSegmentFunc: func(ctx *game.SegmentPreparationContext) error {
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
		OnSegmentStartFunc: func(ctx game.SegmentContext) error {
			if ctx.Move != move {
				t.Fatalf("expected move pointer in segment start context")
			}
			calls = append(calls, "segmentStart")
			return nil
		},
		OnPostSegmentFunc: func(ctx game.PostSegmentContext) error {
			if ctx.Move != move {
				t.Fatalf("expected move pointer in post segment context")
			}
			calls = append(calls, "postSegment")
			return nil
		},
		OnSegmentResolvedFunc: func(ctx game.SegmentResolutionContext) error {
			if ctx.Move != move {
				t.Fatalf("expected move pointer in segment resolution context")
			}
			calls = append(calls, "segmentResolved")
			return nil
		},
		OnCaptureFunc: func(ctx game.CaptureContext) error {
			if ctx.Move != move {
				t.Fatalf("expected move pointer in capture context")
			}
			calls = append(calls, "capture")
			return nil
		},
		ResolveCaptureFunc: func(ctx game.CaptureContext) (game.CaptureOutcome, error) {
			if ctx.Move != move {
				t.Fatalf("expected move pointer in capture resolution context")
			}
			calls = append(calls, "resolveCapture")
			return game.CaptureOutcome{StepAdjustment: -1, ForceTurnEnd: true}, nil
		},
		OnTurnEndFunc: func(ctx game.TurnEndContext) error {
			if ctx.Move != move {
				t.Fatalf("expected move pointer in turn end context")
			}
			calls = append(calls, "turnEnd")
			return nil
		},
		ResolveTurnEndFunc: func(ctx game.TurnEndContext) (game.TurnEndOutcome, error) {
			if ctx.Move != move {
				t.Fatalf("expected move pointer in turn end resolution context")
			}
			calls = append(calls, "resolveTurnEnd")
			outcome := game.TurnEndOutcome{}
			outcome.AddSlow(game.Black, 2)
			outcome.Notes = append(outcome.Notes, "slow note")
			return outcome, nil
		},
		PlanSpecialMoveFunc: func(ctx *game.SpecialMoveContext) (game.SpecialMovePlan, bool, error) {
			if ctx.Move != move {
				t.Fatalf("expected move pointer in special move context")
			}
			calls = append(calls, "planSpecialMove")
			return game.SpecialMovePlan{StepCost: 1, Note: "special move", Ability: game.AbilitySideStep, MarkAbilityUsed: true}, true, nil
		},
		FreeContinuationFunc: func(ctx game.FreeContinuationContext) bool {
			if ctx.Move != move {
				t.Fatalf("expected move pointer in free continuation context")
			}
			calls = append(calls, "freeContinuation")
			return true
		},
		OnDirectionChangeFunc: func(ctx game.DirectionChangeContext) bool {
			if ctx.Move != move {
				t.Fatalf("expected move pointer in direction change context")
			}
			calls = append(calls, "directionChange")
			return true
		},
	}

	if delta, err := hf.StepBudgetModifier(game.StepBudgetContext{Move: move}); err != nil {
		t.Fatalf("step budget modifier: %v", err)
	} else if delta.AddSteps != 1 || len(delta.Notes) != 1 {
		t.Fatalf("unexpected delta: %+v", delta)
	}

	if err := hf.OnMoveStart(game.MoveLifecycleContext{Move: move}); err != nil {
		t.Fatalf("move start: %v", err)
	}

	cost := 2
	budget := move.RemainingSteps
	if err := hf.PrepareSegment(&game.SegmentPreparationContext{Move: move, StepCost: &cost, StepBudget: &budget}); err != nil {
		t.Fatalf("prepare segment: %v", err)
	}
	if cost != 3 {
		t.Fatalf("expected step cost to increase to 3, got %d", cost)
	}
	if budget != move.RemainingSteps+2 {
		t.Fatalf("expected budget to increase by 2, got %d", budget)
	}
	move.RemainingSteps = budget

	if err := hf.OnSegmentStart(game.SegmentContext{Move: move}); err != nil {
		t.Fatalf("segment start: %v", err)
	}
	if err := hf.OnPostSegment(game.PostSegmentContext{Move: move}); err != nil {
		t.Fatalf("post segment: %v", err)
	}
	if err := hf.OnSegmentResolved(game.SegmentResolutionContext{Move: move, StepsConsumed: cost}); err != nil {
		t.Fatalf("segment resolved: %v", err)
	}
	if err := hf.OnCapture(game.CaptureContext{Move: move}); err != nil {
		t.Fatalf("capture: %v", err)
	}
	if outcome, err := hf.ResolveCapture(game.CaptureContext{Move: move}); err != nil {
		t.Fatalf("resolve capture: %v", err)
	} else if outcome.StepAdjustment != -1 || !outcome.ForceTurnEnd {
		t.Fatalf("unexpected capture outcome: %+v", outcome)
	}
	if err := hf.OnTurnEnd(game.TurnEndContext{Move: move}); err != nil {
		t.Fatalf("turn end: %v", err)
	}
	if outcome, err := hf.ResolveTurnEnd(game.TurnEndContext{Move: move}); err != nil {
		t.Fatalf("resolve turn end: %v", err)
	} else {
		if got := outcome.Slow[game.Black]; got != 2 {
			t.Fatalf("expected slow of 2 on black, got %d", got)
		}
		if len(outcome.Notes) != 1 || outcome.Notes[0] != "slow note" {
			t.Fatalf("unexpected turn end notes: %+v", outcome.Notes)
		}
	}
	if plan, handled, err := hf.PlanSpecialMove(&game.SpecialMoveContext{Move: move}); err != nil {
		t.Fatalf("plan special move: %v", err)
	} else {
		if !handled {
			t.Fatalf("expected special move to be handled")
		}
		if plan.StepCost != 1 || plan.Note != "special move" || !plan.MarkAbilityUsed || plan.Ability != game.AbilitySideStep {
			t.Fatalf("unexpected special move plan: %+v", plan)
		}
	}
	if !hf.FreeContinuationAvailable(game.FreeContinuationContext{Move: move}) {
		t.Fatalf("expected free continuation to be available")
	}
	if !hf.OnDirectionChange(game.DirectionChangeContext{Move: move}) {
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
		OnCaptureFunc: func(game.CaptureContext) error {
			captures++
			if captures > 2 {
				return errLimit
			}
			return nil
		},
	}

	ctx := game.CaptureContext{}
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
		ResolveTurnEndFunc: func(game.TurnEndContext) (game.TurnEndOutcome, error) {
			outcome := game.TurnEndOutcome{}
			outcome.AddSlow(game.Black, 3)
			outcome.Notes = append(outcome.Notes, "slow applied")
			return outcome, nil
		},
	}

	outcome, err := hf.ResolveTurnEnd(game.TurnEndContext{Move: &game.MoveState{}})
	if err != nil {
		t.Fatalf("resolve turn end: %v", err)
	}
	if got := outcome.Slow[game.Black]; got != 3 {
		t.Fatalf("expected slow of 3 on black, got %d", got)
	}
	if len(outcome.Notes) != 1 || outcome.Notes[0] != "slow applied" {
		t.Fatalf("unexpected notes: %+v", outcome.Notes)
	}
}
