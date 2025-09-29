package game

import (
	"errors"
	"fmt"

	"battle_chess_poc/internal/shared"
)

// MoveState tracks the current move in progress with a step budget system.
// It holds all the temporary state for a piece's actions within a single turn.
type MoveState struct {
	Piece                 *Piece
	RemainingSteps        int
	UsedPhasing           bool
	Path                  []Square
	Captures              []*Piece
	HasChainKill          bool
	HasQuantumKill        bool
	HasDoubleKill         bool
	HasResurrection       bool
	ResurrectionWindow    bool
	TurnEnded             bool
	FreeTurnsUsed         int  // For Mist Shroud
	BlazeRushUsed         bool // Once per turn
	QuantumStepUsed       bool // Once per turn
	SideStepUsed          bool // Once per turn
	TemporalLockUsed      bool // Once per turn
	ChainKillCaptureCount int  // Track Chain Kill captures (max 2 additional)
	FloodWakePushUsed     bool // Track Flood Wake push usage
	MaxCaptures           int
	LastSegmentCaptured   bool
	QuantumKillUsed       bool
	Promotion             PieceType
	PromotionSet          bool
}

func (ms *MoveState) canCaptureMore() bool {
	if len(ms.Captures) == 0 {
		return true
	}
	if ms.MaxCaptures <= 0 {
		return true
	}
	return len(ms.Captures) < ms.MaxCaptures
}

func (ms *MoveState) registerCapture(captured *Piece) {
	if captured == nil {
		return
	}
	ms.Captures = append(ms.Captures, captured)
	if ms.HasResurrection {
		ms.ResurrectionWindow = true
	}
	if ms.HasChainKill && captured.Color != ms.Piece.Color {
		ms.ChainKillCaptureCount++
	}
}

func (e *Engine) calculateMaxCaptures(pc *Piece) int {
	maxCaptures := 1 // Base capture limit
	if e.hasChainKill(pc) {
		maxCaptures += 2 // Chain Kill allows 2 additional captures
	}
	return maxCaptures
}

func (e *Engine) checkPostCaptureTermination(pc *Piece, target *Piece) bool {
	if target == nil {
		return false
	}
	return e.shouldEndTurnAfterCapture(pc)
}

// ---------------------------
// Move Lifecycle Management
// ---------------------------

