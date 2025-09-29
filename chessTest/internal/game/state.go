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
	temporalSlow  map[Color]int
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
	temporalSlow  map[Color]int
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

func cloneColorIntMap(src map[Color]int) map[Color]int {
	if len(src) == 0 {
		return make(map[Color]int)
	}
	clone := make(map[Color]int, len(src))
	for k, v := range src {
		clone[k] = v
	}
	return clone
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
		temporalSlow:  make(map[Color]int),
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
	if e.temporalSlow == nil {
		e.temporalSlow = make(map[Color]int, 2)
	} else {
		for k := range e.temporalSlow {
			delete(e.temporalSlow, k)
		}
	}
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

	for color, abilities := range e.abilities {
		state.Abilities[color.String()] = abilities.Strings()
	}
	for color, element := range e.elements {
		state.Elements[color.String()] = element.String()
	}
	for id, dir := range e.blockFacing {
		state.BlockFacing[id] = dir
	}

	state.Castling = e.board.Castling
	state.EnPassant = e.board.EnPassant
	state.PromotionChoices = e.board.PromotionChoices

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
	if victim == nil {
		return nil
	}
	if err := e.maybeTriggerDoOver(victim); err != nil {
		return err
	}

	victimRank := rankOf(victim.Type)
	victimColor := victim.Color

	extraRemoved := false
	if attacker != nil && attacker.Abilities.Contains(AbilityDoubleKill) {
		if target := e.trySmartExtraCapture(attacker, captureSquare, victimColor, victimRank); target != nil {
			targetSquare := target.Square
			if removed, err := e.attemptAbilityRemoval(attacker, target); err != nil {
				return err
			} else if removed {
				appendAbilityNote(&e.board.lastNote, fmt.Sprintf("DoubleKill: removed %s %s at %s", target.Color, target.Type, targetSquare))
				extraRemoved = true
			}
		}
	}

	if !extraRemoved && attacker != nil && elementOf(e, attacker) == ElementFire && attacker.Abilities.Contains(AbilityScorch) {
		if target := e.trySmartExtraCapture(attacker, captureSquare, victimColor, victimRank); target != nil {
			targetSquare := target.Square
			if removed, err := e.attemptAbilityRemoval(attacker, target); err != nil {
				return err
			} else if removed {
				appendAbilityNote(&e.board.lastNote, fmt.Sprintf("Fire Scorch: removed %s %s at %s", target.Color, target.Type, targetSquare))
			}
		}
	}

	if attacker != nil && e.currentMove != nil && e.currentMove.HasQuantumKill && !e.currentMove.QuantumKillUsed {
		e.currentMove.QuantumKillUsed = true
		if target := e.findQuantumKillTarget(attacker, victimColor, victimRank); target != nil {
			targetSquare := target.Square
			if removed, err := e.attemptAbilityRemoval(attacker, target); err != nil {
				return err
			} else if removed {
				appendAbilityNote(&e.board.lastNote, fmt.Sprintf("Quantum Kill: removed %s %s at %s", target.Color, target.Type, targetSquare))
				if echo := e.trySmartExtraCapture(attacker, targetSquare, victimColor, rankOf(target.Type)); echo != nil {
					echoSquare := echo.Square
					if removedEcho, err := e.attemptAbilityRemoval(attacker, echo); err != nil {
						return err
					} else if removedEcho {
						appendAbilityNote(&e.board.lastNote, fmt.Sprintf("Quantum Echo: removed %s %s at %s", echo.Color, echo.Type, echoSquare))
					}
				}
			}
		}
	}

	e.applyCapturePenalties(attacker)
	return nil
}

