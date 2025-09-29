package game

import (
	"errors"
	"fmt"

	"battle_chess_poc/internal/shared"
)

// MoveState tracks the current move in progress with a step budget system.
// It holds all the temporary state for a piece's actions within a single turn.
type MoveState struct {
	Piece               *Piece
	RemainingSteps      int
	Path                []Square
	Captures            []*Piece
	AbilityData         map[Ability]*AbilityRuntime
	TurnEnded           bool
	LastSegmentCaptured bool
	Promotion           PieceType
	PromotionSet        bool
	Handlers            map[Ability][]AbilityHandler
}

// AbilityRuntime captures mutable per-ability state for the duration of a move.
// Handlers can store arbitrary flags and counters keyed by semantic labels.
type AbilityRuntime struct {
	Flags    map[string]bool
	Counters map[string]int
}

// Clone produces a deep copy of the runtime data so history rewinds can
// restore the per-ability state safely.
func (ar *AbilityRuntime) Clone() *AbilityRuntime {
	if ar == nil {
		return nil
	}
	clone := &AbilityRuntime{}
	if len(ar.Flags) > 0 {
		clone.Flags = make(map[string]bool, len(ar.Flags))
		for k, v := range ar.Flags {
			clone.Flags[k] = v
		}
	}
	if len(ar.Counters) > 0 {
		clone.Counters = make(map[string]int, len(ar.Counters))
		for k, v := range ar.Counters {
			clone.Counters[k] = v
		}
	}
	return clone
}

func (ar *AbilityRuntime) setFlag(key string, value bool) {
	if ar == nil {
		return
	}
	if ar.Flags == nil {
		ar.Flags = make(map[string]bool)
	}
	ar.Flags[key] = value
}

func (ar *AbilityRuntime) flag(key string) bool {
	if ar == nil || len(ar.Flags) == 0 {
		return false
	}
	return ar.Flags[key]
}

func (ar *AbilityRuntime) setCounter(key string, value int) {
	if ar == nil {
		return
	}
	if ar.Counters == nil {
		ar.Counters = make(map[string]int)
	}
	ar.Counters[key] = value
}

func (ar *AbilityRuntime) addCounter(key string, delta int) int {
	if ar == nil {
		return 0
	}
	if ar.Counters == nil {
		ar.Counters = make(map[string]int)
	}
	ar.Counters[key] += delta
	return ar.Counters[key]
}

func (ar *AbilityRuntime) counter(key string) int {
	if ar == nil || len(ar.Counters) == 0 {
		return 0
	}
	return ar.Counters[key]
}

const (
	abilityFlagUsed         = "used"
	abilityFlagWindow       = "window"
	abilityFlagCaptureExtra = "captureExtra"

	abilityCounterFree             = "free"
	abilityCounterCaptures         = "captures"
	abilityCounterCaptureLimit     = "captureLimit"
	abilityCounterCaptureSquare    = "captureSquare"
	abilityCounterCaptureSegment   = "captureSegment"
	abilityCounterCaptureEnPassant = "captureEnPassant"
	abilityCounterResurrectionHold = "resurrectionHold"
)

func (ms *MoveState) ensureAbilityData() {
	if ms.AbilityData == nil {
		ms.AbilityData = make(map[Ability]*AbilityRuntime)
	}
}

func (ms *MoveState) abilityRuntime(id Ability) *AbilityRuntime {
	if ms == nil {
		return nil
	}
	ms.ensureAbilityData()
	rt, ok := ms.AbilityData[id]
	if !ok {
		rt = &AbilityRuntime{}
		ms.AbilityData[id] = rt
	}
	return rt
}

func (ms *MoveState) abilityFlag(id Ability, key string) bool {
	if ms == nil || len(ms.AbilityData) == 0 {
		return false
	}
	if rt, ok := ms.AbilityData[id]; ok {
		return rt.flag(key)
	}
	return false
}

func (ms *MoveState) setAbilityFlag(id Ability, key string, value bool) {
	if ms == nil {
		return
	}
	ms.abilityRuntime(id).setFlag(key, value)
}

func (ms *MoveState) abilityUsed(id Ability) bool {
	return ms.abilityFlag(id, abilityFlagUsed)
}

func (ms *MoveState) markAbilityUsed(id Ability) {
	ms.setAbilityFlag(id, abilityFlagUsed, true)
}

func (ms *MoveState) clearAbilityUsed(id Ability) {
	ms.setAbilityFlag(id, abilityFlagUsed, false)
}

func (ms *MoveState) abilityCounter(id Ability, key string) int {
	if ms == nil || len(ms.AbilityData) == 0 {
		return 0
	}
	if rt, ok := ms.AbilityData[id]; ok {
		return rt.counter(key)
	}
	return 0
}

func (ms *MoveState) setAbilityCounter(id Ability, key string, value int) {
	if ms == nil {
		return
	}
	ms.abilityRuntime(id).setCounter(key, value)
}

func (ms *MoveState) addAbilityCounter(id Ability, key string, delta int) int {
	if ms == nil {
		return 0
	}
	return ms.abilityRuntime(id).addCounter(key, delta)
}

func (ms *MoveState) handlersFor(id Ability) []AbilityHandler {
	if ms == nil || len(ms.Handlers) == 0 {
		return nil
	}
	return ms.Handlers[id]
}

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
	if pc.Abilities.Contains(AbilityQuantumKill) && len(handlers[AbilityQuantumKill]) == 0 {
		ensureHandlers()
		handlers[AbilityQuantumKill] = append(handlers[AbilityQuantumKill], NewQuantumKillHandler())
	}
	if pc.Abilities.Contains(AbilityChainKill) && len(handlers[AbilityChainKill]) == 0 {
		ensureHandlers()
		handlers[AbilityChainKill] = append(handlers[AbilityChainKill], NewChainKillHandler())
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
		dc, ok := handler.(DirectionChangeHandler)
		if !ok {
			return nil
		}
		if dc.OnDirectionChange(*ctx) {
			handled = true
		}
		return nil
	})
	return handled
}