// startNewMove begins a new move, creating and initializing a MoveState.
// It calculates the initial step budget and executes the first segment of the move.
func (e *Engine) startNewMove(req MoveRequest) error {
	from, to := req.From, req.To
	pc := e.board.pieceAt[from]
	if pc == nil {
		return errors.New("no piece at source square")
	}
	if pc.Color != e.board.turn {
		return errors.New("not your turn")
	}

	target := e.board.pieceAt[to]
	if target != nil && target.Color == pc.Color {
		return errors.New("cannot capture a friendly piece")
	}

	segmentCtx := moveSegmentContext{capture: target, captureSquare: to}
	if target == nil && pc.Type == Pawn && from.File() != to.File() {
		if sq, ok := e.board.EnPassant.Square(); ok && sq == to {
			captureRank := to.Rank()
			if pc.Color == White {
				captureRank--
			} else {
				captureRank++
			}
			captureSq, ok := shared.SquareFromCoords(captureRank, to.File())
			if !ok {
				return errors.New("invalid en passant capture")
			}
			victim := e.board.pieceAt[captureSq]
			if victim == nil || victim.Color == pc.Color || victim.Type != Pawn {
				return errors.New("invalid en passant capture")
			}
			segmentCtx.capture = victim
			segmentCtx.captureSquare = captureSq
			segmentCtx.enPassant = true
		} else {
			return errors.New("illegal pawn capture")
		}
	}

	// Validate the legality of the first move segment.
	if !e.isLegalFirstSegment(pc, from, to) {
		return errors.New("illegal first move")
	}
	if err := e.requireBlockPathDirection(pc, req.Dir); err != nil {
		return err
	}
	if blocked, note := e.captureBlockedByBlockPath(pc, from, target, to); blocked {
		appendAbilityNote(&e.board.lastNote, note)
		return ErrCaptureBlocked
	}

	// Calculate the total step budget for this turn.
	totalSteps := e.calculateStepBudget(pc)
	usedPhasing := e.canPhaseThrough(pc, from, to)

	// Create the new MoveState for the current turn.
	firstSegmentCost := e.calculateMovementCost(pc, from, to)
	remainingSteps := totalSteps - firstSegmentCost
	if remainingSteps < 0 {
		remainingSteps = 0
	}

	maxCaptures := e.calculateMaxCaptures(pc)

	e.currentMove = &MoveState{
		Piece:           pc,
		RemainingSteps:  remainingSteps,
		UsedPhasing:     usedPhasing,
		Path:            []Square{from, to},
		Captures:        []*Piece{},
		HasChainKill:    e.hasChainKill(pc),
		HasQuantumKill:  e.hasQuantumKill(pc),
		HasDoubleKill:   e.hasDoubleKill(pc),
		HasResurrection: pc.Abilities.Contains(AbilityResurrection),
		MaxCaptures:     maxCaptures,
		Promotion:       req.Promotion,
		PromotionSet:    req.HasPromotion,
	}

	// The move is now valid, push the state before executing.
	e.pushHistory()
	e.executeMoveSegment(from, to, segmentCtx)
	e.handlePostSegment(pc, from, to, segmentCtx.capture)

	// Handle capture abilities if a piece was taken.
	if segmentCtx.capture != nil {
		e.currentMove.registerCapture(segmentCtx.capture)
		if err := e.ResolveCaptureAbility(pc, segmentCtx.capture, segmentCtx.captureSquare); err != nil {
			// If DoOver was triggered, the state is already rewound. Abort.
			e.currentMove = nil // Clear the invalid move state
			return err
		}
		if !e.currentMove.canCaptureMore() {
			e.endTurn()
			return nil
		}
	}

	// Check for abilities that end the turn immediately after a capture.
	if segmentCtx.capture != nil && e.shouldEndTurnAfterCapture(pc) {
		e.endTurn()
		return nil
	}

	// Resolve post-move state changes.
	e.resolvePromotion(pc)
	if note := e.resolveBlockPathFacing(pc, req.Dir); note != "" {
		appendAbilityNote(&e.board.lastNote, note)
	}
	e.checkPostMoveAbilities(pc)
	e.checkPostMoveAbilities(pc)

	// Check if the turn should end naturally.
	if e.currentMove.TurnEnded {
		e.endTurn()
	} else if e.currentMove.RemainingSteps <= 0 && !e.hasFreeContinuation(pc) {
		e.endTurn()
	} else {
		appendAbilityNote(&e.board.lastNote, fmt.Sprintf("%d steps remaining", e.currentMove.RemainingSteps))
	}

	return nil
}

