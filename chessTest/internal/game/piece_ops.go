package game

import "battle_chess_poc/internal/shared"

func appendAbilityNote(dst *string, note string) {
	if *dst == "" || *dst == "New game" || *dst == "Configuration locked - game started" {
		*dst = note
	} else {
		*dst += "; " + note
	}
}

type moveSegmentContext struct {
	capture       *Piece
	captureSquare Square
	enPassant     bool
}

func (e *Engine) executeMoveSegment(from, to Square, ctx moveSegmentContext) {
	pc := e.board.pieceAt[from]
	if pc == nil {
		return
	}

	isCastle := pc.Type == King && from.Rank() == to.Rank() && absInt(to.File()-from.File()) == 2

	if ctx.capture != nil {
		e.updateCastlingRightsForCapture(ctx.capture, ctx.captureSquare)
		e.removePiece(ctx.capture, ctx.captureSquare)
		if ctx.enPassant {
			appendAbilityNote(&e.board.lastNote, "En passant capture")
		}
	}

	e.board.EnPassant = NoEnPassantTarget()

	e.board.pieceAt[from] = nil
	pc.Square = to
	e.board.pieceAt[to] = pc

	e.board.pieces[pc.Color][pc.Type] = e.board.pieces[pc.Color][pc.Type].Remove(from).Add(to)
	e.board.occupancy[pc.Color] = e.board.occupancy[pc.Color].Remove(from).Add(to)
	e.board.allOcc = e.board.allOcc.Remove(from).Add(to)

	e.updateCastlingRightsForMove(pc, from)

	if pc.Type == Pawn {
		diff := to.Rank() - from.Rank()
		if diff == 2 || diff == -2 {
			midRank := from.Rank() + diff/2
			if sq, ok := shared.SquareFromCoords(midRank, from.File()); ok {
				e.board.EnPassant = NewEnPassantTarget(sq)
			}
		}
	}

	if isCastle {
		e.performCastleRookMove(pc.Color, from, to)
	}
}

func (e *Engine) placePiece(color Color, pt PieceType, sq Square) {
	id := e.nextPieceID
	e.nextPieceID++
	pc := &Piece{
		ID:        id,
		Color:     color,
		Type:      pt,
		Square:    sq,
		Abilities: e.abilities[color].Clone(),
		Element:   e.elements[color],
		BlockDir:  DirNone,
	}
	e.board.pieceAt[sq] = pc
	e.board.pieces[color][pt] = e.board.pieces[color][pt].Add(sq)
	e.board.occupancy[color] = e.board.occupancy[color].Add(sq)
	e.board.allOcc = e.board.allOcc.Add(sq)
}

func castlingRightForRook(color Color, sq Square) CastlingRights {
	switch color {
	case White:
		if sq.Rank() != 0 {
			return CastlingNone
		}
		switch sq.File() {
		case 0:
			return CastlingRight(White, CastleQueenside)
		case 7:
			return CastlingRight(White, CastleKingside)
		}
	case Black:
		if sq.Rank() != 7 {
			return CastlingNone
		}
		switch sq.File() {
		case 0:
			return CastlingRight(Black, CastleQueenside)
		case 7:
			return CastlingRight(Black, CastleKingside)
		}
	}
	return CastlingNone
}

func (e *Engine) updateCastlingRightsForMove(pc *Piece, from Square) {
	if pc == nil {
		return
	}
	switch pc.Type {
	case King:
		e.board.Castling = e.board.Castling.WithoutColor(pc.Color)
	case Rook:
		e.board.Castling = e.board.Castling.Without(castlingRightForRook(pc.Color, from))
	}
}

func (e *Engine) updateCastlingRightsForCapture(pc *Piece, sq Square) {
	if pc == nil {
		return
	}
	switch pc.Type {
	case King:
		e.board.Castling = e.board.Castling.WithoutColor(pc.Color)
	case Rook:
		e.board.Castling = e.board.Castling.Without(castlingRightForRook(pc.Color, sq))
	}
}

func (e *Engine) performCastleRookMove(color Color, from, to Square) {
	rank := from.Rank()
	var rookFromFile, rookToFile int
	var note string
	if to.File() > from.File() {
		rookFromFile = 7
		rookToFile = to.File() - 1
		note = "Castled kingside"
	} else {
		rookFromFile = 0
		rookToFile = to.File() + 1
		note = "Castled queenside"
	}
	rookFrom, okFrom := shared.SquareFromCoords(rank, rookFromFile)
	rookTo, okTo := shared.SquareFromCoords(rank, rookToFile)
	if !okFrom || !okTo {
		return
	}
	rook := e.board.pieceAt[rookFrom]
	if rook == nil || rook.Type != Rook || rook.Color != color {
		return
	}

	e.board.pieceAt[rookFrom] = nil
	rook.Square = rookTo
	e.board.pieceAt[rookTo] = rook

	e.board.pieces[color][Rook] = e.board.pieces[color][Rook].Remove(rookFrom).Add(rookTo)
	e.board.occupancy[color] = e.board.occupancy[color].Remove(rookFrom).Add(rookTo)
	e.board.allOcc = e.board.allOcc.Remove(rookFrom).Add(rookTo)

	appendAbilityNote(&e.board.lastNote, note)
}

func (e *Engine) removePiece(pc *Piece, sq Square) {
	e.board.pieces[pc.Color][pc.Type] = e.board.pieces[pc.Color][pc.Type].Remove(sq)
	e.board.occupancy[pc.Color] = e.board.occupancy[pc.Color].Remove(sq)
	e.board.allOcc = e.board.allOcc.Remove(sq)
	e.board.pieceAt[sq] = nil
	delete(e.blockFacing, pc.ID)
	delete(e.pendingDoOver, pc.ID)
}

func (e *Engine) isPathClear(from, to Square) bool {
	line := shared.Line(from, to)
	for _, sq := range line {
		if e.board.pieceAt[sq] != nil {
			return false
		}
	}
	return true
}

func (e *Engine) canPhaseThrough(pc *Piece, _ Square, _ Square) bool {
	if pc == nil {
		return false
	}

	hasFlood := pc.Abilities.Contains(AbilityFloodWake)
	hasBastion := pc.Abilities.Contains(AbilityBastion)
	hasGale := pc.Abilities.Contains(AbilityGaleLift)

	if e.abilities != nil {
		if al, ok := e.abilities[pc.Color]; ok {
			if al.Contains(AbilityFloodWake) {
				hasFlood = true
			}
			if al.Contains(AbilityBastion) {
				hasBastion = true
			}
			if al.Contains(AbilityGaleLift) {
				hasGale = true
			}
		}
	}

	if hasFlood || hasBastion {
		return false
	}
	if hasGale {
		return true
	}
	if pc.Abilities.Contains(AbilityUmbralStep) {
		return true
	}
	if e.abilities != nil {
		if al, ok := e.abilities[pc.Color]; ok && al.Contains(AbilityUmbralStep) {
			return true
		}
	}
	return false
}
