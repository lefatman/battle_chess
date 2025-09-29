// path: chessTest/internal/game/engine.go
// Package game implements the core battle chess engine state and API.
package game

import (
	"errors"
	"fmt"
)

// Engine encapsulates the optimized chess engine with ability metadata.
type Engine struct {
	board           Board
	abilities       [2]AbilityList
	elements        [2]Element
	blockFacing     map[int]Direction
	history         []*historyDelta
	activeDelta     *historyDelta
	nextPieceID     int
	locked          bool
	configured      [2]bool
	pendingDoOver   map[int]bool // Tracks per-piece DoOver consumption
	currentMove     *MoveState
	temporalSlow    [2]int
	abilityHandlers map[Ability][]AbilityHandler
	abilityCtx      abilityContextCache
}

// Board represents the state of the chessboard.
type Board struct {
	pieces           [2][6]Bitboard
	occupancy        [2]Bitboard
	allOcc           Bitboard
	pieceAt          [64]*Piece
	turn             Color
	lastNote         string
	InCheck          bool
	GameOver         bool
	HasWinner        bool
	Winner           Color
	Status           string
	Castling         CastlingRights
	EnPassant        EnPassantTarget
	PromotionChoices PromotionChoices
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
	From         Square
	To           Square
	Dir          Direction // Chosen BlockPath direction after move
	Promotion    PieceType
	HasPromotion bool
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
	Pieces           []PieceState        `json:"pieces"`
	Turn             Color               `json:"turn"`
	TurnName         string              `json:"turnName"`
	LastNote         string              `json:"lastNote"`
	Abilities        map[string][]string `json:"abilities"`
	Elements         map[string]string   `json:"elements"`
	BlockFacing      map[int]Direction   `json:"blockFacing"`
	Locked           bool                `json:"locked"`
	InCheck          bool                `json:"inCheck"`
	GameOver         bool                `json:"gameOver"`
	Status           string              `json:"status"`
	HasWinner        bool                `json:"hasWinner"`
	Winner           Color               `json:"winner"`
	WinnerName       string              `json:"winnerName"`
	Castling         CastlingRights      `json:"castling"`
	EnPassant        EnPassantTarget     `json:"enPassant"`
	PromotionChoices PromotionChoices    `json:"promotionChoices"`
	AbilityRuntime   AbilityRuntimeState `json:"abilityRuntime,omitempty"`
}

// AbilityRuntimeState summarizes the engine's ability handler runtime for debugging.
type AbilityRuntimeState struct {
	HandlerCounts map[string]int  `json:"handlerCounts,omitempty"`
	CacheUsage    map[string]bool `json:"cacheUsage,omitempty"`
}

// NewEngine creates and initializes a new game engine.
func NewEngine() *Engine {
	eng := &Engine{
		abilities:       [2]AbilityList{},
		elements:        [2]Element{ElementLight, ElementShadow},
		blockFacing:     make(map[int]Direction),
		configured:      [2]bool{},
		pendingDoOver:   make(map[int]bool),
		currentMove:     nil,
		temporalSlow:    [2]int{},
		abilityHandlers: make(map[Ability][]AbilityHandler),
	}
	if err := eng.Reset(); err != nil {
		panic(err) // Should not happen on initial setup
	}
	return eng
}