// continueMove handles subsequent actions within a single turn, using the existing MoveState.
func (e *Engine) continueMove(req MoveRequest) error {
	if e.currentMove == nil || e.currentMove.TurnEnded {
		return errors.New("no active move to continue")
	}

	pc := e.currentMove.Piece
	from, to := req.From, req.To
	if from != pc.Square {
		return errors.New("must continue move from the piece's current square")
	}

	if req.HasPromotion {
		e.currentMove.Promotion = req.Promotion
		e.currentMove.PromotionSet = true
	}

	if handled, err := e.trySideStepNudge(pc, from, to); handled {
		return err
	}

	if handled, err := e.tryQuantumStep(pc, from, to); handled {
		return err
	}

	// Validate the legality of the continuation move.
	if !e.isLegalContinuation(pc, from, to) {
		return errors.New("illegal move continuation")
	}

	stepsNeeded := e.calculateMovementCost(pc, from, to)
	if stepsNeeded > e.currentMove.RemainingSteps {
		return fmt.Errorf("insufficient steps: %d needed, %d remaining", stepsNeeded, e.currentMove.RemainingSteps)
	}

	if e.currentMove.ResurrectionWindow {
		e.currentMove.ResurrectionWindow = false
	}

	target := e.board.pieceAt[to]
	if target != nil && target.Color == pc.Color {
		return errors.New("cannot capture a friendly piece")
	}
	if blocked, note := e.captureBlockedByBlockPath(pc, from, target, to); blocked {
		appendAbilityNote(&e.board.lastNote, note)
		return ErrCaptureBlocked
	}

	segmentCtx := moveSegmentContext{capture: target, captureSquare: to}
	if target == nil && pc.Type == Pawn && from.File() != to.File() {
		if sq, ok := e.board.EnPassant.Square(); ok && sq == to {
			captureRank := to.Rank()
			if pc.Color == White {
				captureRank--
			} else {
				captureRank++
			}
			captureSq, ok := shared.SquareFromCoords(captureRank, to.File())
			if !ok {
				return errors.New("invalid en passant capture")
			}
			victim := e.board.pieceAt[captureSq]
			if victim == nil || victim.Color == pc.Color || victim.Type != Pawn {
				return errors.New("invalid en passant capture")
			}
			segmentCtx.capture = victim
			segmentCtx.captureSquare = captureSq
			segmentCtx.enPassant = true
		} else {
			return errors.New("illegal pawn capture")
		}
	}

	// Continuation is valid, push state and execute.
	e.pushHistory()
	e.currentMove.RemainingSteps -= stepsNeeded
	e.executeMoveSegment(from, to, segmentCtx)
	e.currentMove.Path = append(e.currentMove.Path, to)
	e.handlePostSegment(pc, from, to, segmentCtx.capture)

	if segmentCtx.capture != nil {
		e.currentMove.registerCapture(segmentCtx.capture)
		if err := e.ResolveCaptureAbility(pc, segmentCtx.capture, segmentCtx.captureSquare); err != nil {
			e.currentMove = nil
			return err
		}
		if !e.currentMove.canCaptureMore() {
			e.endTurn()
			return nil
		}
	}

	// Check for turn-ending conditions after the action.
	if e.checkPostCaptureTermination(pc, segmentCtx.capture) {
		e.endTurn()
	} else if e.currentMove.RemainingSteps <= 0 && !e.hasFreeContinuation(pc) {
		e.endTurn()
	} else {
		appendAbilityNote(&e.board.lastNote, fmt.Sprintf("%d steps remaining", e.currentMove.RemainingSteps))
	}

	return nil
}

func (e *Engine) trySideStepNudge(pc *Piece, from, to Square) (bool, error) {
	if e.currentMove == nil || pc == nil {
		return false, nil
	}
	if !pc.Abilities.Contains(AbilitySideStep) || e.currentMove.SideStepUsed || e.currentMove.RemainingSteps <= 0 {
		return false, nil
	}
	if !isAdjacentSquare(from, to) {
		return false, nil
	}

	if target := e.board.pieceAt[to]; target != nil {
		return false, nil
	}

	e.pushHistory()
	e.currentMove.RemainingSteps--
	e.currentMove.SideStepUsed = true

	segmentCtx := moveSegmentContext{}
	e.executeMoveSegment(from, to, segmentCtx)
	e.currentMove.Path = append(e.currentMove.Path, to)
	e.handlePostSegment(pc, from, to, nil)

	if e.currentMove != nil {
		e.currentMove.ResurrectionWindow = false
	}

	appendAbilityNote(&e.board.lastNote, "Side Step nudge (cost 1 step)")

	if e.checkPostCaptureTermination(pc, nil) {
		e.endTurn()
	} else if e.currentMove.RemainingSteps <= 0 && !e.hasFreeContinuation(pc) {
		e.endTurn()
	} else {
		appendAbilityNote(&e.board.lastNote, fmt.Sprintf("%d steps remaining", e.currentMove.RemainingSteps))
	}

	return true, nil
}

