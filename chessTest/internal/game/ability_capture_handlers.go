package game

import "fmt"

// NewDoubleKillHandler constructs the default Double Kill ability handler.
func NewDoubleKillHandler() AbilityHandler { return doubleKillHandler{} }

type doubleKillHandler struct {
	abilityHandlerBase
}

func (doubleKillHandler) ResolveCapture(ctx CaptureContext) (CaptureOutcome, error) {
	if ctx.Engine == nil || ctx.Attacker == nil || ctx.Victim == nil || ctx.Move == nil {
		return CaptureOutcome{}, nil
	}
	if ctx.Move.extraRemovalConsumed() {
		return CaptureOutcome{}, nil
	}
	attacker := ctx.Attacker
	target := ctx.Engine.trySmartExtraCapture(attacker, ctx.CaptureSquare, ctx.Victim.Color, rankOf(ctx.Victim.Type))
	if target == nil {
		return CaptureOutcome{}, nil
	}
	targetSquare := target.Square
	removed, err := ctx.Engine.attemptAbilityRemoval(attacker, target)
	if err != nil {
		return CaptureOutcome{}, err
	}
	if !removed {
		return CaptureOutcome{}, nil
	}
	appendAbilityNote(&ctx.Engine.board.lastNote, fmt.Sprintf("DoubleKill: removed %s %s at %s", target.Color, target.Type, targetSquare))
	ctx.Move.markExtraRemovalConsumed()
	return CaptureOutcome{}, nil
}

// NewScorchHandler constructs the default Scorch ability handler.
func NewScorchHandler() AbilityHandler { return scorchHandler{} }

type scorchHandler struct {
	abilityHandlerBase
}

func (scorchHandler) StepBudgetModifier(ctx StepBudgetContext) (StepBudgetDelta, error) {
	if ctx.Engine == nil || ctx.Piece == nil {
		return StepBudgetDelta{}, nil
	}
	pc := ctx.Piece
	if !pc.Abilities.Contains(AbilityScorch) {
		return StepBudgetDelta{}, nil
	}
	if elementOf(ctx.Engine, pc) != ElementFire {
		return StepBudgetDelta{}, nil
	}
	return StepBudgetDelta{AddSteps: 1, Notes: []string{"Scorch grants +1 step"}}, nil
}

func (scorchHandler) ResolveCapture(ctx CaptureContext) (CaptureOutcome, error) {
	if ctx.Engine == nil || ctx.Attacker == nil || ctx.Victim == nil || ctx.Move == nil {
		return CaptureOutcome{}, nil
	}
	if ctx.Move.extraRemovalConsumed() {
		return CaptureOutcome{}, nil
	}
	attacker := ctx.Attacker
	if elementOf(ctx.Engine, attacker) != ElementFire {
		return CaptureOutcome{}, nil
	}
	target := ctx.Engine.trySmartExtraCapture(attacker, ctx.CaptureSquare, ctx.Victim.Color, rankOf(ctx.Victim.Type))
	if target == nil {
		return CaptureOutcome{}, nil
	}
	targetSquare := target.Square
	removed, err := ctx.Engine.attemptAbilityRemoval(attacker, target)
	if err != nil {
		return CaptureOutcome{}, err
	}
	if !removed {
		return CaptureOutcome{}, nil
	}
	appendAbilityNote(&ctx.Engine.board.lastNote, fmt.Sprintf("Fire Scorch: removed %s %s at %s", target.Color, target.Type, targetSquare))
	ctx.Move.markExtraRemovalConsumed()
	return CaptureOutcome{}, nil
}

// NewQuantumKillHandler constructs the default Quantum Kill ability handler.
func NewQuantumKillHandler() AbilityHandler { return quantumKillHandler{} }

type quantumKillHandler struct {
	abilityHandlerBase
}

func (quantumKillHandler) ResolveCapture(ctx CaptureContext) (CaptureOutcome, error) {
	if ctx.Engine == nil || ctx.Attacker == nil || ctx.Victim == nil || ctx.Move == nil {
		return CaptureOutcome{}, nil
	}
	if ctx.Move.abilityUsed(AbilityQuantumKill) {
		return CaptureOutcome{}, nil
	}
	ctx.Move.markAbilityUsed(AbilityQuantumKill)
	attacker := ctx.Attacker
	victimColor := ctx.Victim.Color
	victimRank := rankOf(ctx.Victim.Type)
	target := ctx.Engine.findQuantumKillTarget(attacker, victimColor, victimRank)
	if target == nil {
		return CaptureOutcome{}, nil
	}
	targetSquare := target.Square
	removed, err := ctx.Engine.attemptAbilityRemoval(attacker, target)
	if err != nil {
		return CaptureOutcome{}, err
	}
	if !removed {
		return CaptureOutcome{}, nil
	}
	appendAbilityNote(&ctx.Engine.board.lastNote, fmt.Sprintf("Quantum Kill: removed %s %s at %s", target.Color, target.Type, targetSquare))
	if echo := ctx.Engine.trySmartExtraCapture(attacker, targetSquare, victimColor, rankOf(target.Type)); echo != nil {
		echoSquare := echo.Square
		removedEcho, err := ctx.Engine.attemptAbilityRemoval(attacker, echo)
		if err != nil {
			return CaptureOutcome{}, err
		}
		if removedEcho {
			appendAbilityNote(&ctx.Engine.board.lastNote, fmt.Sprintf("Quantum Echo: removed %s %s at %s", echo.Color, echo.Type, echoSquare))
		}
	}
	return CaptureOutcome{}, nil
}

