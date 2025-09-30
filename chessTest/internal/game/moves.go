// path: chessTest/internal/game/moves.go
package game

import (
	"errors"
	"fmt"
)

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
	defer func() {
		if e.currentMove == nil {
			e.resetAbilityHandlers()
		}
	}()

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
			captureSq, ok := SquareFromCoords(captureRank, to.File())
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

	totalSteps, notes, err := e.calculateStepBudget(pc, handlers)
	if err != nil {
		return err
	}
	firstSegmentCost := e.calculateMovementCost(from, to)
	remainingSteps := totalSteps - firstSegmentCost
	if remainingSteps < 0 {
		remainingSteps = 0
	}

	e.currentMove = initializeMoveState(pc, from, remainingSteps, handlers, req.Promotion, req.HasPromotion)

	for _, note := range notes {
		if note == "" {
			continue
		}
		appendAbilityNote(&e.board.lastNote, note)
	}

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
	meta := segmentCtx.metadata()

	if err := e.dispatchMoveStartHandlers(req, meta); err != nil {
		e.currentMove = nil
		e.abilityCtx.clear()
		return err
	}
	if err := e.dispatchSegmentStartHandlers(from, to, meta, segmentStep); err != nil {
		e.currentMove = nil
		e.abilityCtx.clear()
		return err
	}

	e.executeMoveSegment(from, to, segmentCtx)
	e.currentMove.Path = append(e.currentMove.Path, to)
	if err := e.handlePostSegment(pc, from, to, meta); err != nil {
		e.currentMove = nil
		e.abilityCtx.clear()
		return err
	}
	if err := e.dispatchSegmentResolvedHandlers(from, to, meta, segmentStep, firstSegmentCost); err != nil {
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

	e.resolvePromotion(pc)
	if note := e.resolveBlockPathFacing(pc, req.Dir); note != "" {
		appendAbilityNote(&e.board.lastNote, note)
	}
	e.checkPostMoveAbilities(pc)
	e.checkPostMoveAbilities(pc)

	if e.checkPostCaptureTermination() {
		e.endTurn(TurnEndForced)
	} else if e.currentMove.RemainingSteps <= 0 && !e.hasFreeContinuation(pc) {
		e.endTurn(TurnEndNatural)
	} else {
		appendAbilityNote(&e.board.lastNote, fmt.Sprintf("%d steps remaining", e.currentMove.RemainingSteps))
	}

	return nil
}

func (e *Engine) continueMove(req MoveRequest) error {
	if e.currentMove == nil || e.currentMove.TurnEnded {
		return errors.New("no active move to continue")
	}

	defer func() {
		if e.currentMove == nil {
			e.resetAbilityHandlers()
		}
	}()

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

	if !e.isLegalContinuation(pc, from, to) {
		return errors.New("illegal move continuation")
	}

	stepsNeeded := e.calculateMovementCost(from, to)
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
			captureSq, ok := SquareFromCoords(captureRank, to.File())
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

	if len(e.handlersForAbility(AbilityResurrection)) == 0 && pc.HasAbility(AbilityResurrection) && e.currentMove.abilityFlag(AbilityResurrection, abilityFlagWindow) {
		e.currentMove.setAbilityFlag(AbilityResurrection, abilityFlagWindow, false)
		e.currentMove.setAbilityCounter(AbilityResurrection, abilityCounterResurrectionWindow, 0)
	}

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

	if e.checkPostCaptureTermination() {
		e.endTurn(TurnEndForced)
	} else if e.currentMove.RemainingSteps <= 0 && !e.hasFreeContinuation(pc) {
		e.endTurn(TurnEndNatural)
	} else {
		appendAbilityNote(&e.board.lastNote, fmt.Sprintf("%d steps remaining", e.currentMove.RemainingSteps))
	}

	return nil
}

func (e *Engine) endTurn(reason TurnEndReason) {
	if e.currentMove == nil {
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

	e.currentMove = nil
	e.abilityCtx.clear()
	e.resetAbilityHandlers()
}
