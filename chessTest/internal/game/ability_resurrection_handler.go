package game

import "battle_chess_poc/internal/shared"

// NewResurrectionHandler constructs the default Resurrection ability handler.
func NewResurrectionHandler() AbilityHandler { return &resurrectionHandler{} }

type resurrectionHandler struct {
	abilityHandlerBase
}

func (resurrectionHandler) OnMoveStart(ctx MoveLifecycleContext) error {
	resetResurrectionState(ctx.Move)
	return nil
}

func (resurrectionHandler) OnCapture(ctx CaptureContext) error {
	if ctx.Move == nil || ctx.Attacker == nil {
		return nil
	}
	if ctx.Move.Piece != nil && ctx.Move.Piece != ctx.Attacker {
		return nil
	}
	ctx.Move.setAbilityFlag(AbilityResurrection, abilityFlagWindow, true)
	ctx.Move.setAbilityCounter(AbilityResurrection, abilityCounterResurrectionHold, 0)
	return nil
}

func (resurrectionHandler) OnSegmentStart(ctx SegmentContext) error {
	if ctx.Move == nil {
		return nil
	}
	if !ctx.Move.abilityFlag(AbilityResurrection, abilityFlagWindow) {
		return nil
	}
	if ctx.Move.abilityCounter(AbilityResurrection, abilityCounterResurrectionHold) != 0 {
		return nil
	}
	ctx.Move.setAbilityCounter(AbilityResurrection, abilityCounterResurrectionHold, 1)
	return nil
}

func (resurrectionHandler) OnSegmentResolved(ctx SegmentResolutionContext) error {
	if ctx.Move == nil {
		return nil
	}
	if ctx.Move.abilityCounter(AbilityResurrection, abilityCounterResurrectionHold) == 0 {
		return nil
	}
	ctx.Move.setAbilityCounter(AbilityResurrection, abilityCounterResurrectionHold, 0)
	ctx.Move.setAbilityFlag(AbilityResurrection, abilityFlagWindow, false)
	return nil
}

func (resurrectionHandler) OnTurnEnd(ctx TurnEndContext) error {
	resetResurrectionState(ctx.Move)
	return nil
}

func (resurrectionHandler) ResurrectionWindowActive(ctx ResurrectionContext) bool {
	if ctx.Move == nil || ctx.Piece == nil {
		return false
	}
	if ctx.Move.Piece != nil && ctx.Move.Piece != ctx.Piece {
		return false
	}
	return ctx.Move.abilityFlag(AbilityResurrection, abilityFlagWindow)
}

func (resurrectionHandler) AddResurrectionCaptureWindow(ctx ResurrectionContext, moves Bitboard) Bitboard {
	if !(resurrectionHandler{}).ResurrectionWindowActive(ctx) {
		return moves
	}
	if ctx.Engine == nil || ctx.Piece == nil {
		return moves
	}
	from := ctx.Piece.Square
	rank := from.Rank()
	file := from.File()
	for _, dr := range []int{-1, 1} {
		if target, ok := shared.SquareFromCoords(rank+dr, file); ok {
			occupant := ctx.Engine.board.pieceAt[target]
			if occupant != nil && occupant.Color != ctx.Piece.Color && ctx.Engine.canDirectCapture(ctx.Piece, occupant, from, target) {
				moves = moves.Add(target)
			}
		}
	}
	return moves
}

func resetResurrectionState(move *MoveState) {
	if move == nil {
		return
	}
	move.setAbilityFlag(AbilityResurrection, abilityFlagWindow, false)
	move.setAbilityCounter(AbilityResurrection, abilityCounterResurrectionHold, 0)
}

var (
	_ AbilityHandler                   = (*resurrectionHandler)(nil)
	_ ResurrectionWindowHandler        = (*resurrectionHandler)(nil)
	_ ResurrectionCaptureWindowHandler = (*resurrectionHandler)(nil)
)
