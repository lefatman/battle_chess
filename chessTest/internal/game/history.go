// path: chessTest/internal/game/history.go
package game

// historyDelta captures the minimal state needed to undo a single move segment.
type historyDelta struct {
	squares            []squareDelta
	squareIndex        [64]int16
	blockFacingIndex   []int32
	blockFacing        []blockFacingDelta
	pendingDoOverIndex []int32
	pendingDoOver      []pendingDoOverDelta
	lastNote           string
	castling           CastlingRights
	enPassant          EnPassantTarget
	turn               Color
	inCheck            bool
	gameOver           bool
	hasWinner          bool
	winner             Color
	status             string
	temporalSlow       [2]int
	currentMove        *MoveState
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
	id      int
	existed bool
	dir     Direction
}

type pendingDoOverDelta struct {
	id      int
	existed bool
	value   bool
}

const (
	historySquareUnset   int16 = -1
	historyPieceIdxUnset int32 = -1

	historySquareCap     = 8
	historyPieceDeltaCap = 4
)

func newHistoryDelta(e *Engine) *historyDelta {
	d := &historyDelta{
		squares:            make([]squareDelta, 0, historySquareCap),
		blockFacingIndex:   make([]int32, e.nextPieceID),
		blockFacing:        make([]blockFacingDelta, 0, historyPieceDeltaCap),
		pendingDoOverIndex: make([]int32, e.nextPieceID),
		pendingDoOver:      make([]pendingDoOverDelta, 0, historyPieceDeltaCap),
		lastNote:           e.board.lastNote,
		castling:           e.board.Castling,
		enPassant:          e.board.EnPassant,
		turn:               e.board.turn,
		inCheck:            e.board.InCheck,
		gameOver:           e.board.GameOver,
		hasWinner:          e.board.HasWinner,
		winner:             e.board.Winner,
		status:             e.board.Status,
		temporalSlow:       e.temporalSlow,
	}
	for i := range d.squareIndex {
		d.squareIndex[i] = historySquareUnset
	}
	for i := range d.blockFacingIndex {
		d.blockFacingIndex[i] = historyPieceIdxUnset
	}
	for i := range d.pendingDoOverIndex {
		d.pendingDoOverIndex[i] = historyPieceIdxUnset
	}
	if e.currentMove != nil {
		d.currentMove = cloneMoveState(e.currentMove)
	}
	return d
}

func (d *historyDelta) recordSquare(e *Engine, sq Square) {
	idx := d.squareIndex[sq]
	if idx != historySquareUnset {
		return
	}
	entry := squareDelta{square: sq}
	if pc := e.board.pieceAt[sq]; pc != nil {
		entry.hadPiece = true
		entry.piece = pc
		entry.snapshot = pieceSnapshot{pieceData: clonePieceState(pc)}
	}
	d.squareIndex[sq] = int16(len(d.squares))
	d.squares = append(d.squares, entry)
}

func (d *historyDelta) recordBlockFacing(id int, dir Direction, existed bool) {
	if id < 0 {
		return
	}
	d.ensureBlockFacingCapacity(id)
	if d.blockFacingIndex[id] != historyPieceIdxUnset {
		return
	}
	d.blockFacingIndex[id] = int32(len(d.blockFacing))
	d.blockFacing = append(d.blockFacing, blockFacingDelta{id: id, existed: existed, dir: dir})
}

func (d *historyDelta) recordPendingDoOver(id int, value bool, existed bool) {
	if id < 0 {
		return
	}
	d.ensurePendingDoOverCapacity(id)
	if d.pendingDoOverIndex[id] != historyPieceIdxUnset {
		return
	}
	d.pendingDoOverIndex[id] = int32(len(d.pendingDoOver))
	d.pendingDoOver = append(d.pendingDoOver, pendingDoOverDelta{id: id, existed: existed, value: value})
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

	for _, change := range d.blockFacing {
		if change.existed {
			e.blockFacing[change.id] = change.dir
		} else {
			delete(e.blockFacing, change.id)
		}
	}
	for _, change := range d.pendingDoOver {
		if change.existed {
			e.pendingDoOver[change.id] = change.value
		} else {
			delete(e.pendingDoOver, change.id)
		}
	}

	if d.currentMove != nil {
		e.currentMove = cloneMoveState(d.currentMove)
	} else {
		e.currentMove = nil
	}
	e.abilityCtx.clear()
	d.resetIndexes()
}

func (d *historyDelta) ensureBlockFacingCapacity(id int) {
	if id < len(d.blockFacingIndex) {
		return
	}
	need := id + 1
	old := len(d.blockFacingIndex)
	d.blockFacingIndex = append(d.blockFacingIndex, make([]int32, need-old)...)
	for i := old; i < len(d.blockFacingIndex); i++ {
		d.blockFacingIndex[i] = historyPieceIdxUnset
	}
}

func (d *historyDelta) ensurePendingDoOverCapacity(id int) {
	if id < len(d.pendingDoOverIndex) {
		return
	}
	need := id + 1
	old := len(d.pendingDoOverIndex)
	d.pendingDoOverIndex = append(d.pendingDoOverIndex, make([]int32, need-old)...)
	for i := old; i < len(d.pendingDoOverIndex); i++ {
		d.pendingDoOverIndex[i] = historyPieceIdxUnset
	}
}

func (d *historyDelta) resetIndexes() {
	for _, entry := range d.squares {
		d.squareIndex[entry.square] = historySquareUnset
	}
	for _, change := range d.blockFacing {
		if change.id >= 0 && change.id < len(d.blockFacingIndex) {
			d.blockFacingIndex[change.id] = historyPieceIdxUnset
		}
	}
	for _, change := range d.pendingDoOver {
		if change.id >= 0 && change.id < len(d.pendingDoOverIndex) {
			d.pendingDoOverIndex[change.id] = historyPieceIdxUnset
		}
	}
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