func (e *Engine) tryQuantumStep(pc *Piece, from, to Square) (bool, error) {
	if e.currentMove == nil || pc == nil {
		return false, nil
	}
	if !pc.Abilities.Contains(AbilityQuantumStep) || e.currentMove.QuantumStepUsed || e.currentMove.RemainingSteps <= 0 {
		return false, nil
	}

	ally, ok := e.validateQuantumStep(pc, from, to)
	if !ok {
		return false, nil
	}

	e.pushHistory()
	e.currentMove.RemainingSteps--
	if e.currentMove.RemainingSteps < 0 {
		e.currentMove.RemainingSteps = 0
	}
	e.currentMove.QuantumStepUsed = true
	e.currentMove.ResurrectionWindow = false

	if ally == nil {
		e.executeMoveSegment(from, to, moveSegmentContext{})
		appendAbilityNote(&e.board.lastNote, "Quantum Step blink (cost 1 step)")
	} else {
		e.performQuantumSwap(pc, ally, from, to)
		appendAbilityNote(&e.board.lastNote, "Quantum Step swap (cost 1 step)")
	}

	e.currentMove.Path = append(e.currentMove.Path, to)
	e.handlePostSegment(pc, from, to, nil)

	if e.checkPostCaptureTermination(pc, nil) {
		e.endTurn()
		return true, nil
	}
	if e.currentMove == nil {
		return true, nil
	}
	if e.currentMove.RemainingSteps <= 0 && !e.hasFreeContinuation(pc) {
		e.endTurn()
	} else {
		appendAbilityNote(&e.board.lastNote, fmt.Sprintf("%d steps remaining", e.currentMove.RemainingSteps))
	}

	return true, nil
}

func (e *Engine) validateQuantumStep(pc *Piece, from, to Square) (*Piece, bool) {
	if pc == nil {
		return nil, false
	}
	if !isAdjacentSquare(from, to) {
		return nil, false
	}

	target := e.board.pieceAt[to]
	if target != nil && target.Color != pc.Color {
		return nil, false
	}

	if target == nil && e.isLegalContinuation(pc, from, to) {
		return nil, false
	}

	if target != nil && target.Color == pc.Color {
		return target, true
	}

	return nil, true
}

func (e *Engine) performQuantumSwap(pc, ally *Piece, from, to Square) {
	if pc == nil || ally == nil {
		return
	}
	if ally.Color != pc.Color {
		return
	}

	e.board.EnPassant = NoEnPassantTarget()

	e.board.pieceAt[from] = ally
	e.board.pieceAt[to] = pc

	pc.Square = to
	ally.Square = from

	e.board.pieces[pc.Color][pc.Type] = e.board.pieces[pc.Color][pc.Type].Remove(from).Add(to)
	e.board.pieces[ally.Color][ally.Type] = e.board.pieces[ally.Color][ally.Type].Remove(to).Add(from)

	occ := e.board.occupancy[pc.Color]
	occ = occ.Remove(from)
	occ = occ.Remove(to)
	occ = occ.Add(from)
	occ = occ.Add(to)
	e.board.occupancy[pc.Color] = occ

	all := e.board.allOcc
	all = all.Remove(from)
	all = all.Remove(to)
	all = all.Add(from)
	all = all.Add(to)
	e.board.allOcc = all

	e.updateCastlingRightsForMove(pc, from)
	e.updateCastlingRightsForMove(ally, to)
}

func isAdjacentSquare(from, to Square) bool {
	dr := absInt(to.Rank() - from.Rank())
	df := absInt(to.File() - from.File())
	if dr == 0 && df == 0 {
		return false
	}
	return dr <= 1 && df <= 1
}

// endTurn finalizes the move, performs cleanup, and passes control to the other player.
func (e *Engine) endTurn() {
	if e.currentMove == nil {
		// This can happen if a move was aborted (e.g., DoOver).
		return
	}

	pc := e.currentMove.Piece
	e.resolvePromotion(pc)
	e.applyTemporalLockSlow(pc)
	e.flipTurn()
	e.updateGameStatus()

	var note string
	switch {
	case e.board.GameOver && e.board.Status == "checkmate" && e.board.HasWinner:
		note = fmt.Sprintf("Checkmate - %s wins", e.board.Winner.String())
	case e.board.GameOver && e.board.Status == "stalemate":
		note = "Stalemate"
	case e.board.GameOver:
		note = e.board.Status
	case e.board.InCheck:
		note = fmt.Sprintf("%s to move (in check)", e.board.turn)
	default:
		note = fmt.Sprintf("%s's turn", e.board.turn)
	}
	appendAbilityNote(&e.board.lastNote, note)

	// Clear the current move state, officially ending the turn.
	e.currentMove = nil
}

