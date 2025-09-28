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
	TurnEnded             bool
	FreeTurnsUsed         int  // For Mist Shroud
	BlazeRushUsed         bool // Once per turn
	QuantumStepUsed       bool // Once per turn
	SideStepUsed          bool // Once per turn
	TemporalLockUsed      bool // Once per turn
	ChainKillCaptureCount int  // Track Chain Kill captures (max 2 additional)
	FloodWakePushUsed     bool // Track Flood Wake push usage
	ResurrectionQueue     []*Piece
	MaxCaptures           int
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
		ms.ResurrectionQueue = append(ms.ResurrectionQueue, captured)
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
	}

	// The move is now valid, push the state before executing.
	e.pushHistory()
	e.executeMoveSegment(from, to)

	// Handle capture abilities if a piece was taken.
	if target != nil {
		e.currentMove.registerCapture(target)
		if err := e.ResolveCaptureAbility(pc, target, to); err != nil {
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
	if target != nil && e.shouldEndTurnAfterCapture(pc) {
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
	if e.currentMove.RemainingSteps <= 0 || e.currentMove.TurnEnded {
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

	// Validate the legality of the continuation move.
	if !e.isLegalContinuation(pc, from, to) {
		return errors.New("illegal move continuation")
	}

	stepsNeeded := e.calculateMovementCost(pc, from, to)
	if stepsNeeded > e.currentMove.RemainingSteps {
		return fmt.Errorf("insufficient steps: %d needed, %d remaining", stepsNeeded, e.currentMove.RemainingSteps)
	}

	target := e.board.pieceAt[to]
	if target != nil && target.Color == pc.Color {
		return errors.New("cannot capture a friendly piece")
	}
	if blocked, note := e.captureBlockedByBlockPath(pc, from, target, to); blocked {
		appendAbilityNote(&e.board.lastNote, note)
		return ErrCaptureBlocked
	}

	// Continuation is valid, push state and execute.
	e.pushHistory()
	e.currentMove.RemainingSteps -= stepsNeeded
	e.executeMoveSegment(from, to)

	if target != nil {
		e.currentMove.registerCapture(target)
		if err := e.ResolveCaptureAbility(pc, target, to); err != nil {
			e.currentMove = nil
			return err
		}
		if !e.currentMove.canCaptureMore() {
			e.endTurn()
			return nil
		}
	}

	// Check for turn-ending conditions after the action.
	if e.checkPostCaptureTermination(pc, target) || e.currentMove.RemainingSteps <= 0 {
		e.endTurn()
	} else {
		appendAbilityNote(&e.board.lastNote, fmt.Sprintf("%d steps remaining", e.currentMove.RemainingSteps))
	}

	return nil
}

// endTurn finalizes the move, performs cleanup, and passes control to the other player.
func (e *Engine) endTurn() {
	if e.currentMove == nil {
		// This can happen if a move was aborted (e.g., DoOver).
		return
	}

	pc := e.currentMove.Piece
	e.resolvePromotion(pc)
	e.handleResurrectionQueue()
	e.flipTurn()
	appendAbilityNote(&e.board.lastNote, fmt.Sprintf("%s's turn", e.board.turn))

	// Clear the current move state, officially ending the turn.
	e.currentMove = nil
}

func (e *Engine) handleResurrectionQueue() {
	if e.currentMove == nil || len(e.currentMove.ResurrectionQueue) == 0 {
		return
	}

	reviveSquare := e.findResurrectionSquare(e.currentMove.Piece)
	if reviveSquare == -1 {
		return
	}

	revived := e.currentMove.ResurrectionQueue[len(e.currentMove.ResurrectionQueue)-1]
	e.currentMove.ResurrectionQueue = e.currentMove.ResurrectionQueue[:len(e.currentMove.ResurrectionQueue)-1]

	e.placeRevivedPiece(revived, Square(reviveSquare))
	appendAbilityNote(&e.board.lastNote, fmt.Sprintf("%s %s resurrected at %s", revived.Color, revived.Type, Square(reviveSquare)))
}

func (e *Engine) findResurrectionSquare(pc *Piece) int {
	if pc == nil {
		return -1
	}

	deltas := []int{8, -8}
	for _, delta := range deltas {
		sqInt := int(pc.Square) + delta
		if sqInt >= 0 && sqInt < 64 {
			sq := Square(sqInt)
			if e.board.pieceAt[sq] == nil {
				return int(sq)
			}
		}
	}

	return -1
}

func (e *Engine) placeRevivedPiece(pc *Piece, sq Square) {
	if pc == nil {
		return
	}
	pc.Square = sq
	pc.Element = elementOf(e, pc)
	e.board.pieceAt[sq] = pc
	e.board.pieces[pc.Color][pc.Type] = e.board.pieces[pc.Color][pc.Type].Add(sq)
	e.board.occupancy[pc.Color] = e.board.occupancy[pc.Color].Add(sq)
	e.board.allOcc = e.board.allOcc.Add(sq)
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
	}
	if pc.Abilities.Contains(AbilityRadiantVision) && element == ElementLight {
		bonus++ // Radiant Vision grants +1 step
	}
	if pc.Abilities.Contains(AbilityUmbralStep) && element == ElementShadow {
		bonus += 2 // Umbral Step grants +2 steps
	}
	if pc.Abilities.Contains(AbilityBelligerent) {
		bonus++ // Belligerent grants +1 step
	}
	if pc.Abilities.Contains(AbilitySchrodingersLaugh) {
		bonus += 2 // Schrodinger's Laugh grants +2 steps
		if pc.Abilities.Contains(AbilitySideStep) {
			bonus++ // Interaction bonus with Side Step
		}
	}
	if pc.Abilities.Contains(AbilityPoisonousMeat) {
		bonus-- // Poisonous Meat costs 1 step
		if element == ElementShadow {
			bonus++ // Shadow affinity negates the penalty
		}
	}

	// TODO: Add slow penalty from Temporal Lock
	// slowPenalty := 0
	// if slowAmount, ok := e.slowedPieces[pc.ID]; ok { ... }

	totalSteps := baseSteps + bonus
	if totalSteps < 1 {
		return 1 // A piece always gets at least 1 step.
	}
	return totalSteps
}

// calculateMovementCost calculates the step cost for a given move segment.
func (e *Engine) calculateMovementCost(pc *Piece, from, to Square) int {
	cost := 1 // Basic movement costs 1 step.

	// Sliders (Queen, Rook, Bishop) pay an extra step to change direction mid-turn.
	if e.isSlider(pc.Type) && len(e.currentMove.Path) > 1 {
		prevFrom := e.currentMove.Path[len(e.currentMove.Path)-2]
		prevDir := shared.DirectionOf(prevFrom, from)
		currentDir := shared.DirectionOf(from, to)

		if prevDir != currentDir && prevDir != DirNone && currentDir != DirNone {
			cost++ // Direction change costs an extra step.
			appendAbilityNote(&e.board.lastNote, "Direction change cost +1 step")
		}
	}
	return cost
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

// isLegalFirstSegment checks if the initial move is valid.
func (e *Engine) isLegalFirstSegment(pc *Piece, from, to Square) bool {
	// A real implementation uses detailed move generation.
	// For now, this is a simplified check.
	moves := e.generateMoves(pc)
	if !moves.Has(to) {
		return false
	}
	if !e.canPhaseThrough(pc, from, to) && !e.isPathClear(from, to) {
		return false // Path is blocked
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
	return e.isLegalFirstSegment(pc, from, to)
}
