package game

import (
	"fmt"

	"battle_chess_poc/internal/shared"
)

// ResolveCaptureAbility orchestrates post-capture ability handling via
// registered ability handlers and returns the aggregated outcome.
func (e *Engine) ResolveCaptureAbility(attacker, victim *Piece, captureSquare Square, segmentStep int) (CaptureOutcome, error) {
	if victim == nil {
		return CaptureOutcome{}, nil
	}
	if err := e.maybeTriggerDoOver(victim); err != nil {
		return CaptureOutcome{}, err
	}

	if e.currentMove != nil {
		e.currentMove.resetExtraRemoval()
	}

	outcome, err := e.dispatchCaptureResolutionHandlers(attacker, victim, captureSquare, segmentStep)
	if err != nil {
		return outcome, err
	}
	return outcome, nil
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
	e.recordPendingDoOverForUndo(victim.ID)

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

func (e *Engine) dispatchCaptureResolutionHandlers(attacker, victim *Piece, square Square, step int) (CaptureOutcome, error) {
	if e.currentMove == nil || victim == nil {
		return CaptureOutcome{}, nil
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

	outcome := CaptureOutcome{}
	err := e.forEachActiveHandler(func(_ Ability, handler AbilityHandler) error {
		resolver, ok := handler.(CaptureResolutionHandler)
		if !ok {
			return nil
		}
		result, err := resolver.ResolveCapture(*ctx)
		if err != nil {
			return err
		}
		outcome = outcome.Merge(result)
		return nil
	})
	return outcome, err
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