func (e *Engine) applyTemporalLockSlow(pc *Piece) {
	if pc == nil || !pc.Abilities.Contains(AbilityTemporalLock) {
		return
	}
	if e.temporalSlow == nil {
		e.temporalSlow = make(map[Color]int, 2)
	}

	slow := 1
	if elementOf(e, pc) == ElementFire {
		slow = 2
	}

	opponent := pc.Color.Opposite()
	e.temporalSlow[opponent] = slow
	appendAbilityNote(&e.board.lastNote, fmt.Sprintf("Temporal Lock slows %s by %d", opponent, slow))
}

// ---------------------------
// Step & Cost Calculation
// ---------------------------

// calculateStepBudget calculates the total number of steps a piece gets for its turn.
func (e *Engine) calculateStepBudget(pc *Piece) int {
	baseSteps := 1 // Every piece gets at least one step.
	bonus := 0
	element := elementOf(e, pc)

	// Elemental & Ability Bonuses/Penalties
	if pc.Abilities.Contains(AbilityScorch) && element == ElementFire {
		bonus++ // Scorch grants +1 step
	}
	if pc.Abilities.Contains(AbilityTailwind) && element == ElementAir {
		bonus += 2 // Tailwind grants +2 steps
		if pc.Abilities.Contains(AbilityTemporalLock) {
			bonus-- // Temporal Lock slows Tailwind by 1
		}
	}
	if pc.Abilities.Contains(AbilityRadiantVision) && element == ElementLight {
		bonus++ // Radiant Vision grants +1 step
		if pc.Abilities.Contains(AbilityMistShroud) {
			bonus++ // Mist combo grants an additional step
		}
	}
	if pc.Abilities.Contains(AbilityUmbralStep) && element == ElementShadow {
		bonus += 2 // Umbral Step grants +2 steps
		if pc.Abilities.Contains(AbilityRadiantVision) {
			bonus-- // Radiant Vision dampens Umbral Step by 1
		}
	}
	if pc.Abilities.Contains(AbilitySchrodingersLaugh) {
		bonus += 2 // Schrodinger's Laugh grants +2 steps
		if pc.Abilities.Contains(AbilitySideStep) {
			bonus++ // Interaction bonus with Side Step
		}
	}
	slowPenalty := 0
	if e.temporalSlow != nil {
		if slow, ok := e.temporalSlow[pc.Color]; ok {
			slowPenalty = slow
			delete(e.temporalSlow, pc.Color)
		}
	}

	totalSteps := baseSteps + bonus - slowPenalty
	if totalSteps < 1 {
		return 1 // A piece always gets at least 1 step.
	}
	return totalSteps
}

// calculateMovementCost calculates the step cost for a given move segment.
func (e *Engine) calculateMovementCost(pc *Piece, from, to Square) int {
	cost := 1 // Basic movement costs 1 step.

	target := e.board.pieceAt[to]

	// Sliders (Queen, Rook, Bishop) pay an extra step to change direction mid-turn.
	if e.isSlider(pc.Type) {
		pathLen := 0
		if e.currentMove != nil {
			pathLen = len(e.currentMove.Path)
		}
		if pathLen > 1 {
			prevFrom := e.currentMove.Path[pathLen-2]
			prevDir := shared.DirectionOf(prevFrom, from)
			currentDir := shared.DirectionOf(from, to)

			if prevDir != currentDir && prevDir != DirNone && currentDir != DirNone {
				if !(pc.Abilities.Contains(AbilityMistShroud) && e.currentMove != nil && e.currentMove.FreeTurnsUsed == 0) {
					cost++ // Direction change costs an extra step.
				}
			}
		}
	}

	if e.isFloodWakePushAvailable(pc, from, to, target) {
		cost = 0
	}

	if e.isBlazeRushDash(pc, from, to, target) {
		cost = 0
	}

	return cost
}

