// path: chessTest/internal/game/game_status.go
package game

import "battle_chess_poc/internal/shared"

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
