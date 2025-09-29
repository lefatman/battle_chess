// path: chessTest/internal/game/history.go
package game

// historyDelta captures the minimal state needed to undo a single move segment.
type historyDelta struct {
	squares       []squareDelta
	squareIndex   map[Square]int
	blockFacing   map[int]blockFacingDelta
	pendingDoOver map[int]pendingDoOverDelta
	lastNote      string
	castling      CastlingRights
	enPassant     EnPassantTarget
	turn          Color
	inCheck       bool
	gameOver      bool
	hasWinner     bool
	winner        Color
	status        string
	temporalSlow  [2]int
	currentMove   *MoveState
}

type squareDelta struct {
	square   Square
	hadPiece bool
	piece    *Piece
	snapshot pieceSnapshot
}

type pieceSnapshot struct {
	pieceData Piece
}

type blockFacingDelta struct {
	existed bool
	dir     Direction
}

type pendingDoOverDelta struct {
	existed bool
	value   bool
}

func newHistoryDelta(e *Engine) *historyDelta {
	d := &historyDelta{
		squareIndex:   make(map[Square]int),
		blockFacing:   make(map[int]blockFacingDelta),
		pendingDoOver: make(map[int]pendingDoOverDelta),
		lastNote:      e.board.lastNote,
		castling:      e.board.Castling,
		enPassant:     e.board.EnPassant,
		turn:          e.board.turn,
		inCheck:       e.board.InCheck,
		gameOver:      e.board.GameOver,
		hasWinner:     e.board.HasWinner,
		winner:        e.board.Winner,
		status:        e.board.Status,
		temporalSlow:  e.temporalSlow,
	}
	if e.currentMove != nil {
		d.currentMove = cloneMoveState(e.currentMove)
	}
	return d
}

func (d *historyDelta) recordSquare(e *Engine, sq Square) {
	if _, ok := d.squareIndex[sq]; ok {
		return
	}
	entry := squareDelta{square: sq}
	if pc := e.board.pieceAt[sq]; pc != nil {
		entry.hadPiece = true
		entry.piece = pc
		entry.snapshot = pieceSnapshot{pieceData: clonePieceState(pc)}
	}
	d.squareIndex[sq] = len(d.squares)
	d.squares = append(d.squares, entry)
}

func (d *historyDelta) recordBlockFacing(id int, dir Direction, existed bool) {
	if _, ok := d.blockFacing[id]; ok {
		return
	}
	d.blockFacing[id] = blockFacingDelta{existed: existed, dir: dir}
}

func (d *historyDelta) recordPendingDoOver(id int, value bool, existed bool) {
	if _, ok := d.pendingDoOver[id]; ok {
		return
	}
	d.pendingDoOver[id] = pendingDoOverDelta{existed: existed, value: value}
}

func (d *historyDelta) apply(e *Engine) {
	for _, entry := range d.squares {
		if cur := e.board.pieceAt[entry.square]; cur != nil {
			e.board.pieces[cur.Color][cur.Type] = e.board.pieces[cur.Color][cur.Type].Remove(entry.square)
			e.board.occupancy[cur.Color] = e.board.occupancy[cur.Color].Remove(entry.square)
			e.board.allOcc = e.board.allOcc.Remove(entry.square)
		}
		e.board.pieceAt[entry.square] = nil
	}
	for _, entry := range d.squares {
		if !entry.hadPiece || entry.piece == nil {
			continue
		}
		restored := entry.snapshot.pieceData
		restored.Abilities = restored.Abilities.Clone()
		restored.AbilityMask = restored.Abilities.Set()
		*entry.piece = restored
		e.board.pieceAt[entry.square] = entry.piece
		e.board.pieces[entry.piece.Color][entry.piece.Type] = e.board.pieces[entry.piece.Color][entry.piece.Type].Add(entry.square)
		e.board.occupancy[entry.piece.Color] = e.board.occupancy[entry.piece.Color].Add(entry.square)
		e.board.allOcc = e.board.allOcc.Add(entry.square)
	}

	e.board.Castling = d.castling
	e.board.EnPassant = d.enPassant
	e.board.turn = d.turn
	e.board.InCheck = d.inCheck
	e.board.GameOver = d.gameOver
	e.board.HasWinner = d.hasWinner
	e.board.Winner = d.winner
	e.board.Status = d.status
	e.board.lastNote = d.lastNote
	e.temporalSlow = d.temporalSlow

	for id, change := range d.blockFacing {
		if change.existed {
			e.blockFacing[id] = change.dir
		} else {
			delete(e.blockFacing, id)
		}
	}
	for id, change := range d.pendingDoOver {
		if change.existed {
			e.pendingDoOver[id] = change.value
		} else {
			delete(e.pendingDoOver, id)
		}
	}

	if d.currentMove != nil {
		e.currentMove = cloneMoveState(d.currentMove)
	} else {
		e.currentMove = nil
	}
	e.abilityCtx.clear()
}

func clonePieceState(pc *Piece) Piece {
	clone := *pc
	clone.Abilities = pc.Abilities.Clone()
	clone.AbilityMask = clone.Abilities.Set()
	return clone
}

func cloneMoveState(src *MoveState) *MoveState {
	if src == nil {
		return nil
	}
	clone := *src
	clone.Path = append([]Square(nil), src.Path...)
	clone.Captures = append([]*Piece(nil), src.Captures...)
	return &clone
}

func (e *Engine) pushHistory() *historyDelta {
	delta := newHistoryDelta(e)
	e.history = append(e.history, delta)
	e.activeDelta = delta
	return delta
}

func (e *Engine) finalizeHistory(delta *historyDelta) {
	if e.activeDelta == delta {
		e.activeDelta = nil
	}
}

func (e *Engine) recordSquareForUndo(sq Square) {
	if e.activeDelta == nil {
		return
	}
	e.activeDelta.recordSquare(e, sq)
}

func (e *Engine) recordBlockFacingForUndo(id int) {
	if e.activeDelta == nil {
		return
	}
	dir, ok := e.blockFacing[id]
	e.activeDelta.recordBlockFacing(id, dir, ok)
}

func (e *Engine) recordPendingDoOverForUndo(id int) {
	if e.activeDelta == nil {
		return
	}
	val, ok := e.pendingDoOver[id]
	e.activeDelta.recordPendingDoOver(id, val, ok)
}

func (e *Engine) popHistory(n int) {
	for i := 0; i < n && len(e.history) > 0; i++ {
		idx := len(e.history) - 1
		delta := e.history[idx]
		e.history = e.history[:idx]
		e.activeDelta = nil
		if delta != nil {
			delta.apply(e)
		}
	}
}
