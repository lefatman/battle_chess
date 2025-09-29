// path: chessTest/internal/game/ability_dispatch.go
package game

import "errors"

func (e *Engine) resetAbilityHandlers() {
	if e.abilityHandlers == nil {
		return
	}
	for key := range e.abilityHandlers {
		delete(e.abilityHandlers, key)
	}
}

func (e *Engine) instantiateAbilityHandlers(pc *Piece) (map[Ability][]AbilityHandler, error) {
	e.resetAbilityHandlers()
	if pc == nil || len(pc.Abilities) == 0 {
		return nil, nil
	}

	var handlers map[Ability][]AbilityHandler
	for _, ability := range pc.Abilities {
		if ability == AbilityNone {
			continue
		}

		handler, err := resolveAbilityHandler(ability)
		if err != nil {
			if errors.Is(err, ErrAbilityNotRegistered) {
				continue
			}
			if errors.Is(err, ErrAbilityFactoryNotConfigured) {
				return nil, err
			}
			return nil, err
		}
		if handler == nil {
			continue
		}
		if handlers == nil {
			handlers = make(map[Ability][]AbilityHandler)
		}
		handlers[ability] = append(handlers[ability], handler)
	}

	ensureHandlers := func() {
		if handlers == nil {
			handlers = make(map[Ability][]AbilityHandler)
		}
	}

	if pc.Abilities.Contains(AbilityBlazeRush) && len(handlers[AbilityBlazeRush]) == 0 {
		ensureHandlers()
		handlers[AbilityBlazeRush] = append(handlers[AbilityBlazeRush], newBlazeRushFallbackHandler())
	}
	if pc.Abilities.Contains(AbilityFloodWake) && len(handlers[AbilityFloodWake]) == 0 {
		ensureHandlers()
		handlers[AbilityFloodWake] = append(handlers[AbilityFloodWake], newFloodWakeFallbackHandler())
	}
	if pc.Abilities.Contains(AbilitySideStep) && len(handlers[AbilitySideStep]) == 0 {
		ensureHandlers()
		handlers[AbilitySideStep] = append(handlers[AbilitySideStep], newSideStepFallbackHandler())
	}
	if pc.Abilities.Contains(AbilityQuantumStep) && len(handlers[AbilityQuantumStep]) == 0 {
		ensureHandlers()
		handlers[AbilityQuantumStep] = append(handlers[AbilityQuantumStep], newQuantumStepFallbackHandler())
	}
	if pc.Abilities.Contains(AbilityMistShroud) && len(handlers[AbilityMistShroud]) == 0 {
		ensureHandlers()
		handlers[AbilityMistShroud] = append(handlers[AbilityMistShroud], newMistShroudFallbackHandler())
	}
	if pc.Abilities.Contains(AbilityDoubleKill) && len(handlers[AbilityDoubleKill]) == 0 {
		ensureHandlers()
		handlers[AbilityDoubleKill] = append(handlers[AbilityDoubleKill], NewDoubleKillHandler())
	}
	if pc.Abilities.Contains(AbilityScorch) && len(handlers[AbilityScorch]) == 0 {
		ensureHandlers()
		handlers[AbilityScorch] = append(handlers[AbilityScorch], NewScorchHandler())
	}
	if pc.Abilities.Contains(AbilityTailwind) && len(handlers[AbilityTailwind]) == 0 {
		ensureHandlers()
		handlers[AbilityTailwind] = append(handlers[AbilityTailwind], NewTailwindHandler())
	}
	if pc.Abilities.Contains(AbilityRadiantVision) && len(handlers[AbilityRadiantVision]) == 0 {
		ensureHandlers()
		handlers[AbilityRadiantVision] = append(handlers[AbilityRadiantVision], NewRadiantVisionHandler())
	}
	if pc.Abilities.Contains(AbilityUmbralStep) && len(handlers[AbilityUmbralStep]) == 0 {
		ensureHandlers()
		handlers[AbilityUmbralStep] = append(handlers[AbilityUmbralStep], NewUmbralStepHandler())
	}
	if pc.Abilities.Contains(AbilityQuantumKill) && len(handlers[AbilityQuantumKill]) == 0 {
		ensureHandlers()
		handlers[AbilityQuantumKill] = append(handlers[AbilityQuantumKill], NewQuantumKillHandler())
	}
	if pc.Abilities.Contains(AbilityChainKill) && len(handlers[AbilityChainKill]) == 0 {
		ensureHandlers()
		handlers[AbilityChainKill] = append(handlers[AbilityChainKill], NewChainKillHandler())
	}
	if pc.Abilities.Contains(AbilityGaleLift) && len(handlers[AbilityGaleLift]) == 0 {
		ensureHandlers()
		handlers[AbilityGaleLift] = append(handlers[AbilityGaleLift], NewGaleLiftHandler())
	}
	if pc.Abilities.Contains(AbilityPoisonousMeat) && len(handlers[AbilityPoisonousMeat]) == 0 {
		ensureHandlers()
		handlers[AbilityPoisonousMeat] = append(handlers[AbilityPoisonousMeat], NewPoisonousMeatHandler())
	}
	if pc.Abilities.Contains(AbilityOverload) && len(handlers[AbilityOverload]) == 0 {
		ensureHandlers()
		handlers[AbilityOverload] = append(handlers[AbilityOverload], NewOverloadHandler())
	}
	if pc.Abilities.Contains(AbilityBastion) && len(handlers[AbilityBastion]) == 0 {
		ensureHandlers()
		handlers[AbilityBastion] = append(handlers[AbilityBastion], NewBastionHandler())
	}
	if pc.Abilities.Contains(AbilitySchrodingersLaugh) && len(handlers[AbilitySchrodingersLaugh]) == 0 {
		ensureHandlers()
		handlers[AbilitySchrodingersLaugh] = append(handlers[AbilitySchrodingersLaugh], NewSchrodingersLaughHandler())
	}
	if pc.Abilities.Contains(AbilityTemporalLock) && len(handlers[AbilityTemporalLock]) == 0 {
		ensureHandlers()
		handlers[AbilityTemporalLock] = append(handlers[AbilityTemporalLock], NewTemporalLockHandler())
	}
	if pc.Abilities.Contains(AbilityResurrection) && len(handlers[AbilityResurrection]) == 0 {
		ensureHandlers()
		handlers[AbilityResurrection] = append(handlers[AbilityResurrection], NewResurrectionHandler())
	}

	if len(handlers) == 0 {
		e.abilityHandlers = nil
		return nil, nil
	}

	e.abilityHandlers = handlers
	return handlers, nil
}

