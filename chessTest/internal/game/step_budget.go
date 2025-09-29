// path: chessTest/internal/game/step_budget.go
package game

func (e *Engine) baseStepBudget(pc *Piece) int {
	baseSteps := 1
	if pc == nil {
		return baseSteps
	}

	slowPenalty := e.temporalSlow[pc.Color.Index()]
	if slowPenalty > 0 {
		e.temporalSlow[pc.Color.Index()] = 0
		baseSteps -= slowPenalty
	}

	return baseSteps
}

func (e *Engine) calculateStepBudget(pc *Piece, handlers map[Ability][]AbilityHandler) (int, []string, error) {
	total := e.baseStepBudget(pc)

	sideHandlers, err := e.instantiateSideAbilityHandlers(pc, handlers)
	if err != nil {
		return 0, nil, err
	}

	if len(handlers) == 0 && len(sideHandlers) == 0 {
		return total, nil, nil
	}

	ctx := &e.abilityCtx.stepBudget
	*ctx = StepBudgetContext{Engine: e, Piece: pc}
	defer func() {
		e.abilityCtx.stepBudget = StepBudgetContext{}
	}()

	apply := func(handler AbilityHandler, notes *[]string) error {
		if handler == nil {
			return nil
		}
		delta, err := handler.StepBudgetModifier(*ctx)
		if err != nil {
			return err
		}
		total += delta.AddSteps
		if len(delta.Notes) > 0 {
			*notes = append(*notes, delta.Notes...)
		}
		return nil
	}

	var notes []string
	for _, handlerMap := range []map[Ability][]AbilityHandler{handlers, sideHandlers} {
		if len(handlerMap) == 0 {
			continue
		}
		for _, handlerList := range handlerMap {
			for _, handler := range handlerList {
				if err := apply(handler, &notes); err != nil {
					return 0, nil, err
				}
			}
		}
	}

	if total < 1 {
		total = 1
	}
	return total, notes, nil
}

func (e *Engine) calculateMovementCost(pc *Piece, from, to Square) int {
	cost := 1

	if e.currentMove != nil && e.wouldChangeDirection(e.currentMove, from, to) {
		cost++
	}

	return cost
}
