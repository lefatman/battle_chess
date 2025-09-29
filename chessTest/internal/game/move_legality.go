// path: chessTest/internal/game/move_legality.go
package game

import "battle_chess_poc/internal/shared"

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

func (e *Engine) isLegalFirstSegment(pc *Piece, from, to Square) bool {
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

func (e *Engine) isLegalContinuation(pc *Piece, from, to Square) bool {
	if !e.isLegalFirstSegment(pc, from, to) {
		return false
	}
	return true
}