func (e *Engine) instantiateSideAbilityHandlers(pc *Piece, existing map[Ability][]AbilityHandler) (map[Ability][]AbilityHandler, error) {
	if pc == nil {
		return nil, nil
	}

	abilities := e.abilities[pc.Color.Index()]
	if len(abilities) == 0 {
		return nil, nil
	}

	var handlers map[Ability][]AbilityHandler
	for _, ability := range abilities {
		if ability == AbilityNone {
			continue
		}
		if existing != nil && len(existing[ability]) > 0 {
			continue
		}
		handler, err := resolveAbilityHandler(ability)
		if err != nil {
			if errors.Is(err, ErrAbilityNotRegistered) {
				continue
			}
			return nil, err
		}
		if handler == nil {
			continue
		}
		if handlers == nil {
			handlers = make(map[Ability][]AbilityHandler)
		}
		handlers[ability] = append(handlers[ability], handler)
	}
	return handlers, nil
}

func (e *Engine) activeHandlers() map[Ability][]AbilityHandler {
	if e.currentMove != nil && len(e.currentMove.Handlers) > 0 {
		return e.currentMove.Handlers
	}
	return e.abilityHandlers
}

func (e *Engine) handlersForAbility(id Ability) []AbilityHandler {
	if e.currentMove != nil {
		if handlers := e.currentMove.handlersFor(id); len(handlers) > 0 {
			return handlers
		}
	}
	if e.abilityHandlers != nil {
		return e.abilityHandlers[id]
	}
	return nil
}

