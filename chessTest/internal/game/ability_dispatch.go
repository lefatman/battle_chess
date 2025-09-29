// path: chessTest/internal/game/ability_dispatch.go
package game

import (
	"errors"
	"sync"
)

type abilityHandlerTable struct {
	handlers [AbilityCount][]AbilityHandler
	total    int
}

func (t *abilityHandlerTable) reset() {
	if t == nil {
		return
	}
	for i := range t.handlers {
		t.handlers[i] = t.handlers[i][:0]
	}
	t.total = 0
}

func (t *abilityHandlerTable) append(id Ability, handler AbilityHandler) {
	if t == nil || handler == nil {
		return
	}
	idx := abilityIndex(id)
	if idx < 0 {
		return
	}
	t.handlers[idx] = append(t.handlers[idx], handler)
	t.total++
}

func (t *abilityHandlerTable) appendAll(id Ability, src []AbilityHandler) {
	if t == nil || len(src) == 0 {
		return
	}
	idx := abilityIndex(id)
	if idx < 0 {
		return
	}
	t.handlers[idx] = append(t.handlers[idx], src...)
	t.total += len(src)
}

func (t *abilityHandlerTable) handlersFor(id Ability) []AbilityHandler {
	if t == nil {
		return nil
	}
	idx := abilityIndex(id)
	if idx < 0 {
		return nil
	}
	return t.handlers[idx]
}

func (t *abilityHandlerTable) has(id Ability) bool {
	if t == nil {
		return false
	}
	idx := abilityIndex(id)
	if idx < 0 {
		return false
	}
	return len(t.handlers[idx]) > 0
}

func (t *abilityHandlerTable) empty() bool { return t == nil || t.total == 0 }

func (t *abilityHandlerTable) forEach(fn func(Ability, AbilityHandler) error) error {
	if t == nil || t.total == 0 {
		return nil
	}
	for idx := 0; idx < AbilityCount; idx++ {
		ability := Ability(idx)
		for _, handler := range t.handlers[idx] {
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

var abilityHandlerTablePool = sync.Pool{
	New: func() any { return &abilityHandlerTable{} },
}

func borrowAbilityHandlers() *abilityHandlerTable {
	table := abilityHandlerTablePool.Get().(*abilityHandlerTable)
	table.reset()
	return table
}

func releaseAbilityHandlers(table *abilityHandlerTable) {
	if table == nil {
		return
	}
	table.reset()
	abilityHandlerTablePool.Put(table)
}

func (e *Engine) resetAbilityHandlers() {
	if e.abilityHandlers == nil {
		return
	}
	releaseAbilityHandlers(e.abilityHandlers)
	e.abilityHandlers = nil
}

func (e *Engine) instantiateAbilityHandlers(pc *Piece) (*abilityHandlerTable, error) {
	e.resetAbilityHandlers()
	if pc == nil || pc.AbilityMask.Empty() {
		return nil, nil
	}

	var table *abilityHandlerTable
	seen := AbilitySet(0)
	for _, ability := range pc.Abilities {
		if ability == AbilityNone {
			continue
		}
		if seen.Has(ability) {
			continue
		}
		seen = seen.With(ability)

		handler, err := resolveAbilityHandler(ability)
		switch {
		case err == nil:
			if handler != nil {
				if table == nil {
					table = borrowAbilityHandlers()
				}
				table.append(ability, handler)
			}
		case errors.Is(err, ErrAbilityNotRegistered):
			// rely on fallback registry
		case errors.Is(err, ErrAbilityFactoryNotConfigured):
			if table != nil {
				releaseAbilityHandlers(table)
			}
			return nil, err
		default:
			if table != nil {
				releaseAbilityHandlers(table)
			}
			return nil, err
		}

		if table == nil || !table.has(ability) {
			if fallback := fallbackHandlersFor(ability); len(fallback) > 0 {
				if table == nil {
					table = borrowAbilityHandlers()
				}
				table.appendAll(ability, fallback)
			}
		}
	}

	if table == nil || table.total == 0 {
		if table != nil {
			releaseAbilityHandlers(table)
		}
		e.abilityHandlers = nil
		return nil, nil
	}

	e.abilityHandlers = table
	return table, nil
}

func (e *Engine) instantiateSideAbilityHandlers(pc *Piece, existing *abilityHandlerTable) (*abilityHandlerTable, error) {
	if pc == nil {
		return nil, nil
	}

	mask := e.abilityMasks[pc.Color.Index()]
	if mask.Empty() {
		return nil, nil
	}

	var table *abilityHandlerTable
	seen := AbilitySet(0)
	for _, ability := range e.abilities[pc.Color.Index()] {
		if ability == AbilityNone {
			continue
		}
		if !mask.Has(ability) || seen.Has(ability) {
			continue
		}
		seen = seen.With(ability)
		if existing != nil && existing.has(ability) {
			continue
		}

		handler, err := resolveAbilityHandler(ability)
		switch {
		case err == nil:
			if handler != nil {
				if table == nil {
					table = borrowAbilityHandlers()
				}
				table.append(ability, handler)
			}
		case errors.Is(err, ErrAbilityNotRegistered):
			continue
		case errors.Is(err, ErrAbilityFactoryNotConfigured):
			if table != nil {
				releaseAbilityHandlers(table)
			}
			return nil, err
		default:
			if table != nil {
				releaseAbilityHandlers(table)
			}
			return nil, err
		}
	}

	if table == nil || table.total == 0 {
		if table != nil {
			releaseAbilityHandlers(table)
		}
		return nil, nil
	}

	return table, nil
}

func (e *Engine) activeHandlers() *abilityHandlerTable {
	if e.currentMove != nil && e.currentMove.Handlers != nil && !e.currentMove.Handlers.empty() {
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
		return e.abilityHandlers.handlersFor(id)
	}
	return nil
}

func (e *Engine) forEachHandler(table *abilityHandlerTable, fn func(Ability, AbilityHandler) error) error {
	if table == nil {
		return nil
	}
	return table.forEach(fn)
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