// Reset clears the engine state and sets up a standard new game.
func (e *Engine) Reset() error {
	e.ensureAbilityRuntime()
	e.board = Board{}
	e.blockFacing = make(map[int]Direction)
	e.history = e.history[:0]
	e.activeDelta = nil
	e.nextPieceID = 1
	e.locked = false
	e.currentMove = nil
	for i := range e.temporalSlow {
		e.temporalSlow[i] = 0
	}
	for i := range e.configured {
		e.configured[i] = false
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
	e.board.Castling = CastlingAll
	e.board.EnPassant = NoEnPassantTarget()
	e.board.PromotionChoices = PromotionAll
	e.updateGameStatus()
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
	idx := color.Index()
	e.abilities[idx] = abilities.Clone()
	e.elements[idx] = element
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
	e.configured[idx] = true
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
	winnerName := ""
	if e.board.HasWinner {
		winnerName = e.board.Winner.String()
	}

	state := BoardState{
		Pieces:      make([]PieceState, 0, 32),
		Turn:        e.board.turn,
		TurnName:    e.board.turn.String(),
		LastNote:    e.board.lastNote,
		Abilities:   make(map[string][]string),
		Elements:    make(map[string]string),
		BlockFacing: make(map[int]Direction),
		Locked:      e.locked,
		InCheck:     e.board.InCheck,
		GameOver:    e.board.GameOver,
		Status:      e.board.Status,
		HasWinner:   e.board.HasWinner,
		Winner:      e.board.Winner,
		WinnerName:  winnerName,
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

	for _, color := range []Color{White, Black} {
		state.Abilities[color.String()] = e.abilities[color.Index()].Strings()
	}
	for _, color := range []Color{White, Black} {
		state.Elements[color.String()] = e.elements[color.Index()].String()
	}
	for id, dir := range e.blockFacing {
		state.BlockFacing[id] = dir
	}

	state.Castling = e.board.Castling
	state.EnPassant = e.board.EnPassant
	state.PromotionChoices = e.board.PromotionChoices
	state.AbilityRuntime = e.abilityRuntimeState()

	return state
}

func (e *Engine) ensureAbilityRuntime() {
	if e.abilityHandlers == nil {
		e.abilityHandlers = make(map[Ability][]AbilityHandler)
	}
	e.abilityCtx.clear()
}

func (e *Engine) abilityRuntimeState() AbilityRuntimeState {
	state := AbilityRuntimeState{}
	if len(e.abilityHandlers) > 0 {
		counts := make(map[string]int, len(e.abilityHandlers))
		for ability, handlers := range e.abilityHandlers {
			if len(handlers) == 0 {
				continue
			}
			counts[ability.String()] = len(handlers)
		}
		if len(counts) > 0 {
			state.HandlerCounts = counts
		}
	}
	if usage := e.abilityCtx.usage(); len(usage) > 0 {
		state.CacheUsage = usage
	}
	return state
}

func (e *Engine) flipTurn() { e.board.turn = e.board.turn.Opposite() }

func (e *Engine) ensureConfigured() error {
	for _, color := range []Color{White, Black} {
		if !e.configured[color.Index()] {
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
	e.recordBlockFacingForUndo(pc.ID)
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

func (e *Engine) resolvePromotion(pc *Piece) {
	if pc.Type != Pawn {
		return
	}
	if (pc.Color == White && pc.Square.Rank() == 7) || (pc.Color == Black && pc.Square.Rank() == 0) {
		e.board.pieces[pc.Color][Pawn] = e.board.pieces[pc.Color][Pawn].Remove(pc.Square)
		promoteTo := e.selectPromotionPiece(pc.Color)
		pc.Type = promoteTo
		e.board.pieces[pc.Color][promoteTo] = e.board.pieces[pc.Color][promoteTo].Add(pc.Square)
		appendAbilityNote(&e.board.lastNote, fmt.Sprintf("Pawn promoted to %s", promoteTo.String()))
	}
}

func (e *Engine) selectPromotionPiece(color Color) PieceType {
	choices := e.board.PromotionChoices
	if choices == PromotionNone {
		choices = PromotionAll
	}
	if e.currentMove != nil && e.currentMove.PromotionSet && choices.Contains(e.currentMove.Promotion) {
		return e.currentMove.Promotion
	}
	return choices.Default()
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
	if e != nil {
		return e.elements[p.Color.Index()]
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

	var moves Bitboard
	switch pc.Type {
	case Pawn:
		moves = e.generatePawnMoves(pc)
	case Knight:
		moves = e.generateKnightMoves(pc)
	case Bishop:
		moves = e.generateSlidingMoves(pc, bishopDirections[:])
	case Rook:
		moves = e.generateSlidingMoves(pc, rookDirections[:])
	case Queen:
		moves = e.generateSlidingMoves(pc, rookDirections[:])
		moves |= e.generateSlidingMoves(pc, bishopDirections[:])
	case King:
		moves = e.generateKingMoves(pc)
	default:
		moves = 0
	}

	if pc.Abilities.Contains(AbilityScatterShot) {
		moves = e.addScatterShotCaptures(pc, moves)
	}
	if e.resurrectionWindowActive(pc) {
		moves = e.addResurrectionCaptureWindow(pc, moves)
	}
	return moves
}

func (e *Engine) generatePawnMoves(pc *Piece) Bitboard {
	var moves Bitboard

	rank := pc.Square.Rank()
	file := pc.Square.File()
	from := pc.Square
	dir := 1
	startRank := 1

	if pc.Color == Black {
		dir = -1
		startRank = 6
	}

	forwardRank := rank + dir
	if target, ok := SquareFromCoords(forwardRank, file); ok && e.board.pieceAt[target] == nil {
		moves = moves.Add(target)
		if rank == startRank {
			doubleRank := rank + 2*dir
			if doubleSq, ok := SquareFromCoords(doubleRank, file); ok && e.board.pieceAt[doubleSq] == nil {
				moves = moves.Add(doubleSq)
			}
		}
	}

	for _, df := range []int{-1, 1} {
		captureRank := rank + dir
		captureFile := file + df
		if target, ok := SquareFromCoords(captureRank, captureFile); ok {
			if victim := e.board.pieceAt[target]; victim != nil && victim.Color != pc.Color && e.canDirectCapture(pc, victim, from, target) {
				moves = moves.Add(target)
			} else if epSq, ok := e.board.EnPassant.Square(); ok && epSq == target {
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
	from := pc.Square

	for _, delta := range knightOffsets {
		if target, ok := SquareFromCoords(rank+delta.dr, file+delta.df); ok {
			occupant := e.board.pieceAt[target]
			if occupant == nil || (occupant.Color != pc.Color && e.canDirectCapture(pc, occupant, from, target)) {
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
	from := pc.Square

	for _, delta := range kingOffsets {
		if target, ok := SquareFromCoords(rank+delta.dr, file+delta.df); ok {
			occupant := e.board.pieceAt[target]
			if occupant == nil || (occupant.Color != pc.Color && e.canDirectCapture(pc, occupant, from, target)) {
				moves = moves.Add(target)
			}
		}
	}
	if dest, ok := e.castleDestination(pc, CastleKingside); ok {
		moves = moves.Add(dest)
	}
	if dest, ok := e.castleDestination(pc, CastleQueenside); ok {
		moves = moves.Add(dest)
	}
	return moves
}

func (e *Engine) castleDestination(pc *Piece, side CastlingSide) (Square, bool) {
	if pc == nil || pc.Type != King {
		return 0, false
	}
	if !e.board.Castling.HasSide(pc.Color, side) {
		return 0, false
	}
	rank := pc.Square.Rank()
	file := pc.Square.File()
	enemy := pc.Color.Opposite()

	var rookFile int
	var travelFiles []int
	var emptyFiles []int
	var destFile int
	switch side {
	case CastleKingside:
		rookFile = 7
		travelFiles = []int{file + 1, file + 2}
		emptyFiles = []int{file + 1, file + 2}
		destFile = file + 2
	case CastleQueenside:
		rookFile = 0
		travelFiles = []int{file - 1, file - 2}
		emptyFiles = []int{file - 1, file - 2, file - 3}
		destFile = file - 2
	default:
		return 0, false
	}

	rookSq, ok := SquareFromCoords(rank, rookFile)
	if !ok {
		return 0, false
	}
	rook := e.board.pieceAt[rookSq]
	if rook == nil || rook.Color != pc.Color || rook.Type != Rook {
		return 0, false
	}

	for _, f := range emptyFiles {
		sq, ok := SquareFromCoords(rank, f)
		if !ok {
			return 0, false
		}
		if e.board.pieceAt[sq] != nil {
			return 0, false
		}
	}

	if e.isSquareAttackedBy(enemy, pc.Square) {
		return 0, false
	}
	for _, f := range travelFiles {
		sq, ok := SquareFromCoords(rank, f)
		if !ok {
			return 0, false
		}
		if e.isSquareAttackedBy(enemy, sq) {
			return 0, false
		}
	}

	dest, ok := SquareFromCoords(rank, destFile)
	if !ok {
		return 0, false
	}
	return dest, true
}

func (e *Engine) generateSlidingMoves(pc *Piece, directions []moveDelta) Bitboard {
	var moves Bitboard
	startRank := pc.Square.Rank()
	startFile := pc.Square.File()
	from := pc.Square

	for _, delta := range directions {
		rank := startRank + delta.dr
		file := startFile + delta.df
		for {
			if target, ok := SquareFromCoords(rank, file); ok {
				occupant := e.board.pieceAt[target]
				if occupant == nil {
					moves = moves.Add(target)
				} else {
					if occupant.Color != pc.Color && e.canDirectCapture(pc, occupant, from, target) {
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

func (e *Engine) addScatterShotCaptures(pc *Piece, moves Bitboard) Bitboard {
	from := pc.Square
	rank := from.Rank()
	file := from.File()
	for _, df := range []int{-1, 1} {
		if target, ok := SquareFromCoords(rank, file+df); ok {
			occupant := e.board.pieceAt[target]
			if occupant != nil && occupant.Color != pc.Color && e.canDirectCapture(pc, occupant, from, target) {
				moves = moves.Add(target)
			}
		}
	}
	return moves
}

func (e *Engine) resurrectionWindowActive(pc *Piece) bool {
	if pc == nil {
		return false
	}
	if !pc.Abilities.Contains(AbilityResurrection) {
		return false
	}
	if handlers := e.handlersForAbility(AbilityResurrection); len(handlers) > 0 {
		ctx := ResurrectionContext{Engine: e, Move: e.currentMove, Piece: pc}
		handled := false
		for _, handler := range handlers {
			provider, ok := handler.(ResurrectionWindowHandler)
			if !ok {
				continue
			}
			handled = true
			if provider.ResurrectionWindowActive(ctx) {
				return true
			}
		}
		if handled {
			return false
		}
	}
	if e.currentMove == nil || e.currentMove.Piece != pc {
		return false
	}
	if e.currentMove.abilityCounter(AbilityResurrection, abilityCounterResurrectionWindow) > 0 {
		return true
	}
	return e.currentMove.abilityFlag(AbilityResurrection, abilityFlagWindow)
}

func (e *Engine) addResurrectionCaptureWindow(pc *Piece, moves Bitboard) Bitboard {
	if handlers := e.handlersForAbility(AbilityResurrection); len(handlers) > 0 {
		ctx := ResurrectionContext{Engine: e, Move: e.currentMove, Piece: pc}
		handled := false
		for _, handler := range handlers {
			contributor, ok := handler.(ResurrectionCaptureWindowHandler)
			if !ok {
				continue
			}
			handled = true
			moves = contributor.AddResurrectionCaptureWindow(ctx, moves)
		}
		if handled {
			return moves
		}
	}

	from := pc.Square
	rank := from.Rank()
	file := from.File()
	for _, dr := range []int{-1, 1} {
		if target, ok := SquareFromCoords(rank+dr, file); ok {
			occupant := e.board.pieceAt[target]
			if occupant != nil && occupant.Color != pc.Color && e.canDirectCapture(pc, occupant, from, target) {
				moves = moves.Add(target)
			}
		}
	}
	return moves
}
