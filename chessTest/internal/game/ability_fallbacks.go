// path: chessTest/internal/game/ability_fallbacks.go
package game

// abilityHandlerBase provides default implementations for AbilityHandler
// methods that many fallback handlers do not need to customise.
type abilityHandlerBase struct{}

func (abilityHandlerBase) StepBudgetModifier(StepBudgetContext) (StepBudgetDelta, error) {
	return StepBudgetDelta{}, nil
}

func (abilityHandlerBase) CanPhase(PhaseContext) (bool, error) {
	return false, nil
}

func (abilityHandlerBase) OnMoveStart(MoveLifecycleContext) error {
	return nil
}

func (abilityHandlerBase) OnSegmentStart(SegmentContext) error {
	return nil
}

func (abilityHandlerBase) OnPostSegment(PostSegmentContext) error {
	return nil
}

func (abilityHandlerBase) OnCapture(CaptureContext) error {
	return nil
}

func (abilityHandlerBase) OnTurnEnd(TurnEndContext) error {
	return nil
}

func (abilityHandlerBase) ResolveCapture(CaptureContext) (CaptureOutcome, error) {
	return CaptureOutcome{}, nil
}

func (abilityHandlerBase) ResolveTurnEnd(TurnEndContext) (TurnEndOutcome, error) {
	return TurnEndOutcome{}, nil
}

// newBlazeRushFallbackHandler returns a default handler that mirrors the
// existing Blaze Rush behaviour for engines without a registered handler.
func newBlazeRushFallbackHandler() AbilityHandler {
	return blazeRushFallbackHandler{}
}

type blazeRushFallbackHandler struct {
	abilityHandlerBase
}

func (blazeRushFallbackHandler) PlanSpecialMove(ctx *SpecialMoveContext) (SpecialMovePlan, bool, error) {
	return SpecialMovePlan{}, false, nil
}

func (blazeRushFallbackHandler) PrepareSegment(ctx *SegmentPreparationContext) error {
	if ctx == nil || ctx.Engine == nil || ctx.Move == nil {
		return nil
	}
	pc := ctx.Move.Piece
	if pc == nil || !pc.HasAbility(AbilityBlazeRush) {
		return nil
	}
	if !ctx.Engine.isBlazeRushDash(pc, ctx.From, ctx.To, ctx.Segment.Capture) {
		return nil
	}
	if ctx.StepCost != nil {
		*ctx.StepCost = 0
	}
	return nil
}

func (blazeRushFallbackHandler) OnPostSegment(ctx PostSegmentContext) error {
	if ctx.Engine == nil || ctx.Move == nil || ctx.Piece == nil {
		return nil
	}
	if ctx.Engine.isBlazeRushDash(ctx.Piece, ctx.From, ctx.To, ctx.Segment.Capture) {
		ctx.Move.markAbilityUsed(AbilityBlazeRush)
		appendAbilityNote(&ctx.Engine.board.lastNote, "Blaze Rush dash (free)")
	}
	return nil
}

func (blazeRushFallbackHandler) FreeContinuationAvailable(ctx FreeContinuationContext) bool {
	if ctx.Engine == nil || ctx.Piece == nil {
		return false
	}
	return ctx.Engine.blazeRushContinuationAvailable(ctx.Piece)
}

// newFloodWakeFallbackHandler returns a default handler that mirrors the
// existing Flood Wake behaviour for engines without a registered handler.
func newFloodWakeFallbackHandler() AbilityHandler {
	return floodWakeFallbackHandler{}
}

type floodWakeFallbackHandler struct {
	abilityHandlerBase
}

func (floodWakeFallbackHandler) CanPhase(PhaseContext) (bool, error) {
	return false, ErrPhaseDenied
}

func (floodWakeFallbackHandler) PlanSpecialMove(ctx *SpecialMoveContext) (SpecialMovePlan, bool, error) {
	return SpecialMovePlan{}, false, nil
}

func (floodWakeFallbackHandler) PrepareSegment(ctx *SegmentPreparationContext) error {
	if ctx == nil || ctx.Engine == nil || ctx.Move == nil {
		return nil
	}
	pc := ctx.Move.Piece
	if pc == nil || !pc.HasAbility(AbilityFloodWake) {
		return nil
	}
	if !ctx.Engine.isFloodWakePushAvailable(pc, ctx.From, ctx.To, ctx.Segment.Capture) {
		return nil
	}
	if ctx.StepCost != nil {
		*ctx.StepCost = 0
	}
	return nil
}

