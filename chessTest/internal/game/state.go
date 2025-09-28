package game

import (
	"errors"
	"fmt"

	"battle_chess_poc/internal/shared"
)

// ---------------------------
// Core Engine & State Structs
// ---------------------------

// Engine encapsulates the optimized chess engine with ability metadata.
type Engine struct {
	board         Board
	abilities     map[Color]AbilityList
	elements      map[Color]Element
	blockFacing   map[int]Direction
	history       []undoState
	nextPieceID   int
	locked        bool
	configured    map[Color]bool
	pendingDoOver map[int]bool // Tracks per-piece DoOver consumption
	currentMove   *MoveState
}

// undoState captures a snapshot of the engine's state for history.
type undoState struct {
	board         Board
	blockFacing   map[int]Direction
	lastNote      string
	locked        bool
	configured    map[Color]bool
	pendingDoOver map[int]bool
	currentMove   *MoveState
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

func cloneColorBoolMap(src map[Color]bool) map[Color]bool {
	if len(src) == 0 {
		return make(map[Color]bool)
	}
	clone := make(map[Color]bool, len(src))
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

// Board represents the state of the chessboard.
type Board struct {
	pieces    [2][6]Bitboard
	occupancy [2]Bitboard
	allOcc    Bitboard
	pieceAt   [64]*Piece
	turn      Color
	lastNote  string
}

// Piece represents a single piece on the board.
type Piece struct {
	ID        int
	Color     Color
	Type      PieceType
	Square    Square
	Abilities AbilityList
	Element   Element
	BlockDir  Direction
}

// MoveRequest is passed in by an external layer to request a move.
type MoveRequest struct {
	From Square
	To   Square
	Dir  Direction // Chosen BlockPath direction after move
}

// PieceState is a serializable representation of a Piece.
type PieceState struct {
	ID           int         `json:"id"`
	Color        Color       `json:"color"`
	ColorName    string      `json:"colorName"`
	Type         PieceType   `json:"type"`
	TypeName     string      `json:"typeName"`
	Square       Square      `json:"square"`
	Abilities    AbilityList `json:"abilities"`
	AbilityNames []string    `json:"abilityNames"`
	Element      Element     `json:"element"`
	ElementName  string      `json:"elementName"`
	BlockDir     Direction   `json:"blockDir"`
}

// BoardState is a serializable representation of the game state.
type BoardState struct {
	Pieces      []PieceState        `json:"pieces"`
	Turn        Color               `json:"turn"`
	TurnName    string              `json:"turnName"`
	LastNote    string              `json:"lastNote"`
	Abilities   map[string][]string `json:"abilities"`
	Elements    map[string]string   `json:"elements"`
	BlockFacing map[int]Direction   `json:"blockFacing"`
}

// ---------------------------
// Public API
// ---------------------------

// NewEngine creates and initializes a new game engine.
func NewEngine() *Engine {
	eng := &Engine{
		abilities: map[Color]AbilityList{
			White: nil,
			Black: nil,
		},
		elements: map[Color]Element{
			White: ElementLight,
			Black: ElementShadow,
		},
		blockFacing:   make(map[int]Direction),
		configured:    make(map[Color]bool, 2),
		pendingDoOver: make(map[int]bool),
		currentMove:   nil,
	}
	if err := eng.Reset(); err != nil {
		panic(err) // Should not happen on initial setup
	}
	return eng
}

// Reset clears the engine state and sets up a standard new game.
func (e *Engine) Reset() error {
	e.board = Board{}
	e.blockFacing = make(map[int]Direction)
	e.history = e.history[:0]
	e.nextPieceID = 1
	e.locked = false
	e.currentMove = nil
	if e.configured == nil {
		e.configured = make(map[Color]bool, 2)
	} else {
		for k := range e.configured {
			e.configured[k] = false
		}
	}
	if e.pendingDoOver == nil {
		e.pendingDoOver = make(map[int]bool)
	} else {
		for k := range e.pendingDoOver {
			delete(e.pendingDoOver, k)
		}
	}

	setup := func(color Color, backRank, pawnRank int) {
		order := []PieceType{Rook, Knight, Bishop, Queen, King, Bishop, Knight, Rook}
		for file, pt := range order {
			sq := Square(backRank*8 + file)
			e.placePiece(color, pt, sq)
		}
		for file := 0; file < 8; file++ {
			sq := Square(pawnRank*8 + file)
			e.placePiece(color, Pawn, sq)
		}
	}

	setup(Black, 7, 6)
	setup(White, 0, 1)
	e.board.turn = White
	e.board.lastNote = "New game"
	return nil
}

// SetSideConfig chooses ability and element per side before the match starts.
func (e *Engine) SetSideConfig(color Color, abilities AbilityList, element Element) error {
	if len(abilities) == 0 {
		return errors.New("abilities cannot be empty")
	}
	if e.locked {
		return errors.New("configuration locked after game start")
	}
	if e.abilities == nil {
		e.abilities = make(map[Color]AbilityList, 2)
	}
	e.abilities[color] = abilities.Clone()
	e.elements[color] = element
	for _, pc := range e.board.pieceAt {
		if pc != nil && pc.Color == color {
			pc.Abilities = abilities.Clone()
			pc.Element = element
			if !pc.Abilities.Contains(AbilityBlockPath) {
				delete(e.blockFacing, pc.ID)
				pc.BlockDir = DirNone
			} else {
				if dir, ok := e.blockFacing[pc.ID]; ok {
					pc.BlockDir = dir
				} else {
					pc.BlockDir = DirNone
				}
			}
		}
	}
	e.configured[color] = true
	e.board.lastNote = fmt.Sprintf("Configured %s: abilities=%v element=%s", color, abilities.Strings(), element)
	return nil
}

// Move is the primary entry point for making a move. It delegates to start or continue a move.
func (e *Engine) Move(req MoveRequest) error {
	if !e.locked {
		if err := e.ensureConfigured(); err != nil {
			return err
		}
		if err := e.lockConfiguration(); err != nil {
			return err
		}
	}

	if e.currentMove == nil {
		return e.startNewMove(req)
	}
	return e.continueMove(req)
}

// State returns a serializable representation of the current game state.
func (e *Engine) State() BoardState {
	state := BoardState{
		Pieces:      make([]PieceState, 0, 32),
		Turn:        e.board.turn,
		TurnName:    e.board.turn.String(),
		LastNote:    e.board.lastNote,
		Abilities:   make(map[string][]string),
		Elements:    make(map[string]string),
		BlockFacing: make(map[int]Direction),
	}

	for _, pc := range e.board.pieceAt {
		if pc != nil {
			state.Pieces = append(state.Pieces, PieceState{
				ID:           pc.ID,
				Color:        pc.Color,
				ColorName:    pc.Color.String(),
				Type:         pc.Type,
				TypeName:     pc.Type.String(),
				Square:       pc.Square,
				Abilities:    pc.Abilities.Clone(),
				AbilityNames: pc.Abilities.Strings(),
				Element:      pc.Element,
				ElementName:  pc.Element.String(),
				BlockDir:     pc.BlockDir,
			})
		}
	}

	for color, abilities := range e.abilities {
		state.Abilities[color.String()] = abilities.Strings()
	}
	for color, element := range e.elements {
		state.Elements[color.String()] = element.String()
	}
	for id, dir := range e.blockFacing {
		state.BlockFacing[id] = dir
	}

	return state
}

// ---------------------------
// Move Lifecycle Management
// ---------------------------

// ---------------------------
// Ability Resolution Pipeline
// ---------------------------

// ResolveCaptureAbility handles special ability triggers on capture.
func (e *Engine) ResolveCaptureAbility(attacker, victim *Piece, captureSquare Square) error {
	if victim != nil && victim.Abilities.Contains(AbilityDoOver) && !e.pendingDoOver[victim.ID] {
		plies := 4
		if plies > len(e.history) {
			plies = len(e.history)
		}
		if plies > 0 {
			e.popHistory(plies)
			victim.Abilities = victim.Abilities.Without(AbilityDoOver)
			e.pendingDoOver[victim.ID] = true
			e.board.lastNote = fmt.Sprintf("DoOver: %s %s rewound %d plies (%.1f turns)",
				victim.Color, victim.Type, plies, float64(plies)/2.0)
			return ErrDoOverActivated
		}
		victim.Abilities = victim.Abilities.Without(AbilityDoOver)
		e.pendingDoOver[victim.ID] = true
	}

	extraRemoved := false
	if attacker != nil && attacker.Abilities.Contains(AbilityDoubleKill) {
		if note, removed := e.trySmartExtraCapture(captureSquare, victim.Color, rankOf(victim.Type)); removed {
			extraRemoved = true
			appendAbilityNote(&e.board.lastNote, "DoubleKill: "+note)
		}
	}

	if !extraRemoved && attacker != nil {
		if elementOf(e, attacker) == ElementFire && attacker.Abilities.Contains(AbilityScorch) {
			if note, removed := e.trySmartExtraCapture(captureSquare, victim.Color, rankOf(victim.Type)); removed {
				appendAbilityNote(&e.board.lastNote, "Fire Scorch: "+note)
			}
		}
	}
	return nil
}

// trySmartExtraCapture scans neighbors and removes the highest-rank, eligible, lower-rank piece.
func (e *Engine) trySmartExtraCapture(captureSquare Square, victimColor Color, victimRank int) (string, bool) {
	var bestP *Piece
	bestRank := -1

	for _, d := range [...]int{-9, -8, -7, -1, 1, 7, 8, 9} {
		nsqInt := int(captureSquare) + d
		if nsqInt < 0 || nsqInt >= 64 {
			continue
		}
		nsq := Square(nsqInt)
		if !shared.SameFileNeighborhood(captureSquare, nsq, d) {
			continue
		}
		p := e.board.pieceAt[nsq]
		if p == nil || p.Color != victimColor {
			continue
		}
		if elementOf(e, p) == ElementEarth || p.Abilities.Contains(AbilityObstinant) || e.abilities[p.Color].Contains(AbilityObstinant) {
			continue
		}
		r := rankOf(p.Type)
		if r < victimRank && r > bestRank {
			bestRank = r
			bestP = p
		}
	}

	if bestP == nil {
		return "", false
	}

	e.removePiece(bestP, bestP.Square)
	return fmt.Sprintf("removed %s %s at %s", bestP.Color, bestP.Type, bestP.Square), true
}

// ---------------------------
// Step, Cost & State Calculation
// ---------------------------

// ---------------------------
// History & State Management
// ---------------------------

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
		configured:    cloneColorBoolMap(e.configured),
		pendingDoOver: cloneIntBoolMap(e.pendingDoOver),
		currentMove:   moveCopy,
	}
	return s
}

