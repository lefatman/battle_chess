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

func (mistShroudFallbackHandler) OnDirectionChange(ctx DirectionChangeContext) bool {
	if ctx.Engine == nil || ctx.Move == nil || ctx.Piece == nil {
		return false
	}
	if !ctx.Piece.Abilities.Contains(AbilityMistShroud) {
		return false
	}
	if ctx.Move.abilityCounter(AbilityMistShroud, abilityCounterFree) != 0 {
		return false
	}
	ctx.Move.addAbilityCounter(AbilityMistShroud, abilityCounterFree, 1)
	appendAbilityNote(&ctx.Engine.board.lastNote, "Mist Shroud free pivot")
	return true
}

// Ensure the fallback handler satisfies optional interfaces used by the
// dispatcher without exposing additional methods.
var (
	_ FreeContinuationHandler = blazeRushFallbackHandler{}
	_ FreeContinuationHandler = floodWakeFallbackHandler{}
	_ DirectionChangeHandler  = mistShroudFallbackHandler{}
)
