// path: chessTest/internal/game/engine.go
package game

import "strings"

type MoveRequest struct {
	From         Square
	To           Square
	Dir          Direction
	Promotion    PieceType
	HasPromotion bool
}

type PieceState struct {
	ID        int
	Color     Color
	Type      PieceType
	Square    Square
	Abilities []string
}

type BoardState struct {
	Pieces      []PieceState
	Turn        Color
	LastNote    string
	Abilities   map[string][]string
	BlockFacing map[int]Direction
	Locked      bool
}

type Engine struct {
	board        boardSoA
	history      []boardSoA
	abilityLists [2]AbilityList
	abilityMask  [2]AbilitySet
	elements     [2]Element
	doOverUsed   [2]bool
	resolver     abilityResolver
	blockFacing  map[int]Direction
	locked       bool
	lastNote     string
}

func NewEngine() *Engine {
	eng := &Engine{
		board:       newBoard(),
		history:     make([]boardSoA, 0, 16),
		resolver:    newAbilityResolver(),
		blockFacing: make(map[int]Direction),
	}
	return eng
}

func (e *Engine) Reset() error {
	e.board = newBoard()
	e.history = e.history[:0]
	e.doOverUsed = [2]bool{}
	e.lastNote = ""
	e.locked = false
	for k := range e.blockFacing {
		delete(e.blockFacing, k)
	}
	for i := range e.abilityMask {
		e.board.addAbility(e.abilityMask[i], Color(i))
	}
	return nil
}

func (e *Engine) SetSideConfig(color Color, abilities AbilityList, element Element) error {
	if int(color) > 1 {
		return ErrInvalidConfig
	}
	normalized := normalizeAbilities(abilities)
	mask := NewAbilitySet(normalized...)
	e.abilityLists[color.Index()] = normalized
	e.abilityMask[color.Index()] = mask
	e.elements[color.Index()] = element
	e.board.addAbility(mask, color)
	e.doOverUsed[color.Index()] = false
	return nil
}

func (e *Engine) Move(req MoveRequest) error {
	if e.locked {
		return ErrEngineLocked
	}
	idx := e.board.pieceIndexBySquare(req.From)
	if idx < 0 {
		return ErrInvalidMove
	}
	if e.board.colors[idx] != e.board.turn {
		return ErrInvalidMove
	}
	if req.To == SquareInvalid {
		return ErrInvalidMove
	}
	if e.board.squareOccupiedBy(e.board.colors[idx], req.To) {
		return ErrInvalidMove
	}
	color := e.board.colors[idx]
	enemyColor := color.Opposite()
	captureIdx := e.board.pieceIndexBySquare(req.To)
	if err := e.validateMove(idx, req.To, captureIdx >= 0); err != nil {
		return err
	}
	prev := e.board.clone()
	e.history = append(e.history, prev)
	if captureIdx >= 0 {
		e.board.removePiece(captureIdx)
	}
	e.board.movePiece(idx, req.To)
	seed := uint64(e.board.ply)<<32 | uint64(e.board.ids[idx])<<16 | uint64(req.To)
	ctx := resolveContext{
		board:        &e.board,
		mover:        idx,
		target:       req.To,
		captureIdx:   captureIdx,
		sideMask:     e.abilityMask[color.Index()],
		enemyMask:    e.abilityMask[enemyColor.Index()],
		doOverUsed:   &e.doOverUsed,
		requestedDir: req.Dir,
		sideElement:  e.elements[color.Index()],
		enemyElement: e.elements[enemyColor.Index()],
		seed:         seed,
	}
	res, err := e.resolver.resolve(ctx)
	if err != nil {
		return err
	}
	if res.doOver {
		last := e.history[len(e.history)-1]
		e.board = last
		e.history = e.history[:len(e.history)-1]
		e.lastNote = "DoOver rewind"
		return ErrDoOverActivated
	}
	if res.setBlock {
		e.blockFacing[e.board.ids[idx]] = res.blockDir
	}
	e.board.turn = e.board.turn.Opposite()
	e.board.ply++
	e.lastNote = ""
	return nil
}

