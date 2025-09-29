package game

import (
	"fmt"

	"battle_chess_poc/internal/shared"
)

// ResolveCaptureAbility handles special ability triggers on capture.
func (e *Engine) ResolveCaptureAbility(attacker, victim *Piece, captureSquare Square) error {
	if victim == nil {
		return nil
	}
	if err := e.maybeTriggerDoOver(victim); err != nil {
		return err
	}

	victimRank := rankOf(victim.Type)
	victimColor := victim.Color

	extraRemoved := false
	if attacker != nil && attacker.Abilities.Contains(AbilityDoubleKill) {
		if target := e.trySmartExtraCapture(attacker, captureSquare, victimColor, victimRank); target != nil {
			targetSquare := target.Square
			if removed, err := e.attemptAbilityRemoval(attacker, target); err != nil {
				return err
			} else if removed {
				appendAbilityNote(&e.board.lastNote, fmt.Sprintf("DoubleKill: removed %s %s at %s", target.Color, target.Type, targetSquare))
				extraRemoved = true
			}
		}
	}

	if !extraRemoved && attacker != nil && elementOf(e, attacker) == ElementFire && attacker.Abilities.Contains(AbilityScorch) {
		if target := e.trySmartExtraCapture(attacker, captureSquare, victimColor, victimRank); target != nil {
			targetSquare := target.Square
			if removed, err := e.attemptAbilityRemoval(attacker, target); err != nil {
				return err
			} else if removed {
				appendAbilityNote(&e.board.lastNote, fmt.Sprintf("Fire Scorch: removed %s %s at %s", target.Color, target.Type, targetSquare))
			}
		}
	}

	if attacker != nil && e.currentMove != nil && e.currentMove.HasQuantumKill && !e.currentMove.QuantumKillUsed {
		e.currentMove.QuantumKillUsed = true
		if target := e.findQuantumKillTarget(attacker, victimColor, victimRank); target != nil {
			targetSquare := target.Square
			if removed, err := e.attemptAbilityRemoval(attacker, target); err != nil {
				return err
			} else if removed {
				appendAbilityNote(&e.board.lastNote, fmt.Sprintf("Quantum Kill: removed %s %s at %s", target.Color, target.Type, targetSquare))
				if echo := e.trySmartExtraCapture(attacker, targetSquare, victimColor, rankOf(target.Type)); echo != nil {
					echoSquare := echo.Square
					if removedEcho, err := e.attemptAbilityRemoval(attacker, echo); err != nil {
						return err
					} else if removedEcho {
						appendAbilityNote(&e.board.lastNote, fmt.Sprintf("Quantum Echo: removed %s %s at %s", echo.Color, echo.Type, echoSquare))
					}
				}
			}
		}
	}

	e.applyCapturePenalties(attacker)
	return nil
}

func (e *Engine) trySmartExtraCapture(attacker *Piece, captureSquare Square, victimColor Color, victimRank int) *Piece {
	var bestP *Piece
	bestRank := -1

	for _, d := range [...]int{-9, -8, -7, -1, 1, 7, 8, 9} {
		nsqInt := int(captureSquare) + d
		if nsqInt < 0 || nsqInt >= 64 {
			continue
		}
		nsq := Square(nsqInt)
		if !shared.SameFileNeighborhood(captureSquare, nsq, d) {
			continue
		}
		p := e.board.pieceAt[nsq]
		if p == nil || p.Color != victimColor {
			continue
		}
		if !e.canAbilityRemove(attacker, p) {
			continue
		}
		if elementOf(e, p) == ElementEarth || p.Abilities.Contains(AbilityObstinant) || e.abilities[p.Color.Index()].Contains(AbilityObstinant) {
			continue
		}
		r := rankOf(p.Type)
		if r < victimRank && r > bestRank {
			bestRank = r
			bestP = p
		}
	}

	return bestP
}

func (e *Engine) maybeTriggerDoOver(victim *Piece) error {
	if victim == nil || !victim.Abilities.Contains(AbilityDoOver) || e.pendingDoOver[victim.ID] {
		return nil
	}

	plies := 4
	if plies > len(e.history) {
		plies = len(e.history)
	}
	if plies > 0 {
		e.popHistory(plies)
		victim.Abilities = victim.Abilities.Without(AbilityDoOver)
		e.pendingDoOver[victim.ID] = true
		e.board.lastNote = fmt.Sprintf("DoOver: %s %s rewound %d plies (%.1f turns)", victim.Color, victim.Type, plies, float64(plies)/2.0)
		return ErrDoOverActivated
	}

	victim.Abilities = victim.Abilities.Without(AbilityDoOver)
	e.pendingDoOver[victim.ID] = true
	return nil
}

func (e *Engine) canAbilityRemove(attacker, target *Piece) bool {
	if target == nil {
		return false
	}
	if target.Type == King {
		return false
	}
	if target.Abilities.Contains(AbilityIndomitable) {
		return false
	}
	if target.Abilities.Contains(AbilityStalwart) && attacker != nil && rankOf(attacker.Type) < rankOf(target.Type) {
		return false
	}
	if target.Abilities.Contains(AbilityBelligerent) && attacker != nil && rankOf(attacker.Type) > rankOf(target.Type) {
		return false
	}
	return true
}

func (e *Engine) attemptAbilityRemoval(attacker, target *Piece) (bool, error) {
	if target == nil || !e.canAbilityRemove(attacker, target) {
		return false, nil
	}

	e.removePiece(target, target.Square)
	if err := e.maybeTriggerDoOver(target); err != nil {
		return true, err
	}
	return true, nil
}

func (e *Engine) findQuantumKillTarget(attacker *Piece, victimColor Color, maxRank int) *Piece {
	var best *Piece
	bestRank := -1
	bestIndex := 65

	for idx, p := range e.board.pieceAt {
		if p == nil || p.Color != victimColor {
			continue
		}
		rank := rankOf(p.Type)
		if rank > maxRank {
			continue
		}
		if !e.canAbilityRemove(attacker, p) {
			continue
		}
		if rank > bestRank || (rank == bestRank && idx < bestIndex) {
			best = p
			bestRank = rank
			bestIndex = idx
		}
	}

	return best
}

func (e *Engine) applyCapturePenalties(attacker *Piece) {
	if e.currentMove == nil || attacker == nil {
		return
	}

	element := elementOf(e, attacker)
	if attacker.Abilities.Contains(AbilityPoisonousMeat) && element != ElementShadow {
		if e.currentMove.RemainingSteps > 0 {
			e.currentMove.RemainingSteps--
			appendAbilityNote(&e.board.lastNote, "Poisonous Meat drains 1 step")
		}
	}

	if attacker.Abilities.Contains(AbilityOverload) && element == ElementLightning && attacker.Abilities.Contains(AbilityStalwart) {
		if e.currentMove.RemainingSteps > 0 {
			e.currentMove.RemainingSteps--
			appendAbilityNote(&e.board.lastNote, "Overload + Stalwart costs 1 step")
		}
	}
}

func (e *Engine) captureBlockedByBlockPath(attacker *Piece, from Square, defender *Piece, to Square) (bool, string) {
	if defender == nil || !defender.Abilities.Contains(AbilityBlockPath) || defender.BlockDir == DirNone {
		return false, ""
	}
	if attacker != nil && elementOf(e, attacker) == ElementWater {
		return false, ""
	}
	dir := shared.DirectionOf(from, to)
	if dir == defender.BlockDir {
		return true, fmt.Sprintf("Capture blocked by BlockPath (%s)", defender.BlockDir)
	}
	return false, ""
}