func (e *Engine) isFloodWakePushAvailable(pc *Piece, from, to Square, target *Piece) bool {
	if pc == nil || target != nil {
		return false
	}
	if !pc.Abilities.Contains(AbilityFloodWake) {
		return false
	}
	if elementOf(e, pc) != ElementWater {
		return false
	}
	dr := absInt(to.Rank() - from.Rank())
	df := absInt(to.File() - from.File())
	if dr+df != 1 {
		return false
	}
	if e.currentMove != nil && e.currentMove.FloodWakePushUsed {
		return false
	}
	return true
}

func (e *Engine) isBlazeRushDash(pc *Piece, from, to Square, target *Piece) bool {
	if pc == nil || target != nil {
		return false
	}
	if !pc.Abilities.Contains(AbilityBlazeRush) {
		return false
	}
	if !e.isSlider(pc.Type) {
		return false
	}
	if e.currentMove == nil {
		return false
	}
	if e.currentMove.BlazeRushUsed || e.currentMove.LastSegmentCaptured {
		return false
	}
	pathLen := len(e.currentMove.Path)
	var prevStart, prevEnd Square
	switch {
	case pathLen >= 3 && e.currentMove.Path[pathLen-1] == to && e.currentMove.Path[pathLen-2] == from:
		prevStart = e.currentMove.Path[pathLen-3]
		prevEnd = from
	case pathLen >= 2 && e.currentMove.Path[pathLen-1] == from:
		prevStart = e.currentMove.Path[pathLen-2]
		prevEnd = from
	default:
		return false
	}
	if !e.blazeRushSegmentOk(prevStart, prevEnd, from, to) {
		return false
	}
	return true
}

func (e *Engine) handlePostSegment(pc *Piece, from, to Square, target *Piece) {
	if e.currentMove == nil {
		return
	}

	e.currentMove.LastSegmentCaptured = target != nil

	if e.isFloodWakePushAvailable(pc, from, to, target) {
		e.currentMove.FloodWakePushUsed = true
		appendAbilityNote(&e.board.lastNote, "Flood Wake push (free)")
	}

	if e.isBlazeRushDash(pc, from, to, target) {
		e.currentMove.BlazeRushUsed = true
		appendAbilityNote(&e.board.lastNote, "Blaze Rush dash (free)")
	}

	e.logDirectionChange(pc)
}

func (e *Engine) logDirectionChange(pc *Piece) {
	if e.currentMove == nil || !e.isSlider(pc.Type) {
		return
	}
	if len(e.currentMove.Path) < 3 {
		return
	}

	last := len(e.currentMove.Path) - 1
	prevStart := e.currentMove.Path[last-2]
	prevEnd := e.currentMove.Path[last-1]
	currentEnd := e.currentMove.Path[last]
	prevDir := shared.DirectionOf(prevStart, prevEnd)
	currentDir := shared.DirectionOf(prevEnd, currentEnd)
	if prevDir == DirNone || currentDir == DirNone || prevDir == currentDir {
		return
	}

	if pc.Abilities.Contains(AbilityMistShroud) && e.currentMove.FreeTurnsUsed == 0 {
		e.currentMove.FreeTurnsUsed++
		appendAbilityNote(&e.board.lastNote, "Mist Shroud free pivot")
		return
	}

	appendAbilityNote(&e.board.lastNote, "Direction change cost +1 step")
}

func (e *Engine) hasFreeContinuation(pc *Piece) bool {
	if e.currentMove == nil || pc == nil {
		return false
	}
	if e.hasBlazeRushOption(pc) {
		return true
	}
	if e.hasFloodWakePushOption(pc) {
		return true
	}
	return false
}