// trySmartExtraCapture scans neighbors and identifies the highest-rank, eligible, lower-rank piece.
func (e *Engine) trySmartExtraCapture(attacker *Piece, captureSquare Square, victimColor Color, victimRank int) *Piece {
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
		if !e.canAbilityRemove(attacker, p) {
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

	return bestP
}

func (e *Engine) maybeTriggerDoOver(victim *Piece) error {
	if victim == nil || !victim.Abilities.Contains(AbilityDoOver) || e.pendingDoOver[victim.ID] {
		return nil
	}

	plies := 4
	if plies > len(e.history) {
		plies = len(e.history)
	}
	if plies > 0 {
		e.popHistory(plies)
		victim.Abilities = victim.Abilities.Without(AbilityDoOver)
		e.pendingDoOver[victim.ID] = true
		e.board.lastNote = fmt.Sprintf("DoOver: %s %s rewound %d plies (%.1f turns)", victim.Color, victim.Type, plies, float64(plies)/2.0)
		return ErrDoOverActivated
	}

	victim.Abilities = victim.Abilities.Without(AbilityDoOver)
	e.pendingDoOver[victim.ID] = true
	return nil
}

func (e *Engine) canAbilityRemove(attacker, target *Piece) bool {
	if target == nil {
		return false
	}
	if target.Type == King {
		return false
	}
	if target.Abilities.Contains(AbilityIndomitable) {
		return false
	}
	if target.Abilities.Contains(AbilityStalwart) && attacker != nil && rankOf(attacker.Type) < rankOf(target.Type) {
		return false
	}
	if target.Abilities.Contains(AbilityBelligerent) && attacker != nil && rankOf(attacker.Type) > rankOf(target.Type) {
		return false
	}
	return true
}

func (e *Engine) attemptAbilityRemoval(attacker, target *Piece) (bool, error) {
	if target == nil || !e.canAbilityRemove(attacker, target) {
		return false, nil
	}

	e.removePiece(target, target.Square)
	if err := e.maybeTriggerDoOver(target); err != nil {
		return true, err
	}
	return true, nil
}

func (e *Engine) findQuantumKillTarget(attacker *Piece, victimColor Color, maxRank int) *Piece {
	var best *Piece
	bestRank := -1
	bestIndex := 65

	for idx, p := range e.board.pieceAt {
		if p == nil || p.Color != victimColor {
			continue
		}
		rank := rankOf(p.Type)
		if rank > maxRank {
			continue
		}
		if !e.canAbilityRemove(attacker, p) {
			continue
		}
		if rank > bestRank || (rank == bestRank && idx < bestIndex) {
			best = p
			bestRank = rank
			bestIndex = idx
		}
	}

	return best
}

func (e *Engine) applyCapturePenalties(attacker *Piece) {
	if e.currentMove == nil || attacker == nil {
		return
	}

	element := elementOf(e, attacker)
	if attacker.Abilities.Contains(AbilityPoisonousMeat) && element != ElementShadow {
		if e.currentMove.RemainingSteps > 0 {
			e.currentMove.RemainingSteps--
			appendAbilityNote(&e.board.lastNote, "Poisonous Meat drains 1 step")
		}
	}

	if attacker.Abilities.Contains(AbilityOverload) && element == ElementLightning && attacker.Abilities.Contains(AbilityStalwart) {
		if e.currentMove.RemainingSteps > 0 {
			e.currentMove.RemainingSteps--
			appendAbilityNote(&e.board.lastNote, "Overload + Stalwart costs 1 step")
		}
	}
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
		temporalSlow:  cloneColorIntMap(e.temporalSlow),
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
	e.temporalSlow = cloneColorIntMap(s.temporalSlow)
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
			if victim := e.board.pieceAt[target]; victim != nil && victim.Color != pc.Color && e.canDirectCapture(pc, victim, from, target) {
				moves = moves.Add(target)
			} else if epSq, ok := e.board.EnPassant.Square(); ok && epSq == target {
				moves = moves.Add(target)
			}
		}
	}

	hasUmbral := pc.Abilities.Contains(AbilityUmbralStep)
	if !hasUmbral && e.abilities != nil {
		if al, ok := e.abilities[pc.Color]; ok {
			hasUmbral = al.Contains(AbilityUmbralStep)
		}
	}

	if hasUmbral {
		backwardRank := rank - dir
		if target, ok := shared.SquareFromCoords(backwardRank, file); ok && e.board.pieceAt[target] == nil {
			moves = moves.Add(target)
		}

		for _, df := range []int{-1, 1} {
			captureRank := rank - dir
			captureFile := file + df
			if target, ok := shared.SquareFromCoords(captureRank, captureFile); ok {
				if victim := e.board.pieceAt[target]; victim != nil && victim.Color != pc.Color && e.canDirectCapture(pc, victim, from, target) {
					moves = moves.Add(target)
				} else if epSq, ok := e.board.EnPassant.Square(); ok && epSq == target {
					moves = moves.Add(target)
				}
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
		if target, ok := shared.SquareFromCoords(rank+delta.dr, file+delta.df); ok {
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
		if target, ok := shared.SquareFromCoords(rank+delta.dr, file+delta.df); ok {
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

	rookSq, ok := shared.SquareFromCoords(rank, rookFile)
	if !ok {
		return 0, false
	}
	rook := e.board.pieceAt[rookSq]
	if rook == nil || rook.Color != pc.Color || rook.Type != Rook {
		return 0, false
	}

	for _, f := range emptyFiles {
		sq, ok := shared.SquareFromCoords(rank, f)
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
		sq, ok := shared.SquareFromCoords(rank, f)
		if !ok {
			return 0, false
		}
		if e.isSquareAttackedBy(enemy, sq) {
			return 0, false
		}
	}

	dest, ok := shared.SquareFromCoords(rank, destFile)
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
			if target, ok := shared.SquareFromCoords(rank, file); ok {
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
		if target, ok := shared.SquareFromCoords(rank, file+df); ok {
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
	if e.currentMove == nil || e.currentMove.Piece != pc {
		return false
	}
	return e.currentMove.ResurrectionWindow
}

func (e *Engine) addResurrectionCaptureWindow(pc *Piece, moves Bitboard) Bitboard {
	from := pc.Square
	rank := from.Rank()
	file := from.File()
	for _, dr := range []int{-1, 1} {
		if target, ok := shared.SquareFromCoords(rank+dr, file); ok {
			occupant := e.board.pieceAt[target]
			if occupant != nil && occupant.Color != pc.Color && e.canDirectCapture(pc, occupant, from, target) {
				moves = moves.Add(target)
			}
		}
	}
	return moves
}
