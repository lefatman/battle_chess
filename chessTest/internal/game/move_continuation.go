// path: chessTest/internal/game/move_continuation.go
package game

import (
	"errors"
	"fmt"

	"battle_chess_poc/internal/shared"
)

func (e *Engine) trySideStepNudge(pc *Piece, from, to Square) (bool, error) {
	if e.currentMove == nil || pc == nil {
		return false, nil
	}

	handlers := e.handlersForAbility(AbilitySideStep)
	if len(handlers) == 0 {
		return false, nil
	}

	segmentStep := len(e.currentMove.Path) - 1
	if segmentStep < 0 {
		segmentStep = 0
	}

	ctx := &e.abilityCtx.segment
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

	return false, nil
}

func (e *Engine) tryQuantumStep(pc *Piece, from, to Square) (bool, error) {
	if e.currentMove == nil || pc == nil {
		return false, nil
	}
	handlers := e.handlersForAbility(AbilityQuantumStep)
	if len(handlers) == 0 {
		return false, nil
	}

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

	return false, nil
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

func (e *Engine) wouldChangeDirection(move *MoveState, from, to Square) bool {
	if move == nil || move.Piece == nil {
		return false
	}
	if !e.isSlider(move.Piece.Type) {
		return false
	}
	if len(move.Path) <= 1 {
		return false
	}
	prevFrom := move.Path[len(move.Path)-2]
	prevDir := shared.DirectionOf(prevFrom, from)
	currentDir := shared.DirectionOf(from, to)
	if prevDir == DirNone || currentDir == DirNone {
		return false
	}
	return prevDir != currentDir
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

func (e *Engine) checkPostMoveAbilities(pc *Piece) {
	if pc.Abilities.Contains(AbilitySideStep) && !e.currentMove.abilityUsed(AbilitySideStep) && e.currentMove.RemainingSteps > 0 {
		appendAbilityNote(&e.board.lastNote, "Side Step available (costs 1 step)")
	}
	if pc.Abilities.Contains(AbilityQuantumStep) && !e.currentMove.abilityUsed(AbilityQuantumStep) && e.currentMove.RemainingSteps > 0 {
		appendAbilityNote(&e.board.lastNote, "Quantum Step available (costs 1 step)")
	}
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