func (e *Engine) forEachHandler(handlerMap map[Ability][]AbilityHandler, fn func(Ability, AbilityHandler) error) error {
	if len(handlerMap) == 0 {
		return nil
	}
	for ability, handlers := range handlerMap {
		for _, handler := range handlers {
			if handler == nil {
				continue
			}
			if err := fn(ability, handler); err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *Engine) forEachActiveHandler(fn func(Ability, AbilityHandler) error) error {
	return e.forEachHandler(e.activeHandlers(), fn)
}

func (e *Engine) dispatchMoveStartHandlers(req MoveRequest, meta SegmentMetadata) error {
	if e.currentMove == nil {
		return nil
	}
	ctx := &e.abilityCtx.move
	*ctx = MoveLifecycleContext{
		Engine:  e,
		Move:    e.currentMove,
		Request: req,
		Segment: meta,
	}
	defer func() {
		e.abilityCtx.move = MoveLifecycleContext{}
	}()
	return e.forEachActiveHandler(func(_ Ability, handler AbilityHandler) error {
		return handler.OnMoveStart(*ctx)
	})
}

func (e *Engine) dispatchSegmentPreparationHandlers(from, to Square, meta SegmentMetadata, step int, stepCost *int) error {
	if e.currentMove == nil {
		return nil
	}
	budget := e.currentMove.RemainingSteps
	ctx := &e.abilityCtx.segmentPrep
	*ctx = SegmentPreparationContext{
		Engine:      e,
		Move:        e.currentMove,
		From:        from,
		To:          to,
		Segment:     meta,
		SegmentStep: step,
		StepCost:    stepCost,
		StepBudget:  &budget,
	}
	defer func() {
		e.abilityCtx.segmentPrep = SegmentPreparationContext{}
	}()
	if err := e.forEachActiveHandler(func(_ Ability, handler AbilityHandler) error {
		preparer, ok := handler.(SegmentPreparationHandler)
		if !ok {
			return nil
		}
		return preparer.PrepareSegment(ctx)
	}); err != nil {
		return err
	}
	if ctx.StepBudget != nil {
		e.currentMove.RemainingSteps = *ctx.StepBudget
	} else {
		e.currentMove.RemainingSteps = budget
	}
	return nil
}

func (e *Engine) dispatchSegmentStartHandlers(from, to Square, meta SegmentMetadata, step int) error {
	if e.currentMove == nil {
		return nil
	}
	ctx := &e.abilityCtx.segment
	*ctx = SegmentContext{
		Engine:      e,
		Move:        e.currentMove,
		From:        from,
		To:          to,
		Segment:     meta,
		SegmentStep: step,
	}
	defer func() {
		e.abilityCtx.segment = SegmentContext{}
	}()
	return e.forEachActiveHandler(func(_ Ability, handler AbilityHandler) error {
		return handler.OnSegmentStart(*ctx)
	})
}

func (e *Engine) dispatchSegmentResolvedHandlers(from, to Square, meta SegmentMetadata, step int, cost int) error {
	if e.currentMove == nil {
		return nil
	}
	ctx := &e.abilityCtx.segmentResolved
	*ctx = SegmentResolutionContext{
		Engine:        e,
		Move:          e.currentMove,
		From:          from,
		To:            to,
		Segment:       meta,
		SegmentStep:   step,
		StepsConsumed: cost,
	}
	defer func() {
		e.abilityCtx.segmentResolved = SegmentResolutionContext{}
	}()
	return e.forEachActiveHandler(func(_ Ability, handler AbilityHandler) error {
		resolver, ok := handler.(SegmentResolutionHandler)
		if !ok {
			return nil
		}
		return resolver.OnSegmentResolved(*ctx)
	})
}

func (e *Engine) dispatchPostSegmentHandlers(pc *Piece, from, to Square, meta SegmentMetadata, step int) error {
	if e.currentMove == nil {
		return nil
	}
	ctx := &e.abilityCtx.postSegment
	*ctx = PostSegmentContext{
		Engine:      e,
		Move:        e.currentMove,
		Piece:       pc,
		From:        from,
		To:          to,
		Segment:     meta,
		SegmentStep: step,
	}
	defer func() {
		e.abilityCtx.postSegment = PostSegmentContext{}
	}()
	return e.forEachActiveHandler(func(_ Ability, handler AbilityHandler) error {
		return handler.OnPostSegment(*ctx)
	})
}

func (e *Engine) dispatchDirectionChangeHandlers(pc *Piece, prevStart, prevEnd, currentEnd Square, prevDir, currentDir Direction, step int) bool {
	if e.currentMove == nil {
		return false
	}
	ctx := &e.abilityCtx.direction
	*ctx = DirectionChangeContext{
		Engine:            e,
		Move:              e.currentMove,
		Piece:             pc,
		PreviousStart:     prevStart,
		PreviousEnd:       prevEnd,
		CurrentEnd:        currentEnd,
		PreviousDirection: prevDir,
		CurrentDirection:  currentDir,
		SegmentStep:       step,
	}
	defer func() {
		e.abilityCtx.direction = DirectionChangeContext{}
	}()
	handled := false
	_ = e.forEachActiveHandler(func(_ Ability, handler AbilityHandler) error {
		watcher, ok := handler.(DirectionChangeHandler)
		if !ok {
			return nil
		}
		if watcher.OnDirectionChange(*ctx) {
			handled = true
		}
		return nil
	})
	return handled
}

func (e *Engine) dispatchFreeContinuationHandlers(id Ability, pc *Piece) bool {
	handlers := e.handlersForAbility(id)
	if len(handlers) == 0 {
		return false
	}
	ctx := &e.abilityCtx.continuation
	*ctx = FreeContinuationContext{
		Engine:  e,
		Move:    e.currentMove,
		Piece:   pc,
		Ability: id,
	}
	defer func() {
		e.abilityCtx.continuation = FreeContinuationContext{}
	}()
	for _, handler := range handlers {
		grantor, ok := handler.(FreeContinuationHandler)
		if !ok {
			continue
		}
		if grantor.FreeContinuationAvailable(*ctx) {
			return true
		}
	}
	return false
}

func (e *Engine) dispatchCaptureHandlers(attacker, victim *Piece, square Square, step int) error {
	if e.currentMove == nil {
		return nil
	}
	ctx := &e.abilityCtx.capture
	*ctx = CaptureContext{
		Engine:        e,
		Move:          e.currentMove,
		Attacker:      attacker,
		Victim:        victim,
		CaptureSquare: square,
		SegmentStep:   step,
	}
	defer func() {
		e.abilityCtx.capture = CaptureContext{}
	}()
	return e.forEachActiveHandler(func(_ Ability, handler AbilityHandler) error {
		return handler.OnCapture(*ctx)
	})
}

func (e *Engine) dispatchTurnEndHandlers(reason TurnEndReason) (TurnEndOutcome, error) {
	if e.currentMove == nil {
		return TurnEndOutcome{}, nil
	}
	ctx := &e.abilityCtx.turnEnd
	*ctx = TurnEndContext{
		Engine: e,
		Move:   e.currentMove,
		Reason: reason,
	}
	defer func() {
		e.abilityCtx.turnEnd = TurnEndContext{}
	}()

	outcome := TurnEndOutcome{}
	err := e.forEachActiveHandler(func(_ Ability, handler AbilityHandler) error {
		if err := handler.OnTurnEnd(*ctx); err != nil {
			return err
		}
		resolver, ok := handler.(TurnEndResolutionHandler)
		if !ok {
			return nil
		}
		result, err := resolver.ResolveTurnEnd(*ctx)
		if err != nil {
			return err
		}
		outcome = outcome.Merge(result)
		return nil
	})
	return outcome, err
}

func (e *Engine) applyTurnEndOutcome(outcome TurnEndOutcome) {
	if len(outcome.Slow) > 0 {
		for color, amount := range outcome.Slow {
			if amount <= 0 {
				continue
			}
			e.temporalSlow[color.Index()] = amount
		}
	}
	for _, note := range outcome.Notes {
		if note == "" {
			continue
		}
		appendAbilityNote(&e.board.lastNote, note)
	}
}