func (floodWakeFallbackHandler) OnPostSegment(ctx PostSegmentContext) error {
	if ctx.Engine == nil || ctx.Move == nil || ctx.Piece == nil {
		return nil
	}
	if ctx.Engine.isFloodWakePushAvailable(ctx.Piece, ctx.From, ctx.To, ctx.Segment.Capture) {
		ctx.Move.markAbilityUsed(AbilityFloodWake)
		appendAbilityNote(&ctx.Engine.board.lastNote, "Flood Wake push (free)")
	}
	return nil
}

func (floodWakeFallbackHandler) FreeContinuationAvailable(ctx FreeContinuationContext) bool {
	if ctx.Engine == nil || ctx.Piece == nil {
		return false
	}
	return ctx.Engine.floodWakeContinuationAvailable(ctx.Piece)
}

// newMistShroudFallbackHandler returns a default handler that mirrors the
// existing Mist Shroud behaviour for engines without a registered handler.
func newMistShroudFallbackHandler() AbilityHandler {
	return mistShroudFallbackHandler{}
}

type mistShroudFallbackHandler struct {
	abilityHandlerBase
}

func (mistShroudFallbackHandler) PlanSpecialMove(ctx *SpecialMoveContext) (SpecialMovePlan, bool, error) {
	return SpecialMovePlan{}, false, nil
}

func (mistShroudFallbackHandler) PrepareSegment(ctx *SegmentPreparationContext) error {
	if ctx == nil || ctx.Engine == nil || ctx.Move == nil {
		return nil
	}
	move := ctx.Move
	pc := move.Piece
	if pc == nil || !pc.HasAbility(AbilityMistShroud) {
		return nil
	}
	if move.abilityCounter(AbilityMistShroud, abilityCounterFree) != 0 {
		return nil
	}
	if !ctx.Engine.wouldChangeDirection(move, ctx.From, ctx.To) {
		return nil
	}
	if ctx.StepCost != nil && *ctx.StepCost > 0 {
		*ctx.StepCost--
	}
	return nil
}

func (mistShroudFallbackHandler) OnDirectionChange(ctx DirectionChangeContext) bool {
	if ctx.Engine == nil || ctx.Move == nil || ctx.Piece == nil {
		return false
	}
	if !ctx.Piece.HasAbility(AbilityMistShroud) {
		return false
	}
	if ctx.Move.abilityCounter(AbilityMistShroud, abilityCounterFree) != 0 {
		return false
	}
	ctx.Move.addAbilityCounter(AbilityMistShroud, abilityCounterFree, 1)
	appendAbilityNote(&ctx.Engine.board.lastNote, "Mist Shroud free pivot")
	return true
}

// newSideStepFallbackHandler returns a default handler that mirrors the
// existing Side Step behaviour for engines without a registered handler.
func newSideStepFallbackHandler() AbilityHandler {
	return sideStepFallbackHandler{}
}

type sideStepFallbackHandler struct {
	abilityHandlerBase
}

func (sideStepFallbackHandler) PlanSpecialMove(ctx *SpecialMoveContext) (SpecialMovePlan, bool, error) {
	if ctx == nil || ctx.Engine == nil || ctx.Move == nil || ctx.Piece == nil {
		return SpecialMovePlan{}, false, nil
	}
	pc := ctx.Piece
	if !pc.HasAbility(AbilitySideStep) {
		return SpecialMovePlan{}, false, nil
	}
	if ctx.Move.abilityUsed(AbilitySideStep) || ctx.Move.RemainingSteps <= 0 {
		return SpecialMovePlan{}, false, nil
	}
	if !isAdjacentSquare(ctx.From, ctx.To) {
		return SpecialMovePlan{}, false, nil
	}
	if target := ctx.Engine.board.pieceAt[ctx.To]; target != nil {
		return SpecialMovePlan{}, false, nil
	}

	plan := SpecialMovePlan{
		StepCost:          1,
		Action:            SpecialMoveActionMove,
		Note:              "Side Step nudge (cost 1 step)",
		Ability:           AbilitySideStep,
		MarkAbilityUsed:   true,
		ResetResurrection: true,
	}

	return plan, true, nil
}

