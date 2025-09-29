package game

// NewChainKillHandler constructs the default Chain Kill ability handler.
func NewChainKillHandler() AbilityHandler { return &chainKillHandler{} }

type chainKillHandler struct {
	abilityHandlerBase
}

func (chainKillHandler) OnMoveStart(ctx MoveLifecycleContext) error {
	if ctx.Move == nil {
		return nil
	}
	// Chain Kill grants two additional captures beyond the base allowance.
	ctx.Move.increaseCaptureLimit(2)
	return nil
}

var _ AbilityHandler = (*chainKillHandler)(nil)