func (e *Engine) dispatchFreeContinuationHandlers(id Ability, pc *Piece) bool {
	if len(e.handlersForAbility(id)) == 0 {
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
	available := false
	for _, handler := range e.handlersForAbility(id) {
		if handler == nil {
			continue
		}
		fc, ok := handler.(FreeContinuationHandler)
		if !ok {
			continue
		}
		if fc.FreeContinuationAvailable(*ctx) {
			available = true
		}
	}
	return available
}

func (e *Engine) dispatchCaptureHandlers(attacker, victim *Piece, square Square, step int) error {
	if e.currentMove == nil || victim == nil {
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

func newAbilityRuntimeMap(abilities AbilityList) map[Ability]*AbilityRuntime {
	if len(abilities) == 0 {
		return nil
	}
	runtimes := make(map[Ability]*AbilityRuntime)
	for _, ability := range abilities {
		if ability == AbilityNone {
			continue
		}
		if _, exists := runtimes[ability]; !exists {
			runtimes[ability] = &AbilityRuntime{}
		}
	}
	if len(runtimes) == 0 {
		return nil
	}
	return runtimes
}

func (ms *MoveState) captureCount() int {
	if ms == nil {
		return 0
	}
	captures := ms.abilityCounter(AbilityNone, abilityCounterCaptures)
	if captures == 0 && len(ms.Captures) > 0 {
		return len(ms.Captures)
	}
	return captures
}

func (ms *MoveState) captureLimit() int {
	if ms == nil {
		return 0
	}
	return ms.abilityCounter(AbilityNone, abilityCounterCaptureLimit)
}

func (ms *MoveState) setCaptureLimit(limit int) {
	if ms == nil {
		return
	}
	if limit < 0 {
		limit = 0
	}
	ms.setAbilityCounter(AbilityNone, abilityCounterCaptureLimit, limit)
}

func (ms *MoveState) increaseCaptureLimit(delta int) {
	if ms == nil || delta == 0 {
		return
	}
	current := ms.captureLimit()
	ms.setCaptureLimit(current + delta)
}

func (ms *MoveState) extraRemovalConsumed() bool {
	if ms == nil {
		return false
	}
	return ms.abilityFlag(AbilityNone, abilityFlagCaptureExtra)
}

func (ms *MoveState) resetExtraRemoval() {
	if ms == nil {
		return
	}
	ms.setAbilityFlag(AbilityNone, abilityFlagCaptureExtra, false)
}

func (ms *MoveState) markExtraRemovalConsumed() {
	if ms == nil {
		return
	}
	ms.setAbilityFlag(AbilityNone, abilityFlagCaptureExtra, true)
}

func (ms *MoveState) canCaptureMore() bool {
	if ms == nil {
		return false
	}

	captures := ms.captureCount()
	if captures == 0 {
		return true
	}

	limit := ms.captureLimit()
	if limit <= 0 {
		return true
	}
	return captures < limit
}

func (ms *MoveState) registerCapture(meta SegmentMetadata) {
	if meta.Capture == nil {
		return
	}

	ms.Captures = append(ms.Captures, meta.Capture)

	ms.addAbilityCounter(AbilityNone, abilityCounterCaptures, 1)

	step := len(ms.Path)
	if step >= 2 {
		step -= 2
	} else {
		step = 0
	}
	ms.setAbilityCounter(AbilityNone, abilityCounterCaptureSegment, step)
	ms.setAbilityCounter(AbilityNone, abilityCounterCaptureSquare, int(meta.CaptureSquare))
	ms.setAbilityCounter(AbilityNone, abilityCounterCaptureEnPassant, boolToInt(meta.EnPassant))
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func (e *Engine) checkPostCaptureTermination(pc *Piece, target *Piece) bool {
	if e.currentMove == nil {
		return false
	}
	return e.currentMove.TurnEnded
}

// ---------------------------
// Move Lifecycle Management
// ---------------------------

// startNewMove begins a new move, creating and initializing a MoveState.
// It calculates the initial step budget and executes the first segment of the move.
func (e *Engine) startNewMove(req MoveRequest) error {
	from, to := req.From, req.To
	pc := e.board.pieceAt[from]
	if pc == nil {
		return errors.New("no piece at source square")
	}
	if pc.Color != e.board.turn {
		return errors.New("not your turn")
	}

	handlers, err := e.instantiateAbilityHandlers(pc)
	if err != nil {
		return err
	}

	target := e.board.pieceAt[to]
	if target != nil && target.Color == pc.Color {
		return errors.New("cannot capture a friendly piece")
	}

	segmentCtx := moveSegmentContext{capture: target, captureSquare: to}
	if target == nil && pc.Type == Pawn && from.File() != to.File() {
		if sq, ok := e.board.EnPassant.Square(); ok && sq == to {
			captureRank := to.Rank()
			if pc.Color == White {
				captureRank--
			} else {
				captureRank++
			}
			captureSq, ok := shared.SquareFromCoords(captureRank, to.File())
			if !ok {
				return errors.New("invalid en passant capture")
			}
			victim := e.board.pieceAt[captureSq]
			if victim == nil || victim.Color == pc.Color || victim.Type != Pawn {
				return errors.New("invalid en passant capture")
			}
			segmentCtx.capture = victim
			segmentCtx.captureSquare = captureSq
			segmentCtx.enPassant = true
		} else {
			return errors.New("illegal pawn capture")
		}
	}

	// Validate the legality of the first move segment.
	if !e.isLegalFirstSegment(pc, from, to) {
		return errors.New("illegal first move")
	}
	if err := e.requireBlockPathDirection(pc, req.Dir); err != nil {
		return err
	}
	if blocked, note := e.captureBlockedByBlockPath(pc, from, target, to); blocked {
		appendAbilityNote(&e.board.lastNote, note)
		return ErrCaptureBlocked
	}

	// Calculate the total step budget for this turn.
	totalSteps, notes, err := e.calculateStepBudget(pc, handlers)
	if err != nil {
		return err
	}
	firstSegmentCost := e.calculateMovementCost(pc, from, to)
	remainingSteps := totalSteps - firstSegmentCost
	if remainingSteps < 0 {
		remainingSteps = 0
	}

	e.currentMove = &MoveState{
		Piece:          pc,
		RemainingSteps: remainingSteps,
		Path:           []Square{from},
		Captures:       []*Piece{},
		AbilityData:    newAbilityRuntimeMap(pc.Abilities),
		Promotion:      req.Promotion,
		PromotionSet:   req.HasPromotion,
		Handlers:       handlers,
	}

	e.currentMove.setCaptureLimit(1)
	e.currentMove.setAbilityCounter(AbilityNone, abilityCounterCaptures, len(e.currentMove.Captures))
	e.currentMove.setAbilityCounter(AbilityNone, abilityCounterCaptureSegment, -1)
	e.currentMove.setAbilityCounter(AbilityNone, abilityCounterCaptureSquare, -1)
	e.currentMove.setAbilityCounter(AbilityNone, abilityCounterCaptureEnPassant, 0)

	for _, note := range notes {
		if note == "" {
			continue
		}
		appendAbilityNote(&e.board.lastNote, note)
	}

	// The move is now valid, push the state before executing.
	delta := e.pushHistory()
	defer e.finalizeHistory(delta)

	e.recordSquareForUndo(from)
	e.recordSquareForUndo(to)
	if segmentCtx.capture != nil {
		e.recordSquareForUndo(segmentCtx.captureSquare)
	}
	segmentStep := len(e.currentMove.Path) - 1
	if segmentStep < 0 {
		segmentStep = 0
	}
	if err := e.dispatchMoveStartHandlers(req, segmentCtx.metadata()); err != nil {
		e.currentMove = nil
		e.abilityCtx.clear()
		return err
	}
	if err := e.dispatchSegmentStartHandlers(from, to, segmentCtx.metadata(), segmentStep); err != nil {
		e.currentMove = nil
		e.abilityCtx.clear()
		return err
	}
	e.executeMoveSegment(from, to, segmentCtx)
	e.currentMove.Path = append(e.currentMove.Path, to)
	if err := e.handlePostSegment(pc, from, to, segmentCtx.metadata()); err != nil {
		e.currentMove = nil
		e.abilityCtx.clear()
		return err
	}

	// Handle capture abilities if a piece was taken.
	if segmentCtx.capture != nil {
		e.currentMove.registerCapture(segmentCtx.metadata())
		if err := e.dispatchCaptureHandlers(pc, segmentCtx.capture, segmentCtx.captureSquare, segmentStep); err != nil {
			e.currentMove = nil
			e.abilityCtx.clear()
			return err
		}
		outcome, err := e.ResolveCaptureAbility(pc, segmentCtx.capture, segmentCtx.captureSquare, segmentStep)
		if err != nil {
			// If DoOver was triggered, the state is already rewound. Abort.
			e.currentMove = nil // Clear the invalid move state
			e.abilityCtx.clear()
			return err
		}
		if delta := outcome.StepAdjustment; delta != 0 {
			e.currentMove.RemainingSteps += delta
			if e.currentMove.RemainingSteps < 0 {
				e.currentMove.RemainingSteps = 0
			}
		}
		if !e.currentMove.canCaptureMore() {
			e.currentMove.TurnEnded = true
			e.endTurn(TurnEndForced)
			return nil
		}
		if outcome.ForceTurnEnd {
			e.currentMove.TurnEnded = true
			e.endTurn(TurnEndForced)
			return nil
		}
	}

	// Resolve post-move state changes.
	e.resolvePromotion(pc)
	if note := e.resolveBlockPathFacing(pc, req.Dir); note != "" {
		appendAbilityNote(&e.board.lastNote, note)
	}
	e.checkPostMoveAbilities(pc)
	e.checkPostMoveAbilities(pc)

	// Check if the turn should end naturally.
	if e.currentMove.TurnEnded {
		e.endTurn(TurnEndForced)
	} else if e.currentMove.RemainingSteps <= 0 && !e.hasFreeContinuation(pc) {
		e.endTurn(TurnEndNatural)
	} else {
		appendAbilityNote(&e.board.lastNote, fmt.Sprintf("%d steps remaining", e.currentMove.RemainingSteps))
	}

	return nil
}

// continueMove handles subsequent actions within a single turn, using the existing MoveState.
func (e *Engine) continueMove(req MoveRequest) error {
	if e.currentMove == nil || e.currentMove.TurnEnded {
		return errors.New("no active move to continue")
	}

	pc := e.currentMove.Piece
	from, to := req.From, req.To
	if from != pc.Square {
		return errors.New("must continue move from the piece's current square")
	}

	if req.HasPromotion {
		e.currentMove.Promotion = req.Promotion
		e.currentMove.PromotionSet = true
	}

	if handled, err := e.trySideStepNudge(pc, from, to); handled {
		return err
	}

	if handled, err := e.tryQuantumStep(pc, from, to); handled {
		return err
	}

	// Validate the legality of the continuation move.
	if !e.isLegalContinuation(pc, from, to) {
		return errors.New("illegal move continuation")
	}

	stepsNeeded := e.calculateMovementCost(pc, from, to)
	target := e.board.pieceAt[to]
	if target != nil && target.Color == pc.Color {
		return errors.New("cannot capture a friendly piece")
	}
	if blocked, note := e.captureBlockedByBlockPath(pc, from, target, to); blocked {
		appendAbilityNote(&e.board.lastNote, note)
		return ErrCaptureBlocked
	}

	segmentCtx := moveSegmentContext{capture: target, captureSquare: to}
	if target == nil && pc.Type == Pawn && from.File() != to.File() {
		if sq, ok := e.board.EnPassant.Square(); ok && sq == to {
			captureRank := to.Rank()
			if pc.Color == White {
				captureRank--
			} else {
				captureRank++
			}
			captureSq, ok := shared.SquareFromCoords(captureRank, to.File())
			if !ok {
				return errors.New("invalid en passant capture")
			}
			victim := e.board.pieceAt[captureSq]
			if victim == nil || victim.Color == pc.Color || victim.Type != Pawn {
				return errors.New("invalid en passant capture")
			}
			segmentCtx.capture = victim
			segmentCtx.captureSquare = captureSq
			segmentCtx.enPassant = true
		} else {
			return errors.New("illegal pawn capture")
		}
	}

	segmentStep := len(e.currentMove.Path) - 1
	if segmentStep < 0 {
		segmentStep = 0
	}

	meta := segmentCtx.metadata()
	if err := e.dispatchSegmentPreparationHandlers(from, to, meta, segmentStep, &stepsNeeded); err != nil {
		e.currentMove = nil
		e.abilityCtx.clear()
		return err
	}

	if stepsNeeded > e.currentMove.RemainingSteps {
		return fmt.Errorf("insufficient steps: %d needed, %d remaining", stepsNeeded, e.currentMove.RemainingSteps)
	}

	if len(e.handlersForAbility(AbilityResurrection)) == 0 && pc.Abilities.Contains(AbilityResurrection) && e.currentMove.abilityFlag(AbilityResurrection, abilityFlagWindow) {
		e.currentMove.setAbilityFlag(AbilityResurrection, abilityFlagWindow, false)
		e.currentMove.setAbilityCounter(AbilityResurrection, abilityFlagWindow, 0)
	}

	// Continuation is valid, push state and execute.
	delta := e.pushHistory()
	defer e.finalizeHistory(delta)

	e.recordSquareForUndo(from)
	e.recordSquareForUndo(to)
	if segmentCtx.capture != nil {
		e.recordSquareForUndo(segmentCtx.captureSquare)
	}
	e.currentMove.RemainingSteps -= stepsNeeded
	if err := e.dispatchSegmentStartHandlers(from, to, meta, segmentStep); err != nil {
		e.currentMove = nil
		e.abilityCtx.clear()
		return err
	}
	e.executeMoveSegment(from, to, segmentCtx)
	e.currentMove.Path = append(e.currentMove.Path, to)
	if err := e.handlePostSegment(pc, from, to, segmentCtx.metadata()); err != nil {
		e.currentMove = nil
		e.abilityCtx.clear()
		return err
	}
	if err := e.dispatchSegmentResolvedHandlers(from, to, meta, segmentStep, stepsNeeded); err != nil {
		e.currentMove = nil
		e.abilityCtx.clear()
		return err
	}

	if segmentCtx.capture != nil {
		e.currentMove.registerCapture(meta)
		if err := e.dispatchCaptureHandlers(pc, segmentCtx.capture, segmentCtx.captureSquare, segmentStep); err != nil {
			e.currentMove = nil
			e.abilityCtx.clear()
			return err
		}
		outcome, err := e.ResolveCaptureAbility(pc, segmentCtx.capture, segmentCtx.captureSquare, segmentStep)
		if err != nil {
			e.currentMove = nil
			e.abilityCtx.clear()
			return err
		}
		if delta := outcome.StepAdjustment; delta != 0 {
			e.currentMove.RemainingSteps += delta
			if e.currentMove.RemainingSteps < 0 {
				e.currentMove.RemainingSteps = 0
			}
		}
		if !e.currentMove.canCaptureMore() {
			e.currentMove.TurnEnded = true
			e.endTurn(TurnEndForced)
			return nil
		}
		if outcome.ForceTurnEnd {
			e.currentMove.TurnEnded = true
			e.endTurn(TurnEndForced)
			return nil
		}
	}

	// Check for turn-ending conditions after the action.
	if e.checkPostCaptureTermination(pc, segmentCtx.capture) {
		e.endTurn(TurnEndForced)
	} else if e.currentMove.RemainingSteps <= 0 && !e.hasFreeContinuation(pc) {
		e.endTurn(TurnEndNatural)
	} else {
		appendAbilityNote(&e.board.lastNote, fmt.Sprintf("%d steps remaining", e.currentMove.RemainingSteps))
	}

	return nil
}

func (e *Engine) trySideStepNudge(pc *Piece, from, to Square) (bool, error) {
	if e.currentMove == nil || pc == nil {
		return false, nil
	}
	if handlers := e.handlersForAbility(AbilitySideStep); len(handlers) > 0 {
		segmentStep := len(e.currentMove.Path) - 1
		if segmentStep < 0 {
			segmentStep = 0
		}
		ctx := &e.abilityCtx.segment
		// Reuse the cached segment context slot to avoid a new allocation
		// while building the special-move planning request.
		*ctx = SegmentContext{
			Engine:      e,
			Move:        e.currentMove,
			From:        from,
			To:          to,
			SegmentStep: segmentStep,
		}
		planCtx := SpecialMoveContext{
			Engine:      e,
			Move:        e.currentMove,
			Piece:       pc,
			From:        from,
			To:          to,
			Ability:     AbilitySideStep,
			SegmentStep: segmentStep,
		}
		defer func() {
			e.abilityCtx.segment = SegmentContext{}
		}()
		for _, handler := range handlers {
			planner, ok := handler.(SpecialMoveHandler)
			if !ok {
				continue
			}
			plan, handled, err := planner.PlanSpecialMove(&planCtx)
			if err != nil {
				e.currentMove = nil
				e.abilityCtx.clear()
				return true, err
			}
			if !handled {
				continue
			}
			if plan.MarkAbilityUsed && plan.Ability == AbilityNone {
				plan.Ability = AbilitySideStep
			}
			if plan.Action == SpecialMoveActionNone {
				plan.Action = SpecialMoveActionMove
			}
			if plan.StepCost < 0 {
				plan.StepCost = 0
			}
			return true, e.executeSpecialMovePlan(pc, from, to, plan)
		}
	}
	if !pc.Abilities.Contains(AbilitySideStep) || e.currentMove.abilityUsed(AbilitySideStep) || e.currentMove.RemainingSteps <= 0 {
		return false, nil
	}
	if !isAdjacentSquare(from, to) {
		return false, nil
	}

	if target := e.board.pieceAt[to]; target != nil {
		return false, nil
	}

	delta := e.pushHistory()
	defer e.finalizeHistory(delta)

	e.recordSquareForUndo(from)
	e.recordSquareForUndo(to)
	e.currentMove.RemainingSteps--
	e.currentMove.markAbilityUsed(AbilitySideStep)

	segmentCtx := moveSegmentContext{}
	segmentStep := len(e.currentMove.Path) - 1
	if segmentStep < 0 {
		segmentStep = 0
	}
	if err := e.dispatchSegmentStartHandlers(from, to, segmentCtx.metadata(), segmentStep); err != nil {
		e.currentMove = nil
		e.abilityCtx.clear()
		return true, err
	}
	e.executeMoveSegment(from, to, segmentCtx)
	e.currentMove.Path = append(e.currentMove.Path, to)
	if err := e.handlePostSegment(pc, from, to, segmentCtx.metadata()); err != nil {
		e.currentMove = nil
		e.abilityCtx.clear()
		return true, err
	}

	if err := e.dispatchSegmentResolvedHandlers(from, to, segmentCtx.metadata(), segmentStep, 1); err != nil {
		e.currentMove = nil
		e.abilityCtx.clear()
		return true, err
	}

	if e.currentMove != nil {
		if pc.Abilities.Contains(AbilityResurrection) {
			e.currentMove.setAbilityFlag(AbilityResurrection, abilityFlagWindow, false)
			e.currentMove.setAbilityCounter(AbilityResurrection, abilityFlagWindow, 0)
		}
	}

	appendAbilityNote(&e.board.lastNote, "Side Step nudge (cost 1 step)")

	if e.checkPostCaptureTermination(pc, nil) {
		e.endTurn(TurnEndForced)
	} else if e.currentMove.RemainingSteps <= 0 && !e.hasFreeContinuation(pc) {
		e.endTurn(TurnEndNatural)
	} else {
		appendAbilityNote(&e.board.lastNote, fmt.Sprintf("%d steps remaining", e.currentMove.RemainingSteps))
	}

	return true, nil
}

func (e *Engine) tryQuantumStep(pc *Piece, from, to Square) (bool, error) {
	if e.currentMove == nil || pc == nil {
		return false, nil
	}
	if handlers := e.handlersForAbility(AbilityQuantumStep); len(handlers) > 0 {
		segmentStep := len(e.currentMove.Path) - 1
		if segmentStep < 0 {
			segmentStep = 0
		}
		planCtx := SpecialMoveContext{
			Engine:      e,
			Move:        e.currentMove,
			Piece:       pc,
			From:        from,
			To:          to,
			Ability:     AbilityQuantumStep,
			SegmentStep: segmentStep,
		}
		for _, handler := range handlers {
			planner, ok := handler.(SpecialMoveHandler)
			if !ok {
				continue
			}
			plan, handled, err := planner.PlanSpecialMove(&planCtx)
			if err != nil {
				e.currentMove = nil
				e.abilityCtx.clear()
				return true, err
			}
			if !handled {
				continue
			}
			if plan.MarkAbilityUsed && plan.Ability == AbilityNone {
				plan.Ability = AbilityQuantumStep
			}
			if plan.Action == SpecialMoveActionNone {
				plan.Action = SpecialMoveActionMove
			}
			if plan.StepCost < 0 {
				plan.StepCost = 0
			}
			return true, e.executeSpecialMovePlan(pc, from, to, plan)
		}
	}
	if !pc.Abilities.Contains(AbilityQuantumStep) || e.currentMove.abilityUsed(AbilityQuantumStep) || e.currentMove.RemainingSteps <= 0 {
		return false, nil
	}

	ally, ok := e.validateQuantumStep(pc, from, to)
	if !ok {
		return false, nil
	}

	delta := e.pushHistory()
	defer e.finalizeHistory(delta)

	e.recordSquareForUndo(from)
	e.recordSquareForUndo(to)
	e.currentMove.RemainingSteps--
	if e.currentMove.RemainingSteps < 0 {
		e.currentMove.RemainingSteps = 0
	}
	e.currentMove.markAbilityUsed(AbilityQuantumStep)
	if pc.Abilities.Contains(AbilityResurrection) {
		e.currentMove.setAbilityFlag(AbilityResurrection, abilityFlagWindow, false)
		e.currentMove.setAbilityCounter(AbilityResurrection, abilityFlagWindow, 0)
	}

	segmentStep := len(e.currentMove.Path) - 1
	if segmentStep < 0 {
		segmentStep = 0
	}
	segmentCtx := moveSegmentContext{}
	if err := e.dispatchSegmentStartHandlers(from, to, segmentCtx.metadata(), segmentStep); err != nil {
		e.currentMove = nil
		e.abilityCtx.clear()
		return true, err
	}
	if ally == nil {
		e.executeMoveSegment(from, to, segmentCtx)
		appendAbilityNote(&e.board.lastNote, "Quantum Step blink (cost 1 step)")
	} else {
		e.performQuantumSwap(pc, ally, from, to)
		appendAbilityNote(&e.board.lastNote, "Quantum Step swap (cost 1 step)")
	}

	e.currentMove.Path = append(e.currentMove.Path, to)
	if err := e.handlePostSegment(pc, from, to, segmentCtx.metadata()); err != nil {
		e.currentMove = nil
		e.abilityCtx.clear()
		return true, err
	}

	if err := e.dispatchSegmentResolvedHandlers(from, to, segmentCtx.metadata(), segmentStep, 1); err != nil {
		e.currentMove = nil
		e.abilityCtx.clear()
		return true, err
	}

	if e.checkPostCaptureTermination(pc, nil) {
		e.endTurn(TurnEndForced)
		return true, nil
	}
	if e.currentMove == nil {
		return true, nil
	}
	if e.currentMove.RemainingSteps <= 0 && !e.hasFreeContinuation(pc) {
		e.endTurn(TurnEndNatural)
	} else {
		appendAbilityNote(&e.board.lastNote, fmt.Sprintf("%d steps remaining", e.currentMove.RemainingSteps))
	}

	return true, nil
}

func (e *Engine) executeSpecialMovePlan(pc *Piece, from, to Square, plan SpecialMovePlan) error {
	if e.currentMove == nil {
		return errors.New("no active move to continue")
	}
	stepsNeeded := plan.StepCost
	if stepsNeeded < 0 {
		stepsNeeded = 0
	}
	if stepsNeeded > e.currentMove.RemainingSteps {
		return fmt.Errorf("insufficient steps: %d needed, %d remaining", stepsNeeded, e.currentMove.RemainingSteps)
	}

	delta := e.pushHistory()
	defer e.finalizeHistory(delta)

	e.recordSquareForUndo(from)
	e.recordSquareForUndo(to)
	if plan.Metadata.Capture != nil {
		e.recordSquareForUndo(plan.Metadata.CaptureSquare)
	}

	e.currentMove.RemainingSteps -= stepsNeeded
	if plan.ClampRemaining && e.currentMove.RemainingSteps < 0 {
		e.currentMove.RemainingSteps = 0
	}
	if plan.MarkAbilityUsed && plan.Ability != AbilityNone {
		e.currentMove.markAbilityUsed(plan.Ability)
	}
	if plan.ResetResurrection && pc != nil && pc.Abilities.Contains(AbilityResurrection) {
		e.currentMove.setAbilityFlag(AbilityResurrection, abilityFlagWindow, false)
		e.currentMove.setAbilityCounter(AbilityResurrection, abilityFlagWindow, 0)
	}

	segmentStep := len(e.currentMove.Path) - 1
	if segmentStep < 0 {
		segmentStep = 0
	}

	segmentCtx := moveSegmentContext{
		capture:       plan.Metadata.Capture,
		captureSquare: plan.Metadata.CaptureSquare,
		enPassant:     plan.Metadata.EnPassant,
	}
	meta := segmentCtx.metadata()

	if err := e.dispatchSegmentStartHandlers(from, to, meta, segmentStep); err != nil {
		e.currentMove = nil
		e.abilityCtx.clear()
		return err
	}

	switch plan.Action {
	case SpecialMoveActionSwap:
		if plan.SwapWith == nil {
			e.currentMove = nil
			e.abilityCtx.clear()
			return errors.New("invalid special move plan: missing swap target")
		}
		e.performQuantumSwap(pc, plan.SwapWith, from, to)
	default:
		e.executeMoveSegment(from, to, segmentCtx)
	}

	e.currentMove.Path = append(e.currentMove.Path, to)
	if err := e.handlePostSegment(pc, from, to, meta); err != nil {
		e.currentMove = nil
		e.abilityCtx.clear()
		return err
	}

	if err := e.dispatchSegmentResolvedHandlers(from, to, meta, segmentStep, stepsNeeded); err != nil {
		e.currentMove = nil
		e.abilityCtx.clear()
		return err
	}

	if plan.Note != "" {
		appendAbilityNote(&e.board.lastNote, plan.Note)
	}

	if plan.Metadata.Capture != nil {
		e.currentMove.registerCapture(meta)
		if err := e.dispatchCaptureHandlers(pc, plan.Metadata.Capture, plan.Metadata.CaptureSquare, segmentStep); err != nil {
			e.currentMove = nil
			e.abilityCtx.clear()
			return err
		}
		outcome, err := e.ResolveCaptureAbility(pc, plan.Metadata.Capture, plan.Metadata.CaptureSquare, segmentStep)
		if err != nil {
			e.currentMove = nil
			e.abilityCtx.clear()
			return err
		}
		if delta := outcome.StepAdjustment; delta != 0 {
			e.currentMove.RemainingSteps += delta
			if e.currentMove.RemainingSteps < 0 {
				e.currentMove.RemainingSteps = 0
			}
		}
		if !e.currentMove.canCaptureMore() {
			e.currentMove.TurnEnded = true
			e.endTurn(TurnEndForced)
			return nil
		}
		if outcome.ForceTurnEnd {
			e.currentMove.TurnEnded = true
			e.endTurn(TurnEndForced)
			return nil
		}
	}

	if e.checkPostCaptureTermination(pc, plan.Metadata.Capture) {
		e.endTurn(TurnEndForced)
		return nil
	}
	if e.currentMove == nil {
		return nil
	}
	if e.currentMove.RemainingSteps <= 0 && !e.hasFreeContinuation(pc) {
		e.endTurn(TurnEndNatural)
	} else {
		appendAbilityNote(&e.board.lastNote, fmt.Sprintf("%d steps remaining", e.currentMove.RemainingSteps))
	}
	return nil
}

func (e *Engine) validateQuantumStep(pc *Piece, from, to Square) (*Piece, bool) {
	if pc == nil {
		return nil, false
	}
	if !isAdjacentSquare(from, to) {
		return nil, false
	}

	target := e.board.pieceAt[to]
	if target != nil && target.Color != pc.Color {
		return nil, false
	}

	if target == nil && e.isLegalContinuation(pc, from, to) {
		return nil, false
	}

	if target != nil && target.Color == pc.Color {
		return target, true
	}

	return nil, true
}

func (e *Engine) performQuantumSwap(pc, ally *Piece, from, to Square) {
	if pc == nil || ally == nil {
		return
	}
	if ally.Color != pc.Color {
		return
	}

	e.recordSquareForUndo(from)
	e.recordSquareForUndo(to)

	e.board.EnPassant = NoEnPassantTarget()

	e.board.pieceAt[from] = ally
	e.board.pieceAt[to] = pc

	pc.Square = to
	ally.Square = from

	e.board.pieces[pc.Color][pc.Type] = e.board.pieces[pc.Color][pc.Type].Remove(from).Add(to)
	e.board.pieces[ally.Color][ally.Type] = e.board.pieces[ally.Color][ally.Type].Remove(to).Add(from)

	occ := e.board.occupancy[pc.Color]
	occ = occ.Remove(from)
	occ = occ.Remove(to)
	occ = occ.Add(from)
	occ = occ.Add(to)
	e.board.occupancy[pc.Color] = occ

	all := e.board.allOcc
	all = all.Remove(from)
	all = all.Remove(to)
	all = all.Add(from)
	all = all.Add(to)
	e.board.allOcc = all

	e.updateCastlingRightsForMove(pc, from)
	e.updateCastlingRightsForMove(ally, to)
}

func isAdjacentSquare(from, to Square) bool {
	dr := absInt(to.Rank() - from.Rank())
	df := absInt(to.File() - from.File())
	if dr == 0 && df == 0 {
		return false
	}
	return dr <= 1 && df <= 1
}

// endTurn finalizes the move, performs cleanup, and passes control to the other player.
func (e *Engine) endTurn(reason TurnEndReason) {
	if e.currentMove == nil {
		// This can happen if a move was aborted (e.g., DoOver).
		return
	}

	pc := e.currentMove.Piece
	outcome, handlerErr := e.dispatchTurnEndHandlers(reason)
	e.resolvePromotion(pc)
	e.flipTurn()
	e.updateGameStatus()

	e.applyTurnEndOutcome(outcome)
	if handlerErr != nil {
		appendAbilityNote(&e.board.lastNote, fmt.Sprintf("Turn end handler error: %v", handlerErr))
	}

	var note string
	switch {
	case e.board.GameOver && e.board.Status == "checkmate" && e.board.HasWinner:
		note = fmt.Sprintf("Checkmate - %s wins", e.board.Winner.String())
	case e.board.GameOver && e.board.Status == "stalemate":
		note = "Stalemate"
	case e.board.GameOver:
		note = e.board.Status
	case e.board.InCheck:
		note = fmt.Sprintf("%s to move (in check)", e.board.turn)
	default:
		note = fmt.Sprintf("%s's turn", e.board.turn)
	}
	appendAbilityNote(&e.board.lastNote, note)

	// Clear the current move state, officially ending the turn.
	e.currentMove = nil
	e.abilityCtx.clear()
	e.resetAbilityHandlers()
}

// ---------------------------
// Step & Cost Calculation
// ---------------------------

// baseStepBudget calculates the total number of steps a piece gets for its turn without handler overrides.
func (e *Engine) baseStepBudget(pc *Piece) int {
	baseSteps := 1 // Every piece gets at least one step.
	bonus := 0
	element := elementOf(e, pc)

	// Elemental & Ability Bonuses/Penalties
	if pc.Abilities.Contains(AbilityScorch) && element == ElementFire {
		bonus++ // Scorch grants +1 step
	}
	if pc.Abilities.Contains(AbilityTailwind) && element == ElementAir {
		bonus += 2 // Tailwind grants +2 steps
		if pc.Abilities.Contains(AbilityTemporalLock) {
			bonus-- // Temporal Lock slows Tailwind by 1
		}
	}
	if pc.Abilities.Contains(AbilityRadiantVision) && element == ElementLight {
		bonus++ // Radiant Vision grants +1 step
		if pc.Abilities.Contains(AbilityMistShroud) {
			bonus++ // Mist combo grants an additional step
		}
	}
	if pc.Abilities.Contains(AbilityUmbralStep) && element == ElementShadow {
		bonus += 2 // Umbral Step grants +2 steps
		if pc.Abilities.Contains(AbilityRadiantVision) {
			bonus-- // Radiant Vision dampens Umbral Step by 1
		}
	}
	if pc.Abilities.Contains(AbilitySchrodingersLaugh) {
		bonus += 2 // Schrodinger's Laugh grants +2 steps
		if pc.Abilities.Contains(AbilitySideStep) {
			bonus++ // Interaction bonus with Side Step
		}
	}
	slowPenalty := e.temporalSlow[pc.Color.Index()]
	if slowPenalty > 0 {
		e.temporalSlow[pc.Color.Index()] = 0
	}

	totalSteps := baseSteps + bonus - slowPenalty
	if totalSteps < 1 {
		return 1 // A piece always gets at least 1 step.
	}
	return totalSteps
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

// calculateMovementCost calculates the step cost for a given move segment.
func (e *Engine) calculateMovementCost(pc *Piece, from, to Square) int {
	cost := 1 // Basic movement costs 1 step.

	target := e.board.pieceAt[to]

	// Sliders (Queen, Rook, Bishop) pay an extra step to change direction mid-turn.
	if e.isSlider(pc.Type) {
		pathLen := 0
		if e.currentMove != nil {
			pathLen = len(e.currentMove.Path)
		}
		if pathLen > 1 {
			prevFrom := e.currentMove.Path[pathLen-2]
			prevDir := shared.DirectionOf(prevFrom, from)
			currentDir := shared.DirectionOf(from, to)

			if prevDir != currentDir && prevDir != DirNone && currentDir != DirNone {
				if !(pc.Abilities.Contains(AbilityMistShroud) && e.currentMove != nil && e.currentMove.abilityCounter(AbilityMistShroud, abilityCounterFree) == 0) {
					cost++ // Direction change costs an extra step.
				}
			}
		}
	}

	if e.isFloodWakePushAvailable(pc, from, to, target) {
		cost = 0
	}

	if e.isBlazeRushDash(pc, from, to, target) {
		cost = 0
	}

	return cost
}

func (e *Engine) isFloodWakePushAvailable(pc *Piece, from, to Square, target *Piece) bool {
	if pc == nil || target != nil {
		return false
	}
	if !pc.Abilities.Contains(AbilityFloodWake) {
		return false
	}
	if elementOf(e, pc) != ElementWater {
		return false
	}
	dr := absInt(to.Rank() - from.Rank())
	df := absInt(to.File() - from.File())
	if dr+df != 1 {
		return false
	}
	if e.currentMove != nil && e.currentMove.abilityUsed(AbilityFloodWake) {
		return false
	}
	return true
}

func (e *Engine) isBlazeRushDash(pc *Piece, from, to Square, target *Piece) bool {
	if pc == nil || target != nil {
		return false
	}
	if !pc.Abilities.Contains(AbilityBlazeRush) {
		return false
	}
	if !e.isSlider(pc.Type) {
		return false
	}
	if e.currentMove == nil {
		return false
	}
	if e.currentMove.abilityUsed(AbilityBlazeRush) || e.currentMove.LastSegmentCaptured {
		return false
	}
	pathLen := len(e.currentMove.Path)
	var prevStart, prevEnd Square
	switch {
	case pathLen >= 3 && e.currentMove.Path[pathLen-1] == to && e.currentMove.Path[pathLen-2] == from:
		prevStart = e.currentMove.Path[pathLen-3]
		prevEnd = from
	case pathLen >= 2 && e.currentMove.Path[pathLen-1] == from:
		prevStart = e.currentMove.Path[pathLen-2]
		prevEnd = from
	default:
		return false
	}
	if !e.blazeRushSegmentOk(prevStart, prevEnd, from, to) {
		return false
	}
	return true
}

func (e *Engine) handlePostSegment(pc *Piece, from, to Square, meta SegmentMetadata) error {
	if e.currentMove == nil {
		return nil
	}

	e.currentMove.LastSegmentCaptured = meta.Capture != nil

	step := len(e.currentMove.Path) - 1
	if step < 0 {
		step = 0
	}

	if err := e.dispatchPostSegmentHandlers(pc, from, to, meta, step); err != nil {
		return err
	}

	e.logDirectionChange(pc, step)
	return nil
}

func (e *Engine) logDirectionChange(pc *Piece, segmentStep int) {
	if e.currentMove == nil || !e.isSlider(pc.Type) {
		return
	}
	if len(e.currentMove.Path) < 3 {
		return
	}

	last := len(e.currentMove.Path) - 1
	prevStart := e.currentMove.Path[last-2]
	prevEnd := e.currentMove.Path[last-1]
	currentEnd := e.currentMove.Path[last]
	prevDir := shared.DirectionOf(prevStart, prevEnd)
	currentDir := shared.DirectionOf(prevEnd, currentEnd)
	if prevDir == DirNone || currentDir == DirNone || prevDir == currentDir {
		return
	}

	if e.dispatchDirectionChangeHandlers(pc, prevStart, prevEnd, currentEnd, prevDir, currentDir, segmentStep) {
		return
	}

	appendAbilityNote(&e.board.lastNote, "Direction change cost +1 step")
}

func (e *Engine) hasFreeContinuation(pc *Piece) bool {
	if e.currentMove == nil || pc == nil {
		return false
	}
	if e.hasBlazeRushOption(pc) {
		return true
	}
	if e.hasFloodWakePushOption(pc) {
		return true
	}
	return false
}

func (e *Engine) hasBlazeRushOption(pc *Piece) bool {
	return e.dispatchFreeContinuationHandlers(AbilityBlazeRush, pc)
}

func (e *Engine) blazeRushSegmentOk(prevStart, prevEnd, from, to Square) bool {
	prevDir := shared.DirectionOf(prevStart, prevEnd)
	currentDir := shared.DirectionOf(from, to)
	if prevDir == DirNone || currentDir == DirNone || prevDir != currentDir {
		return false
	}
	steps := maxInt(absInt(to.Rank()-from.Rank()), absInt(to.File()-from.File()))
	if steps == 0 || steps > 2 {
		return false
	}
	for _, sq := range shared.Line(from, to) {
		if e.board.pieceAt[sq] != nil {
			return false
		}
	}
	return true
}

func (e *Engine) hasFloodWakePushOption(pc *Piece) bool {
	return e.dispatchFreeContinuationHandlers(AbilityFloodWake, pc)
}

func (e *Engine) blazeRushContinuationAvailable(pc *Piece) bool {
	if pc == nil || !pc.Abilities.Contains(AbilityBlazeRush) {
		return false
	}
	if e.currentMove == nil || e.currentMove.abilityUsed(AbilityBlazeRush) || e.currentMove.LastSegmentCaptured {
		return false
	}
	if !e.isSlider(pc.Type) {
		return false
	}
	if len(e.currentMove.Path) < 2 {
		return false
	}
	prevFrom := e.currentMove.Path[len(e.currentMove.Path)-2]
	prevTo := e.currentMove.Path[len(e.currentMove.Path)-1]
	dr, df, ok := directionStep(shared.DirectionOf(prevFrom, prevTo))
	if !ok {
		return false
	}
	nextRank := prevTo.Rank() + dr
	nextFile := prevTo.File() + df
	if sq, valid := shared.SquareFromCoords(nextRank, nextFile); valid {
		if e.board.pieceAt[sq] == nil {
			return true
		}
	}
	return false
}

func (e *Engine) floodWakeContinuationAvailable(pc *Piece) bool {
	if pc == nil || !pc.Abilities.Contains(AbilityFloodWake) {
		return false
	}
	if e.currentMove == nil || e.currentMove.abilityUsed(AbilityFloodWake) {
		return false
	}
	if elementOf(e, pc) != ElementWater {
		return false
	}
	offsets := [...]struct{ dr, df int }{
		{dr: 1, df: 0},
		{dr: -1, df: 0},
		{dr: 0, df: 1},
		{dr: 0, df: -1},
	}
	for _, off := range offsets {
		rank := pc.Square.Rank() + off.dr
		file := pc.Square.File() + off.df
		if sq, valid := shared.SquareFromCoords(rank, file); valid {
			if e.board.pieceAt[sq] == nil {
				return true
			}
		}
	}
	return false
}

func directionStep(dir Direction) (int, int, bool) {
	switch dir {
	case DirN:
		return -1, 0, true
	case DirNE:
		return -1, 1, true
	case DirE:
		return 0, 1, true
	case DirSE:
		return 1, 1, true
	case DirS:
		return 1, 0, true
	case DirSW:
		return 1, -1, true
	case DirW:
		return 0, -1, true
	case DirNW:
		return -1, -1, true
	default:
		return 0, 0, false
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ---------------------------
// Turn State Conditionals
// ---------------------------

// checkPostMoveAbilities checks for abilities that can be activated after a move segment.
func (e *Engine) checkPostMoveAbilities(pc *Piece) {
	// Side Step: Note that an 8-directional nudge is available.
	if pc.Abilities.Contains(AbilitySideStep) && !e.currentMove.abilityUsed(AbilitySideStep) && e.currentMove.RemainingSteps > 0 {
		appendAbilityNote(&e.board.lastNote, "Side Step available (costs 1 step)")
	}
	// Quantum Step: Note that an adjacent relocation is available.
	if pc.Abilities.Contains(AbilityQuantumStep) && !e.currentMove.abilityUsed(AbilityQuantumStep) && e.currentMove.RemainingSteps > 0 {
		appendAbilityNote(&e.board.lastNote, "Quantum Step available (costs 1 step)")
	}
}

// ---------------------------
// Move Legality Helpers
// ---------------------------

func (e *Engine) pathIsPassable(pc *Piece, from, to Square) bool {
	line := shared.Line(from, to)
	if len(line) == 0 {
		return true
	}

	canPhase := e.canPhaseThrough(pc, from, to)
	for _, sq := range line {
		occupant := e.board.pieceAt[sq]
		if occupant == nil {
			continue
		}
		if occupant.Color == pc.Color {
			return false
		}
		if !canPhase {
			return false
		}
		if occupant.Abilities.Contains(AbilityIndomitable) || occupant.Abilities.Contains(AbilityStalwart) {
			return false
		}
	}
	return true
}

func isScatterShotCapture(pc *Piece, from, to Square) bool {
	if pc == nil || !pc.Abilities.Contains(AbilityScatterShot) {
		return false
	}
	if from.Rank() != to.Rank() {
		return false
	}
	fileDiff := from.File() - to.File()
	if fileDiff < 0 {
		fileDiff = -fileDiff
	}
	return fileDiff == 1
}

func (e *Engine) canDirectCapture(attacker, defender *Piece, from, to Square) bool {
	if defender == nil {
		return true
	}
	if attacker == nil {
		return false
	}
	if defender.Abilities.Contains(AbilityStalwart) && rankOf(attacker.Type) < rankOf(defender.Type) {
		return false
	}
	if defender.Abilities.Contains(AbilityBelligerent) && rankOf(attacker.Type) > rankOf(defender.Type) {
		return false
	}
	if isScatterShotCapture(attacker, from, to) && defender.Abilities.Contains(AbilityIndomitable) {
		return false
	}
	return true
}

// wouldLeaveKingInCheck determines whether moving a piece from `from` to `to`
// would result in that piece's own king remaining in or entering check.
func (e *Engine) wouldLeaveKingInCheck(pc *Piece, from, to Square) bool {
	if pc == nil {
		return true
	}

	boardBackup := e.board
	originalSquare := pc.Square

	e.board.pieceAt[from] = nil
	e.board.pieceAt[to] = pc
	pc.Square = to

	inCheck := e.isKingInCheck(pc.Color)

	pc.Square = originalSquare
	e.board = boardBackup

	return inCheck
}

func (e *Engine) findKingSquare(color Color) (Square, bool) {
	for idx, pc := range e.board.pieceAt {
		if pc != nil && pc.Color == color && pc.Type == King {
			return Square(idx), true
		}
	}
	return 0, false
}

func (e *Engine) isSquareAttackedBy(color Color, target Square) bool {
	defender := e.board.pieceAt[target]
	for _, attacker := range e.board.pieceAt {
		if attacker == nil || attacker.Color != color {
			continue
		}
		if attacker.Type == King {
			atkRank := attacker.Square.Rank()
			atkFile := attacker.Square.File()
			for _, delta := range kingOffsets {
				if sq, ok := shared.SquareFromCoords(atkRank+delta.dr, atkFile+delta.df); ok && sq == target {
					if e.canDirectCapture(attacker, defender, attacker.Square, target) {
						return true
					}
				}
			}
			continue
		}
		if !e.pathIsPassable(attacker, attacker.Square, target) {
			continue
		}
		moves := e.generateMoves(attacker)
		if !moves.Has(target) {
			continue
		}
		if !e.canDirectCapture(attacker, defender, attacker.Square, target) {
			continue
		}
		return true
	}
	return false
}

func (e *Engine) isKingInCheck(color Color) bool {
	kingSq, ok := e.findKingSquare(color)
	if !ok {
		return false
	}
	return e.isSquareAttackedBy(color.Opposite(), kingSq)
}

func (e *Engine) hasLegalMove(color Color) bool {
	for _, pc := range e.board.pieceAt {
		if pc == nil || pc.Color != color {
			continue
		}
		from := pc.Square
		moves := e.generateMoves(pc)
		found := false
		moves.Iter(func(to Square) {
			if found {
				return
			}
			if !e.pathIsPassable(pc, from, to) {
				return
			}
			target := e.board.pieceAt[to]
			if !e.canDirectCapture(pc, target, from, to) {
				return
			}
			if !e.wouldLeaveKingInCheck(pc, from, to) {
				found = true
			}
		})
		if found {
			return true
		}
	}
	return false
}

func (e *Engine) updateGameStatus() {
	current := e.board.turn
	inCheck := e.isKingInCheck(current)
	hasMove := e.hasLegalMove(current)

	e.board.InCheck = inCheck
	e.board.GameOver = false
	e.board.HasWinner = false
	e.board.Status = "ongoing"
	e.board.Winner = 0

	if inCheck {
		e.board.Status = "check"
	}

	if !hasMove {
		e.board.GameOver = true
		if inCheck {
			e.board.Status = "checkmate"
			e.board.HasWinner = true
			e.board.Winner = current.Opposite()
		} else {
			e.board.Status = "stalemate"
		}
	}
}

// isLegalFirstSegment checks if the initial move is valid.
func (e *Engine) isLegalFirstSegment(pc *Piece, from, to Square) bool {
	// A real implementation uses detailed move generation.
	// For now, this is a simplified check.
	moves := e.generateMoves(pc)
	if !moves.Has(to) {
		return false
	}
	if !e.pathIsPassable(pc, from, to) {
		return false
	}
	target := e.board.pieceAt[to]
	if !e.canDirectCapture(pc, target, from, to) {
		return false
	}
	if e.wouldLeaveKingInCheck(pc, from, to) {
		return false
	}
	return true
}

// isLegalContinuation checks if a subsequent move segment is valid.
func (e *Engine) isLegalContinuation(pc *Piece, from, to Square) bool {
	// For normal continuations, movement rules are more restrictive.
	// This could be, for example, continuing a slide in the same direction.
	// For simplicity, we'll allow any legal move for now.
	if !e.isLegalFirstSegment(pc, from, to) {
		return false
	}
	return true
}