func (e *Engine) applySnapshot(s undoState) {
	boardClone, mapping := s.board.cloneWithMap()
	e.board = boardClone
	e.blockFacing = cloneIntDirectionMap(s.blockFacing)
	e.board.lastNote = s.lastNote
	e.locked = s.locked
	e.configured = cloneColorBoolMap(s.configured)
	e.pendingDoOver = cloneIntBoolMap(s.pendingDoOver)
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

// ---------------------------
// Legality & Validation Helpers
// ---------------------------

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

// ---------------------------
// General Helpers
// ---------------------------

func appendAbilityNote(dst *string, note string) {
	if *dst == "" || *dst == "New game" || *dst == "Configuration locked - game started" {
		*dst = note
	} else {
		*dst += "; " + note
	}
}

func (e *Engine) executeMoveSegment(from, to Square) {
	pc := e.board.pieceAt[from]
	target := e.board.pieceAt[to]

	if target != nil {
		e.removePiece(target, to)
	}

	pc.Square = to
	e.board.pieceAt[from] = nil
	e.board.pieceAt[to] = pc

	e.board.pieces[pc.Color][pc.Type] = e.board.pieces[pc.Color][pc.Type].Remove(from).Add(to)
	e.board.occupancy[pc.Color] = e.board.occupancy[pc.Color].Remove(from).Add(to)
	e.board.allOcc = e.board.allOcc.Remove(from).Add(to)
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

func (e *Engine) removePiece(pc *Piece, sq Square) {
	e.board.pieces[pc.Color][pc.Type] = e.board.pieces[pc.Color][pc.Type].Remove(sq)
	e.board.occupancy[pc.Color] = e.board.occupancy[pc.Color].Remove(sq)
	e.board.allOcc = e.board.allOcc.Remove(sq)
	e.board.pieceAt[sq] = nil
	delete(e.blockFacing, pc.ID)
	delete(e.pendingDoOver, pc.ID)
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

func (e *Engine) flipTurn() { e.board.turn = e.board.turn.Opposite() }

func (e *Engine) ensureConfigured() error {
	for _, color := range []Color{White, Black} {
		if !e.configured[color] {
			return fmt.Errorf("side %s not configured", color)
		}
	}
	return nil
}

func (e *Engine) lockConfiguration() error {
	if e.locked {
		return errors.New("configuration already locked")
	}
	e.locked = true
	e.board.lastNote = "Configuration locked - game started"
	return nil
}

func (e *Engine) requireBlockPathDirection(pc *Piece, dir Direction) error {
	if pc == nil || !pc.Abilities.Contains(AbilityBlockPath) {
		return nil
	}
	if dir == DirNone {
		if _, ok := e.blockFacing[pc.ID]; !ok {
			return ErrBlockPathDirectionRequired
		}
	}
	return nil
}

func (e *Engine) resolveBlockPathFacing(pc *Piece, dir Direction) string {
	if pc == nil || !pc.Abilities.Contains(AbilityBlockPath) {
		return ""
	}
	if dir == DirNone {
		if pc.BlockDir == DirNone {
			pc.BlockDir = DirN
			e.blockFacing[pc.ID] = pc.BlockDir
			return "BlockPath default facing N"
		}
		return ""
	}
	pc.BlockDir = dir
	e.blockFacing[pc.ID] = dir
	return fmt.Sprintf("BlockPath facing %s", dir)
}

func (e *Engine) captureBlockedByBlockPath(attacker *Piece, from Square, defender *Piece, to Square) (bool, string) {
	if defender == nil || !defender.Abilities.Contains(AbilityBlockPath) || defender.BlockDir == DirNone {
		return false, ""
	}
	if attacker != nil && elementOf(e, attacker) == ElementWater {
		return false, ""
	}
	dir := shared.DirectionOf(from, to)
	if dir == defender.BlockDir {
		return true, fmt.Sprintf("Capture blocked by BlockPath (%s)", defender.BlockDir)
	}
	return false, ""
}

func (e *Engine) resolvePromotion(pc *Piece) {
	if pc.Type != Pawn {
		return
	}
	if (pc.Color == White && pc.Square.Rank() == 7) || (pc.Color == Black && pc.Square.Rank() == 0) {
		e.board.pieces[pc.Color][Pawn] = e.board.pieces[pc.Color][Pawn].Remove(pc.Square)
		pc.Type = Queen
		e.board.pieces[pc.Color][Queen] = e.board.pieces[pc.Color][Queen].Add(pc.Square)
		appendAbilityNote(&e.board.lastNote, "Pawn promoted to Queen")
	}
}

// ---------------------------
// Data-driven & Static Helpers
// ---------------------------

func (e *Engine) hasChainKill(p *Piece) bool {
	return p != nil && p.Abilities.Contains(AbilityChainKill)
}
func (e *Engine) hasQuantumKill(p *Piece) bool {
	return p != nil && p.Abilities.Contains(AbilityQuantumKill)
}
func (e *Engine) hasDoubleKill(p *Piece) bool {
	return p != nil && p.Abilities.Contains(AbilityDoubleKill)
}
func (e *Engine) isSlider(pt PieceType) bool { return pt == Queen || pt == Rook || pt == Bishop }

var RankOrder = map[PieceType]int{King: 5, Queen: 4, Rook: 3, Bishop: 2, Knight: 2, Pawn: 1}

func rankOf(pt PieceType) int { return RankOrder[pt] }

func elementOf(e *Engine, p *Piece) Element {
	if p == nil {
		return ElementLight
	}
	if p.Element != 0 && p.Element != ElementNone {
		return p.Element
	}
	if e != nil && e.elements != nil {
		return e.elements[p.Color]
	}
	return ElementLight
}

type moveDelta struct {
	dr int
	df int
}

var (
	rookDirections = [...]moveDelta{
		{dr: 1, df: 0},
		{dr: -1, df: 0},
		{dr: 0, df: 1},
		{dr: 0, df: -1},
	}
	bishopDirections = [...]moveDelta{
		{dr: 1, df: 1},
		{dr: 1, df: -1},
		{dr: -1, df: 1},
		{dr: -1, df: -1},
	}
	knightOffsets = [...]moveDelta{
		{dr: 2, df: 1},
		{dr: 1, df: 2},
		{dr: -1, df: 2},
		{dr: -2, df: 1},
		{dr: -2, df: -1},
		{dr: -1, df: -2},
		{dr: 1, df: -2},
		{dr: 2, df: -1},
	}
	kingOffsets = [...]moveDelta{
		{dr: 1, df: 0}, {dr: 1, df: 1}, {dr: 0, df: 1}, {dr: -1, df: 1},
		{dr: -1, df: 0}, {dr: -1, df: -1}, {dr: 0, df: -1}, {dr: 1, df: -1},
	}
)

func (e *Engine) generateMoves(pc *Piece) Bitboard {
	if pc == nil {
		return 0
	}

	switch pc.Type {
	case Pawn:
		return e.generatePawnMoves(pc)
	case Knight:
		return e.generateKnightMoves(pc)
	case Bishop:
		return e.generateSlidingMoves(pc, bishopDirections[:])
	case Rook:
		return e.generateSlidingMoves(pc, rookDirections[:])
	case Queen:
		moves := e.generateSlidingMoves(pc, rookDirections[:])
		moves |= e.generateSlidingMoves(pc, bishopDirections[:])
		return moves
	case King:
		return e.generateKingMoves(pc)
	default:
		return 0
	}
}

func (e *Engine) generatePawnMoves(pc *Piece) Bitboard {
	var moves Bitboard

	rank := pc.Square.Rank()
	file := pc.Square.File()
	dir := 1
	startRank := 1

	if pc.Color == Black {
		dir = -1
		startRank = 6
	}

	forwardRank := rank + dir
	if target, ok := shared.SquareFromCoords(forwardRank, file); ok && e.board.pieceAt[target] == nil {
		moves = moves.Add(target)
		if rank == startRank {
			doubleRank := rank + 2*dir
			if doubleSq, ok := shared.SquareFromCoords(doubleRank, file); ok && e.board.pieceAt[doubleSq] == nil {
				moves = moves.Add(doubleSq)
			}
		}
	}

	for _, df := range []int{-1, 1} {
		captureRank := rank + dir
		captureFile := file + df
		if target, ok := shared.SquareFromCoords(captureRank, captureFile); ok {
			if victim := e.board.pieceAt[target]; victim != nil && victim.Color != pc.Color {
				moves = moves.Add(target)
			}
		}
	}

	return moves
}

func (e *Engine) generateKnightMoves(pc *Piece) Bitboard {
	var moves Bitboard
	rank := pc.Square.Rank()
	file := pc.Square.File()

	for _, delta := range knightOffsets {
		if target, ok := shared.SquareFromCoords(rank+delta.dr, file+delta.df); ok {
			if occupant := e.board.pieceAt[target]; occupant == nil || occupant.Color != pc.Color {
				moves = moves.Add(target)
			}
		}
	}
	return moves
}

func (e *Engine) generateKingMoves(pc *Piece) Bitboard {
	var moves Bitboard
	rank := pc.Square.Rank()
	file := pc.Square.File()

	for _, delta := range kingOffsets {
		if target, ok := shared.SquareFromCoords(rank+delta.dr, file+delta.df); ok {
			if occupant := e.board.pieceAt[target]; occupant == nil || occupant.Color != pc.Color {
				moves = moves.Add(target)
			}
		}
	}
	return moves
}

func (e *Engine) generateSlidingMoves(pc *Piece, directions []moveDelta) Bitboard {
	var moves Bitboard
	startRank := pc.Square.Rank()
	startFile := pc.Square.File()

	for _, delta := range directions {
		rank := startRank + delta.dr
		file := startFile + delta.df
		for {
			if target, ok := shared.SquareFromCoords(rank, file); ok {
				occupant := e.board.pieceAt[target]
				if occupant == nil {
					moves = moves.Add(target)
				} else {
					if occupant.Color != pc.Color {
						moves = moves.Add(target)
					}
					break
				}
				rank += delta.dr
				file += delta.df
				continue
			}
			break
		}
	}
	return moves
}