func (e *Engine) hasBlazeRushOption(pc *Piece) bool {
	if pc == nil || !pc.Abilities.Contains(AbilityBlazeRush) {
		return false
	}
	if e.currentMove == nil || e.currentMove.BlazeRushUsed || e.currentMove.LastSegmentCaptured {
		return false
	}
	if !e.isSlider(pc.Type) {
		return false
	}
	if len(e.currentMove.Path) < 2 {
		return false
	}
	prevFrom := e.currentMove.Path[len(e.currentMove.Path)-2]
	prevTo := e.currentMove.Path[len(e.currentMove.Path)-1]
	dr, df, ok := directionStep(shared.DirectionOf(prevFrom, prevTo))
	if !ok {
		return false
	}
	nextRank := prevTo.Rank() + dr
	nextFile := prevTo.File() + df
	if sq, valid := shared.SquareFromCoords(nextRank, nextFile); valid {
		if e.board.pieceAt[sq] == nil {
			return true
		}
	}
	return false
}

func (e *Engine) blazeRushSegmentOk(prevStart, prevEnd, from, to Square) bool {
	prevDir := shared.DirectionOf(prevStart, prevEnd)
	currentDir := shared.DirectionOf(from, to)
	if prevDir == DirNone || currentDir == DirNone || prevDir != currentDir {
		return false
	}
	steps := maxInt(absInt(to.Rank()-from.Rank()), absInt(to.File()-from.File()))
	if steps == 0 || steps > 2 {
		return false
	}
	for _, sq := range shared.Line(from, to) {
		if e.board.pieceAt[sq] != nil {
			return false
		}
	}
	return true
}

func (e *Engine) hasFloodWakePushOption(pc *Piece) bool {
	if pc == nil || !pc.Abilities.Contains(AbilityFloodWake) {
		return false
	}
	if e.currentMove == nil || e.currentMove.FloodWakePushUsed {
		return false
	}
	if elementOf(e, pc) != ElementWater {
		return false
	}
	offsets := [...]struct{ dr, df int }{
		{dr: 1, df: 0},
		{dr: -1, df: 0},
		{dr: 0, df: 1},
		{dr: 0, df: -1},
	}
	for _, off := range offsets {
		rank := pc.Square.Rank() + off.dr
		file := pc.Square.File() + off.df
		if sq, valid := shared.SquareFromCoords(rank, file); valid {
			if e.board.pieceAt[sq] == nil {
				return true
			}
		}
	}
	return false
}