// newQuantumStepFallbackHandler returns a default handler that mirrors the
// existing Quantum Step behaviour for engines without a registered handler.
func newQuantumStepFallbackHandler() AbilityHandler {
	return quantumStepFallbackHandler{}
}

type quantumStepFallbackHandler struct {
	abilityHandlerBase
}

func (quantumStepFallbackHandler) PlanSpecialMove(ctx *SpecialMoveContext) (SpecialMovePlan, bool, error) {
	if ctx == nil || ctx.Engine == nil || ctx.Move == nil || ctx.Piece == nil {
		return SpecialMovePlan{}, false, nil
	}
	pc := ctx.Piece
	if !pc.HasAbility(AbilityQuantumStep) {
		return SpecialMovePlan{}, false, nil
	}
	if ctx.Move.abilityUsed(AbilityQuantumStep) || ctx.Move.RemainingSteps <= 0 {
		return SpecialMovePlan{}, false, nil
	}

	ally, ok := ctx.Engine.validateQuantumStep(pc, ctx.From, ctx.To)
	if !ok {
		return SpecialMovePlan{}, false, nil
	}

	plan := SpecialMovePlan{
		StepCost:          1,
		Ability:           AbilityQuantumStep,
		MarkAbilityUsed:   true,
		ResetResurrection: true,
	}

	if ally == nil {
		plan.Action = SpecialMoveActionMove
		plan.Note = "Quantum Step blink (cost 1 step)"
	} else {
		plan.Action = SpecialMoveActionSwap
		plan.SwapWith = ally
		plan.Note = "Quantum Step swap (cost 1 step)"
	}

	return plan, true, nil
}

// Ensure the fallback handler satisfies optional interfaces used by the
// dispatcher without exposing additional methods.
var (
	_ FreeContinuationHandler   = blazeRushFallbackHandler{}
	_ FreeContinuationHandler   = floodWakeFallbackHandler{}
	_ DirectionChangeHandler    = mistShroudFallbackHandler{}
	_ SegmentPreparationHandler = blazeRushFallbackHandler{}
	_ SegmentPreparationHandler = floodWakeFallbackHandler{}
	_ SegmentPreparationHandler = mistShroudFallbackHandler{}
	_ SpecialMoveHandler        = sideStepFallbackHandler{}
	_ SpecialMoveHandler        = quantumStepFallbackHandler{}
)

func init() {
	registerAbilityFallback(AbilityBlazeRush, newBlazeRushFallbackHandler())
	registerAbilityFallback(AbilityFloodWake, newFloodWakeFallbackHandler())
	registerAbilityFallback(AbilitySideStep, newSideStepFallbackHandler())
	registerAbilityFallback(AbilityQuantumStep, newQuantumStepFallbackHandler())
	registerAbilityFallback(AbilityMistShroud, newMistShroudFallbackHandler())
	registerAbilityFallback(AbilityDoubleKill, NewDoubleKillHandler())
	registerAbilityFallback(AbilityScorch, NewScorchHandler())
	registerAbilityFallback(AbilityTailwind, NewTailwindHandler())
	registerAbilityFallback(AbilityRadiantVision, NewRadiantVisionHandler())
	registerAbilityFallback(AbilityUmbralStep, NewUmbralStepHandler())
	registerAbilityFallback(AbilityQuantumKill, NewQuantumKillHandler())
	registerAbilityFallback(AbilityChainKill, NewChainKillHandler())
	registerAbilityFallback(AbilityGaleLift, NewGaleLiftHandler())
	registerAbilityFallback(AbilityPoisonousMeat, NewPoisonousMeatHandler())
	registerAbilityFallback(AbilityOverload, NewOverloadHandler())
	registerAbilityFallback(AbilityBastion, NewBastionHandler())
	registerAbilityFallback(AbilitySchrodingersLaugh, NewSchrodingersLaughHandler())
	registerAbilityFallback(AbilityTemporalLock, NewTemporalLockHandler())
	registerAbilityFallback(AbilityResurrection, NewResurrectionHandler())
}
