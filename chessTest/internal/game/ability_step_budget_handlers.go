// path: chessTest/internal/game/ability_step_budget_handlers.go
package game

// NewTailwindHandler constructs the default Tailwind ability handler.
func NewTailwindHandler() AbilityHandler { return tailwindHandler{} }

type tailwindHandler struct {
	abilityHandlerBase
}

func (tailwindHandler) StepBudgetModifier(ctx StepBudgetContext) (StepBudgetDelta, error) {
	if ctx.Engine == nil || ctx.Piece == nil {
		return StepBudgetDelta{}, nil
	}
	pc := ctx.Piece
	if !pc.HasAbility(AbilityTailwind) {
		return StepBudgetDelta{}, nil
	}
	if elementOf(ctx.Engine, pc) != ElementAir {
		return StepBudgetDelta{}, nil
	}

	delta := StepBudgetDelta{AddSteps: 2, Notes: []string{"Tailwind grants +2 steps"}}
	if pc.HasAbility(AbilityTemporalLock) {
		delta.AddSteps--
		delta.Notes = append(delta.Notes, "Temporal Lock dampens Tailwind (-1 step)")
	}
	return delta, nil
}

// NewRadiantVisionHandler constructs the default Radiant Vision ability handler.
func NewRadiantVisionHandler() AbilityHandler { return radiantVisionHandler{} }

type radiantVisionHandler struct {
	abilityHandlerBase
}

func (radiantVisionHandler) StepBudgetModifier(ctx StepBudgetContext) (StepBudgetDelta, error) {
	if ctx.Engine == nil || ctx.Piece == nil {
		return StepBudgetDelta{}, nil
	}
	pc := ctx.Piece
	if !pc.HasAbility(AbilityRadiantVision) {
		return StepBudgetDelta{}, nil
	}
	if elementOf(ctx.Engine, pc) != ElementLight {
		return StepBudgetDelta{}, nil
	}

	delta := StepBudgetDelta{AddSteps: 1, Notes: []string{"Radiant Vision grants +1 step"}}
	if pc.HasAbility(AbilityMistShroud) {
		delta.AddSteps++
		delta.Notes = append(delta.Notes, "Mist Shroud combo adds +1 step")
	}
	return delta, nil
}

// NewUmbralStepHandler constructs the default Umbral Step ability handler.
func NewUmbralStepHandler() AbilityHandler { return umbralStepHandler{} }

type umbralStepHandler struct {
	abilityHandlerBase
}

func (umbralStepHandler) StepBudgetModifier(ctx StepBudgetContext) (StepBudgetDelta, error) {
	if ctx.Engine == nil || ctx.Piece == nil {
		return StepBudgetDelta{}, nil
	}
	pc := ctx.Piece
	if !pc.HasAbility(AbilityUmbralStep) {
		return StepBudgetDelta{}, nil
	}
	if elementOf(ctx.Engine, pc) != ElementShadow {
		return StepBudgetDelta{}, nil
	}

	delta := StepBudgetDelta{AddSteps: 2, Notes: []string{"Umbral Step grants +2 steps"}}
	if pc.HasAbility(AbilityRadiantVision) {
		delta.AddSteps--
		delta.Notes = append(delta.Notes, "Radiant Vision reduces Umbral Step by 1")
	}
	return delta, nil
}

func (umbralStepHandler) CanPhase(ctx PhaseContext) (bool, error) {
	if ctx.Piece == nil {
		return false, nil
	}
	if !ctx.Piece.HasAbility(AbilityUmbralStep) {
		return false, nil
	}
	return true, nil
}

// NewSchrodingersLaughHandler constructs the default Schrödinger's Laugh ability handler.
func NewSchrodingersLaughHandler() AbilityHandler { return schrodingersLaughHandler{} }

type schrodingersLaughHandler struct {
	abilityHandlerBase
}

func (schrodingersLaughHandler) StepBudgetModifier(ctx StepBudgetContext) (StepBudgetDelta, error) {
	if ctx.Piece == nil {
		return StepBudgetDelta{}, nil
	}
	pc := ctx.Piece
	if !pc.HasAbility(AbilitySchrodingersLaugh) {
		return StepBudgetDelta{}, nil
	}

	delta := StepBudgetDelta{AddSteps: 2, Notes: []string{"Schrödinger's Laugh grants +2 steps"}}
	if pc.HasAbility(AbilitySideStep) {
		delta.AddSteps++
		delta.Notes = append(delta.Notes, "Side Step combo adds +1 step")
	}
	return delta, nil
}

// NewGaleLiftHandler constructs the default Gale Lift ability handler.
func NewGaleLiftHandler() AbilityHandler { return galeLiftHandler{} }

type galeLiftHandler struct {
	abilityHandlerBase
}

func (galeLiftHandler) CanPhase(ctx PhaseContext) (bool, error) {
	if ctx.Piece == nil {
		return false, nil
	}
	if !ctx.Piece.HasAbility(AbilityGaleLift) {
		return false, nil
	}
	return true, nil
}