func directionStep(dir Direction) (int, int, bool) {
	switch dir {
	case DirN:
		return -1, 0, true
	case DirNE:
		return -1, 1, true
	case DirE:
		return 0, 1, true
	case DirSE:
		return 1, 1, true
	case DirS:
		return 1, 0, true
	case DirSW:
		return 1, -1, true
	case DirW:
		return 0, -1, true
	case DirNW:
		return -1, -1, true
	default:
		return 0, 0, false
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ---------------------------
// Turn State Conditionals
// ---------------------------

// shouldEndTurnAfterCapture checks for abilities that force a turn to end after a capture.
func (e *Engine) shouldEndTurnAfterCapture(pc *Piece) bool {
	element := elementOf(e, pc)

	// Poisonous Meat ends the turn immediately.
	if pc.Abilities.Contains(AbilityPoisonousMeat) {
		appendAbilityNote(&e.board.lastNote, "Poisonous Meat ends the turn")
		return true
	}
	// Overload (Lightning) ends the move after any capture.
	if pc.Abilities.Contains(AbilityOverload) && element == ElementLightning {
		appendAbilityNote(&e.board.lastNote, "Overload ends the turn")
		return true
	}
	// Bastion (Earth) ends the move after a capture ("stop after hit").
	if pc.Abilities.Contains(AbilityBastion) && element == ElementEarth {
		appendAbilityNote(&e.board.lastNote, "Bastion ends the turn")
		return true
	}
	return false
}

// checkPostMoveAbilities checks for abilities that can be activated after a move segment.
func (e *Engine) checkPostMoveAbilities(pc *Piece) {
	// Side Step: Note that an 8-directional nudge is available.
	if pc.Abilities.Contains(AbilitySideStep) && !e.currentMove.SideStepUsed && e.currentMove.RemainingSteps > 0 {
		appendAbilityNote(&e.board.lastNote, "Side Step available (costs 1 step)")
	}
	// Quantum Step: Note that an adjacent relocation is available.
	if pc.Abilities.Contains(AbilityQuantumStep) && !e.currentMove.QuantumStepUsed && e.currentMove.RemainingSteps > 0 {
		appendAbilityNote(&e.board.lastNote, "Quantum Step available (costs 1 step)")
	}
}

// ---------------------------
// Move Legality Helpers
// ---------------------------

func (e *Engine) pathIsPassable(pc *Piece, from, to Square) bool {
	line := shared.Line(from, to)
	if len(line) == 0 {
		return true
	}

	canPhase := e.canPhaseThrough(pc, from, to)
	for _, sq := range line {
		occupant := e.board.pieceAt[sq]
		if occupant == nil {
			continue
		}
		if occupant.Color == pc.Color {
			return false
		}
		if !canPhase {
			return false
		}
		if occupant.Abilities.Contains(AbilityIndomitable) || occupant.Abilities.Contains(AbilityStalwart) {
			return false
		}
	}
	return true
}

func isScatterShotCapture(pc *Piece, from, to Square) bool {
	if pc == nil || !pc.Abilities.Contains(AbilityScatterShot) {
		return false
	}
	if from.Rank() != to.Rank() {
		return false
	}
	fileDiff := from.File() - to.File()
	if fileDiff < 0 {
		fileDiff = -fileDiff
	}
	return fileDiff == 1
}

func (e *Engine) canDirectCapture(attacker, defender *Piece, from, to Square) bool {
	if defender == nil {
		return true
	}
	if attacker == nil {
		return false
	}
	if defender.Abilities.Contains(AbilityStalwart) && rankOf(attacker.Type) < rankOf(defender.Type) {
		return false
	}
	if defender.Abilities.Contains(AbilityBelligerent) && rankOf(attacker.Type) > rankOf(defender.Type) {
		return false
	}
	if isScatterShotCapture(attacker, from, to) && defender.Abilities.Contains(AbilityIndomitable) {
		return false
	}
	return true
}

// wouldLeaveKingInCheck determines whether moving a piece from `from` to `to`
// would result in that piece's own king remaining in or entering check.
func (e *Engine) wouldLeaveKingInCheck(pc *Piece, from, to Square) bool {
	if pc == nil {
		return true
	}

	boardBackup := e.board
	originalSquare := pc.Square

	e.board.pieceAt[from] = nil
	e.board.pieceAt[to] = pc
	pc.Square = to

	inCheck := e.isKingInCheck(pc.Color)

	pc.Square = originalSquare
	e.board = boardBackup

	return inCheck
}

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

// isLegalFirstSegment checks if the initial move is valid.
func (e *Engine) isLegalFirstSegment(pc *Piece, from, to Square) bool {
	// A real implementation uses detailed move generation.
	// For now, this is a simplified check.
	moves := e.generateMoves(pc)
	if !moves.Has(to) {
		return false
	}
	if !e.pathIsPassable(pc, from, to) {
		return false
	}
	target := e.board.pieceAt[to]
	if !e.canDirectCapture(pc, target, from, to) {
		return false
	}
	if e.wouldLeaveKingInCheck(pc, from, to) {
		return false
	}
	return true
}

// isLegalContinuation checks if a subsequent move segment is valid.
func (e *Engine) isLegalContinuation(pc *Piece, from, to Square) bool {
	// For Chain Kill, any legal move is a valid continuation to another capture.
	target := e.board.pieceAt[to]
	if e.currentMove.HasChainKill && target != nil && target.Color != pc.Color {
		return e.isLegalFirstSegment(pc, from, to)
	}

	// For normal continuations, movement rules are more restrictive.
	// This could be, for example, continuing a slide in the same direction.
	// For simplicity, we'll allow any legal move for now.
	if !e.isLegalFirstSegment(pc, from, to) {
		return false
	}
	return true
}