// NewPoisonousMeatHandler constructs the default Poisonous Meat ability handler.
func NewPoisonousMeatHandler() AbilityHandler { return poisonousMeatHandler{} }

type poisonousMeatHandler struct {
	abilityHandlerBase
}

func (poisonousMeatHandler) ResolveCapture(ctx CaptureContext) (CaptureOutcome, error) {
	if ctx.Engine == nil || ctx.Attacker == nil || ctx.Move == nil {
		return CaptureOutcome{}, nil
	}
	attacker := ctx.Attacker
	if !attacker.Abilities.Contains(AbilityPoisonousMeat) {
		return CaptureOutcome{}, nil
	}
	outcome := CaptureOutcome{ForceTurnEnd: true}
	if elementOf(ctx.Engine, attacker) != ElementShadow && ctx.Move.RemainingSteps > 0 {
		outcome.StepAdjustment = -1
		appendAbilityNote(&ctx.Engine.board.lastNote, "Poisonous Meat drains 1 step")
	}
	appendAbilityNote(&ctx.Engine.board.lastNote, "Poisonous Meat ends the turn")
	return outcome, nil
}

// NewOverloadHandler constructs the default Overload ability handler.
func NewOverloadHandler() AbilityHandler { return overloadHandler{} }

type overloadHandler struct {
	abilityHandlerBase
}

func (overloadHandler) ResolveCapture(ctx CaptureContext) (CaptureOutcome, error) {
	if ctx.Engine == nil || ctx.Attacker == nil || ctx.Move == nil {
		return CaptureOutcome{}, nil
	}
	attacker := ctx.Attacker
	if !attacker.Abilities.Contains(AbilityOverload) {
		return CaptureOutcome{}, nil
	}
	element := elementOf(ctx.Engine, attacker)
	outcome := CaptureOutcome{}
	if attacker.Abilities.Contains(AbilityStalwart) && element == ElementLightning && ctx.Move.RemainingSteps > 0 {
		outcome.StepAdjustment = -1
		appendAbilityNote(&ctx.Engine.board.lastNote, "Overload + Stalwart costs 1 step")
	}
	if element == ElementLightning {
		outcome.ForceTurnEnd = true
		appendAbilityNote(&ctx.Engine.board.lastNote, "Overload ends the turn")
	}
	return outcome, nil
}

// NewBastionHandler constructs the default Bastion ability handler.
func NewBastionHandler() AbilityHandler { return bastionHandler{} }

type bastionHandler struct {
	abilityHandlerBase
}

func (bastionHandler) CanPhase(PhaseContext) (bool, error) {
	return false, ErrPhaseDenied
}

func (bastionHandler) ResolveCapture(ctx CaptureContext) (CaptureOutcome, error) {
	if ctx.Engine == nil || ctx.Attacker == nil {
		return CaptureOutcome{}, nil
	}
	attacker := ctx.Attacker
	if !attacker.Abilities.Contains(AbilityBastion) {
		return CaptureOutcome{}, nil
	}
	if elementOf(ctx.Engine, attacker) != ElementEarth {
		return CaptureOutcome{}, nil
	}
	appendAbilityNote(&ctx.Engine.board.lastNote, "Bastion ends the turn")
	return CaptureOutcome{ForceTurnEnd: true}, nil
}

// NewTemporalLockHandler constructs the default Temporal Lock ability handler.
func NewTemporalLockHandler() AbilityHandler { return temporalLockHandler{} }

type temporalLockHandler struct {
	abilityHandlerBase
}

func (temporalLockHandler) ResolveTurnEnd(ctx TurnEndContext) (TurnEndOutcome, error) {
	if ctx.Engine == nil || ctx.Move == nil || ctx.Move.Piece == nil {
		return TurnEndOutcome{}, nil
	}
	pc := ctx.Move.Piece
	if !pc.Abilities.Contains(AbilityTemporalLock) {
		return TurnEndOutcome{}, nil
	}
	slow := 1
	if elementOf(ctx.Engine, pc) == ElementFire {
		slow = 2
	}
	opponent := pc.Color.Opposite()
	outcome := TurnEndOutcome{}
	outcome.AddSlow(opponent, slow)
	outcome.Notes = append(outcome.Notes, fmt.Sprintf("Temporal Lock slows %s by %d", opponent, slow))
	return outcome, nil
}