func (e *Engine) validateMove(idx int, to Square, isCapture bool) error {
	from := e.board.squares[idx]
	typ := e.board.types[idx]
	color := e.board.colors[idx]
	switch typ {
	case Pawn:
		if !e.validPawnMove(color, from, to, isCapture) {
			return ErrInvalidMove
		}
	default:
		return ErrInvalidMove
	}
	return nil
}

func (e *Engine) validPawnMove(color Color, from, to Square, isCapture bool) bool {
	fromRank := int(from) / 8
	fromFile := int(from) % 8
	toRank := int(to) / 8
	toFile := int(to) % 8
	dir := 1
	startRank := 1
	if color == Black {
		dir = -1
		startRank = 6
	}
	if isCapture {
		if (toRank - fromRank) != dir {
			return false
		}
		if abs(toFile-fromFile) != 1 {
			return false
		}
		if !e.board.squareOccupiedBy(color.Opposite(), to) {
			return false
		}
		return true
	}
	if toFile != fromFile {
		return false
	}
	if (toRank - fromRank) == dir {
		return e.board.empty(to)
	}
	if (toRank-fromRank) == 2*dir && fromRank == startRank {
		middle := Square(int(from) + dir*8)
		return e.board.empty(middle) && e.board.empty(to)
	}
	return false
}

func (e *Engine) State() BoardState {
	pieces := make([]PieceState, 0, len(e.board.ids))
	for i := range e.board.ids {
		if !e.board.alive[i] {
			continue
		}
		abilities := abilitySetToNames(e.board.ability[i])
		pieces = append(pieces, PieceState{
			ID:        e.board.ids[i],
			Color:     e.board.colors[i],
			Type:      e.board.types[i],
			Square:    e.board.squares[i],
			Abilities: abilities,
		})
	}
	abilityMap := map[string][]string{
		White.String(): abilityListToStrings(e.abilityLists[White.Index()]),
		Black.String(): abilityListToStrings(e.abilityLists[Black.Index()]),
	}
	blockCopy := make(map[int]Direction, len(e.blockFacing))
	for id, dir := range e.blockFacing {
		blockCopy[id] = dir
	}
	return BoardState{
		Pieces:      pieces,
		Turn:        e.board.turn,
		LastNote:    e.lastNote,
		Abilities:   abilityMap,
		BlockFacing: blockCopy,
		Locked:      e.locked,
	}
}

func abilitySetToNames(set AbilitySet) []string {
	if set == 0 {
		return nil
	}
	out := make([]string, 0, len(abilityCatalog))
	for _, entry := range abilityCatalog {
		if set.Has(entry.id) {
			out = append(out, entry.name)
		}
	}
	return out
}

func abilityListToStrings(list AbilityList) []string {
	out := make([]string, 0, len(list))
	for _, id := range list {
		if name, ok := abilityNameByID[id]; ok {
			out = append(out, name)
		}
	}
	return out
}

func normalizeAbilities(list AbilityList) AbilityList {
	if len(list) == 0 {
		return nil
	}
	seen := make(map[Ability]struct{}, len(list))
	out := make(AbilityList, 0, len(list))
	for _, id := range list {
		if id == AbilityNone {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func CoordToSquare(coord string) (Square, bool) {
	trimmed := strings.TrimSpace(strings.ToLower(coord))
	if len(trimmed) != 2 {
		return SquareInvalid, false
	}
	file := trimmed[0]
	rank := trimmed[1]
	if file < 'a' || file > 'h' {
		return SquareInvalid, false
	}
	if rank < '1' || rank > '8' {
		return SquareInvalid, false
	}
	idx := int(rank-'1')*8 + int(file-'a')
	return Square(idx), true
}

func SquareToCoord(sq Square) string {
	if sq == SquareInvalid {
		return ""
	}
	rank := int(sq) / 8
	file := int(sq) % 8
	return string([]byte{'a' + byte(file), '1' + byte(rank)})
}

func ParsePromotionPiece(s string) (PieceType, bool) {
	trimmed := strings.ToLower(strings.TrimSpace(s))
	switch trimmed {
	case "q", "queen":
		return Queen, true
	case "r", "rook":
		return Rook, true
	case "b", "bishop":
		return Bishop, true
	case "n", "knight":
		return Knight, true
	default:
		return Pawn, false
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
