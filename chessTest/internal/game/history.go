package game

// undoState captures a snapshot of the engine's state for history.
type undoState struct {
	board         Board
	blockFacing   map[int]Direction
	lastNote      string
	locked        bool
	configured    [2]bool
	pendingDoOver map[int]bool
	currentMove   *MoveState
	temporalSlow  [2]int
}

func cloneMoveState(src *MoveState, pieceMap map[*Piece]*Piece) *MoveState {
	if src == nil {
		return nil
	}
	clone := *src
	clone.Path = append([]Square(nil), src.Path...)
	clone.Captures = make([]*Piece, len(src.Captures))
	for i, captured := range src.Captures {
		clone.Captures[i] = mapPiecePointer(captured, pieceMap)
	}
	clone.Piece = mapPiecePointer(src.Piece, pieceMap)
	return &clone
}

func mapPiecePointer(pc *Piece, pieceMap map[*Piece]*Piece) *Piece {
	if pc == nil {
		return nil
	}
	if mapped, ok := pieceMap[pc]; ok {
		return mapped
	}
	return clonePiece(pc)
}

func cloneIntDirectionMap(src map[int]Direction) map[int]Direction {
	if len(src) == 0 {
		return make(map[int]Direction)
	}
	clone := make(map[int]Direction, len(src))
	for k, v := range src {
		clone[k] = v
	}
	return clone
}

func cloneIntBoolMap(src map[int]bool) map[int]bool {
	if len(src) == 0 {
		return make(map[int]bool)
	}
	clone := make(map[int]bool, len(src))
	for k, v := range src {
		clone[k] = v
	}
	return clone
}

func (e *Engine) snapshot() undoState {
	boardClone, mapping := e.board.cloneWithMap()
	var moveCopy *MoveState
	if e.currentMove != nil {
		moveCopy = cloneMoveState(e.currentMove, mapping)
	}
	s := undoState{
		board:         boardClone,
		blockFacing:   cloneIntDirectionMap(e.blockFacing),
		lastNote:      e.board.lastNote,
		locked:        e.locked,
		configured:    e.configured,
		pendingDoOver: cloneIntBoolMap(e.pendingDoOver),
		currentMove:   moveCopy,
		temporalSlow:  e.temporalSlow,
	}
	return s
}

func (e *Engine) applySnapshot(s undoState) {
	boardClone, mapping := s.board.cloneWithMap()
	e.board = boardClone
	e.blockFacing = cloneIntDirectionMap(s.blockFacing)
	e.board.lastNote = s.lastNote
	e.locked = s.locked
	e.configured = s.configured
	e.pendingDoOver = cloneIntBoolMap(s.pendingDoOver)
	e.temporalSlow = s.temporalSlow
	if s.currentMove != nil {
		e.currentMove = cloneMoveState(s.currentMove, mapping)
	} else {
		e.currentMove = nil
	}
}

func (e *Engine) pushHistory() { e.history = append(e.history, e.snapshot()) }

func (e *Engine) popHistory(n int) {
	for i := 0; i < n && len(e.history) > 0; i++ {
		idx := len(e.history) - 1
		s := e.history[idx]
		e.history = e.history[:idx]
		e.applySnapshot(s)
	}
}

func (b *Board) cloneWithMap() (Board, map[*Piece]*Piece) {
	out := *b
	mapping := make(map[*Piece]*Piece, len(b.pieceAt))
	for i, pc := range b.pieceAt {
		if pc != nil {
			copyPc := clonePiece(pc)
			out.pieceAt[i] = copyPc
			mapping[pc] = copyPc
		} else {
			out.pieceAt[i] = nil
		}
	}
	return out, mapping
}

func clonePiece(pc *Piece) *Piece {
	if pc == nil {
		return nil
	}
	clone := *pc
	clone.Abilities = pc.Abilities.Clone()
	return &clone
}
