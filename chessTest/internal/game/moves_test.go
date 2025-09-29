package game

import (
	"fmt"
	"strings"
	"testing"
)

func TestSliderOpeningMovesDoNotPanic(t *testing.T) {
	tests := []struct {
		name     string
		from     string
		to       string
		removals []string
	}{
		{
			name:     "Rook",
			from:     "a1",
			to:       "a3",
			removals: []string{"a2"},
		},
		{
			name:     "Bishop",
			from:     "c1",
			to:       "g5",
			removals: []string{"d2"},
		},
		{
			name:     "Queen",
			from:     "d1",
			to:       "h5",
			removals: []string{"d2", "e2"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			eng := NewEngine()
			if err := eng.SetSideConfig(White, AbilityList{AbilityStalwart}, ElementLight); err != nil {
				t.Fatalf("configure white: %v", err)
			}
			if err := eng.SetSideConfig(Black, AbilityList{AbilityStalwart}, ElementShadow); err != nil {
				t.Fatalf("configure black: %v", err)
			}

			for _, coord := range tt.removals {
				sq, ok := CoordToSquare(coord)
				if !ok {
					t.Fatalf("invalid removal coordinate %q", coord)
				}
				pc := eng.board.pieceAt[sq]
				if pc == nil {
					t.Fatalf("no piece to remove at %s", coord)
				}
				eng.removePiece(pc, sq)
			}

			fromSq, ok := CoordToSquare(tt.from)
			if !ok {
				t.Fatalf("invalid from square %q", tt.from)
			}
			toSq, ok := CoordToSquare(tt.to)
			if !ok {
				t.Fatalf("invalid to square %q", tt.to)
			}

			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("move panicked: %v", r)
				}
			}()

			if err := eng.Move(MoveRequest{From: fromSq, To: toSq, Dir: DirNone}); err != nil {
				t.Fatalf("move returned error: %v", err)
			}
		})
	}
}

func TestMistShroudFreePivot(t *testing.T) {
	eng := NewEngine()
	if err := eng.SetSideConfig(White, AbilityList{AbilityDoOver, AbilityMistShroud, AbilityRadiantVision}, ElementLight); err != nil {
		t.Fatalf("configure white: %v", err)
	}
	if err := eng.SetSideConfig(Black, AbilityList{AbilityDoOver}, ElementShadow); err != nil {
		t.Fatalf("configure black: %v", err)
	}

	if err := removePieceAt(eng, "d2"); err != nil {
		t.Fatalf("remove d2: %v", err)
	}

	from, _ := CoordToSquare("c1")
	turnMid, _ := CoordToSquare("e3")
	final, _ := CoordToSquare("c5")

	if err := eng.Move(MoveRequest{From: from, To: turnMid, Dir: DirNone}); err != nil {
		t.Fatalf("first move: %v", err)
	}
	if err := eng.Move(MoveRequest{From: turnMid, To: final, Dir: DirNone}); err != nil {
		t.Fatalf("pivot move: %v", err)
	}

	if eng.currentMove == nil {
		t.Fatalf("expected current move to remain active after Mist Shroud pivot")
	}
	if got := eng.currentMove.FreeTurnsUsed; got != 1 {
		t.Fatalf("expected one free pivot, got %d", got)
	}
	if got := eng.currentMove.RemainingSteps; got != 1 {
		t.Fatalf("expected 1 remaining step after pivot, got %d", got)
	}
}

func TestUmbralStepPawnMoves(t *testing.T) {
	tests := []struct {
		name           string
		pawnColor      Color
		whiteAbilities AbilityList
		blackAbilities AbilityList
		pawnSquare     string
		backwardSquare string
		captureSquare  string
		captureColor   Color
		expectBackward bool
		expectCapture  bool
	}{
		{
			name:           "WhitePawnWithUmbralStepCanRetreatAndCapture",
			pawnColor:      White,
			whiteAbilities: AbilityList{AbilityDoOver, AbilityUmbralStep},
			blackAbilities: AbilityList{AbilityDoOver},
			pawnSquare:     "d4",
			backwardSquare: "d3",
			captureSquare:  "c3",
			captureColor:   Black,
			expectBackward: true,
			expectCapture:  true,
		},
		{
			name:           "BlackPawnWithUmbralStepCanRetreatAndCapture",
			pawnColor:      Black,
			whiteAbilities: AbilityList{AbilityDoOver},
			blackAbilities: AbilityList{AbilityDoOver, AbilityUmbralStep},
			pawnSquare:     "d5",
			backwardSquare: "d6",
			captureSquare:  "c6",
			captureColor:   White,
			expectBackward: true,
			expectCapture:  true,
		},
		{
			name:           "PawnWithoutUmbralStepRemainsForwardOnly",
			pawnColor:      White,
			whiteAbilities: AbilityList{AbilityDoOver},
			blackAbilities: AbilityList{AbilityDoOver},
			pawnSquare:     "d4",
			backwardSquare: "d3",
			captureSquare:  "c3",
			captureColor:   Black,
			expectBackward: false,
			expectCapture:  false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			eng := NewEngine()
			if err := eng.SetSideConfig(White, tt.whiteAbilities, ElementLight); err != nil {
				t.Fatalf("configure white: %v", err)
			}
			if err := eng.SetSideConfig(Black, tt.blackAbilities, ElementShadow); err != nil {
				t.Fatalf("configure black: %v", err)
			}

			clearBoard(eng)

			whiteKing := mustSquare(t, "e1")
			blackKing := mustSquare(t, "e8")
			eng.placePiece(White, King, whiteKing)
			eng.placePiece(Black, King, blackKing)

			pawnSq := mustSquare(t, tt.pawnSquare)
			eng.placePiece(tt.pawnColor, Pawn, pawnSq)

			if tt.captureSquare != "" {
				captureSq := mustSquare(t, tt.captureSquare)
				eng.placePiece(tt.captureColor, Knight, captureSq)
			}

			pc := eng.board.pieceAt[pawnSq]
			if pc == nil {
				t.Fatalf("no pawn at %s", tt.pawnSquare)
			}

			moves := eng.generatePawnMoves(pc)

			backwardSq := mustSquare(t, tt.backwardSquare)
			if got := moves.Has(backwardSq); got != tt.expectBackward {
				t.Fatalf("backward move to %s expected %t, got %t", tt.backwardSquare, tt.expectBackward, got)
			}

			if tt.captureSquare != "" {
				captureSq := mustSquare(t, tt.captureSquare)
				if got := moves.Has(captureSq); got != tt.expectCapture {
					t.Fatalf("capture move to %s expected %t, got %t", tt.captureSquare, tt.expectCapture, got)
				}
			}
		})
	}
}

func TestBlazeRushDashExtendsMove(t *testing.T) {
	eng := NewEngine()
	if err := eng.SetSideConfig(White, AbilityList{AbilityDoOver, AbilityBlazeRush}, ElementFire); err != nil {
		t.Fatalf("configure white: %v", err)
	}
	if err := eng.SetSideConfig(Black, AbilityList{AbilityDoOver}, ElementShadow); err != nil {
		t.Fatalf("configure black: %v", err)
	}

	if err := removePieceAt(eng, "a2"); err != nil {
		t.Fatalf("remove a2: %v", err)
	}

	from, _ := CoordToSquare("a1")
	slide, _ := CoordToSquare("a4")
	dash, _ := CoordToSquare("a6")

	if err := eng.Move(MoveRequest{From: from, To: slide, Dir: DirNone}); err != nil {
		t.Fatalf("first slide: %v", err)
	}
	if eng.currentMove == nil {
		t.Fatalf("expected Blaze Rush to keep move active")
	}
	if got := eng.currentMove.RemainingSteps; got != 0 {
		t.Fatalf("expected 0 steps after initial slide, got %d", got)
	}

	if err := eng.Move(MoveRequest{From: slide, To: dash, Dir: DirNone}); err != nil {
		t.Fatalf("dash move: %v", err)
	}

	if eng.currentMove != nil {
		t.Fatalf("expected turn to end after Blaze Rush dash")
	}
	dashSq := dash
	pc := eng.board.pieceAt[dashSq]
	if pc == nil || pc.Type != Rook {
		t.Fatalf("expected rook on a6 after dash")
	}
	if !strings.Contains(eng.board.lastNote, "Blaze Rush dash (free)") {
		t.Fatalf("expected note for Blaze Rush dash, got %q", eng.board.lastNote)
	}
}

func TestFloodWakeDisablesPhasing(t *testing.T) {
	eng := NewEngine()
	if err := eng.SetSideConfig(White, AbilityList{AbilityDoOver, AbilityFloodWake, AbilityGaleLift}, ElementWater); err != nil {
		t.Fatalf("configure white: %v", err)
	}
	if err := eng.SetSideConfig(Black, AbilityList{AbilityDoOver}, ElementShadow); err != nil {
		t.Fatalf("configure black: %v", err)
	}

	rookSq, _ := CoordToSquare("a1")
	rook := eng.board.pieceAt[rookSq]
	if rook == nil {
		t.Fatalf("no rook at a1")
	}
	if eng.canPhaseThrough(rook, rookSq, rookSq) {
		t.Fatalf("expected Flood Wake to suppress phasing despite Gale Lift")
	}
}

func TestCastlingKingside(t *testing.T) {
	eng := NewEngine()
	if err := eng.SetSideConfig(White, AbilityList{AbilityStalwart}, ElementLight); err != nil {
		t.Fatalf("configure white: %v", err)
	}
	if err := eng.SetSideConfig(Black, AbilityList{AbilityStalwart}, ElementShadow); err != nil {
		t.Fatalf("configure black: %v", err)
	}

	clearBoard(eng)

	whiteKing := mustSquare(t, "e1")
	whiteRook := mustSquare(t, "h1")
	blackKing := mustSquare(t, "e8")
	eng.placePiece(White, King, whiteKing)
	eng.placePiece(White, Rook, whiteRook)
	eng.placePiece(Black, King, blackKing)
	eng.board.turn = White
	eng.board.Castling = CastlingRight(White, CastleKingside)

	if err := eng.Move(MoveRequest{From: whiteKing, To: mustSquare(t, "g1"), Dir: DirNone}); err != nil {
		t.Fatalf("castling move failed: %v", err)
	}

	kingDest := mustSquare(t, "g1")
	rookDest := mustSquare(t, "f1")
	if pc := eng.board.pieceAt[kingDest]; pc == nil || pc.Type != King || pc.Color != White {
		t.Fatalf("expected white king on g1 after castling")
	}
	if pc := eng.board.pieceAt[rookDest]; pc == nil || pc.Type != Rook || pc.Color != White {
		t.Fatalf("expected white rook on f1 after castling")
	}
	if eng.board.Castling.HasSide(White, CastleKingside) {
		t.Fatalf("expected white kingside castling rights to be cleared")
	}
}

func TestEnPassantCapture(t *testing.T) {
	eng := NewEngine()
	if err := eng.SetSideConfig(White, AbilityList{AbilityStalwart}, ElementLight); err != nil {
		t.Fatalf("configure white: %v", err)
	}
	if err := eng.SetSideConfig(Black, AbilityList{AbilityStalwart}, ElementShadow); err != nil {
		t.Fatalf("configure black: %v", err)
	}

	clearBoard(eng)

	whiteKing := mustSquare(t, "e1")
	blackKing := mustSquare(t, "g8")
	whitePawn := mustSquare(t, "e5")
	blackPawnStart := mustSquare(t, "d7")
	blackPawnLanding := mustSquare(t, "d5")
	enPassantSquare := mustSquare(t, "d6")

	eng.placePiece(White, King, whiteKing)
	eng.placePiece(Black, King, blackKing)
	eng.placePiece(White, Pawn, whitePawn)
	eng.placePiece(Black, Pawn, blackPawnStart)
	eng.board.turn = Black
	eng.board.Castling = CastlingNone
	eng.board.EnPassant = NoEnPassantTarget()

	if err := eng.Move(MoveRequest{From: blackPawnStart, To: blackPawnLanding, Dir: DirNone}); err != nil {
		t.Fatalf("black double move failed: %v", err)
	}

	if sq, ok := eng.board.EnPassant.Square(); !ok || sq != enPassantSquare {
		t.Fatalf("expected en passant target at d6, got %v", eng.board.EnPassant)
	}

	if err := eng.Move(MoveRequest{From: whitePawn, To: enPassantSquare, Dir: DirNone}); err != nil {
		t.Fatalf("en passant capture failed: %v", err)
	}

	if pc := eng.board.pieceAt[enPassantSquare]; pc == nil || pc.Type != Pawn || pc.Color != White {
		t.Fatalf("expected white pawn on d6 after en passant")
	}
	if pc := eng.board.pieceAt[blackPawnLanding]; pc != nil {
		t.Fatalf("expected captured pawn removed from d5")
	}
	if eng.board.EnPassant.Valid() {
		t.Fatalf("expected en passant target cleared after capture")
	}
	if !strings.Contains(eng.board.lastNote, "En passant capture") {
		t.Fatalf("expected en passant note, got %q", eng.board.lastNote)
	}
}

func TestPromotionSelection(t *testing.T) {
	eng := NewEngine()
	if err := eng.SetSideConfig(White, AbilityList{AbilityStalwart}, ElementLight); err != nil {
		t.Fatalf("configure white: %v", err)
	}
	if err := eng.SetSideConfig(Black, AbilityList{AbilityStalwart}, ElementShadow); err != nil {
		t.Fatalf("configure black: %v", err)
	}

	clearBoard(eng)

	whiteKing := mustSquare(t, "e1")
	blackKing := mustSquare(t, "g8")
	pawnStart := mustSquare(t, "e7")
	pawnPromotion := mustSquare(t, "e8")

	eng.placePiece(White, King, whiteKing)
	eng.placePiece(Black, King, blackKing)
	eng.placePiece(White, Pawn, pawnStart)
	eng.board.turn = White
	eng.board.Castling = CastlingNone
	eng.board.PromotionChoices = PromotionChoicesFromTypes(Knight)
	eng.board.EnPassant = NoEnPassantTarget()

	if occ := eng.board.pieceAt[pawnPromotion]; occ != nil {
		t.Fatalf("expected promotion square empty, found %s %s", occ.Color.String(), occ.Type.String())
	}

	moves := eng.generateMoves(eng.board.pieceAt[pawnStart])
	if !moves.Has(pawnPromotion) {
		var squares []string
		moves.Iter(func(s Square) {
			squares = append(squares, s.String())
		})
		t.Fatalf("expected promotion square e8 in generated moves, got %v", squares)
	}

	req := MoveRequest{From: pawnStart, To: pawnPromotion, Dir: DirNone, Promotion: Knight, HasPromotion: true}
	if err := eng.Move(req); err != nil {
		t.Fatalf("promotion move failed: %v", err)
	}

	if pc := eng.board.pieceAt[pawnPromotion]; pc == nil || pc.Type != Knight || pc.Color != White {
		t.Fatalf("expected promoted knight on e8")
	}
	if !strings.Contains(eng.board.lastNote, "Pawn promoted to N") {
		t.Fatalf("expected promotion note, got %q", eng.board.lastNote)
	}
}

func TestMoveRejectedIfSelfCheck(t *testing.T) {
	eng := NewEngine()
	if err := eng.SetSideConfig(White, AbilityList{AbilityDoOver}, ElementLight); err != nil {
		t.Fatalf("configure white: %v", err)
	}
	if err := eng.SetSideConfig(Black, AbilityList{AbilityDoOver}, ElementShadow); err != nil {
		t.Fatalf("configure black: %v", err)
	}

	clearBoard(eng)

	whiteKing := mustSquare(t, "e1")
	whiteRook := mustSquare(t, "e2")
	whiteRookTarget := mustSquare(t, "f2")
	blackRook := mustSquare(t, "e8")

	eng.placePiece(White, King, whiteKing)
	eng.placePiece(White, Rook, whiteRook)
	eng.placePiece(Black, Rook, blackRook)
	eng.board.turn = White

	if err := eng.Move(MoveRequest{From: whiteRook, To: whiteRookTarget, Dir: DirNone}); err == nil {
		t.Fatalf("expected move to be rejected when exposing king to check")
	}

	if pc := eng.board.pieceAt[whiteRook]; pc == nil || pc.Type != Rook {
		t.Fatalf("expected rook to remain on e2 after illegal move")
	}
	if eng.board.turn != White {
		t.Fatalf("expected turn to remain with white after illegal move")
	}
}

func TestCheckmateDetection(t *testing.T) {
	eng := NewEngine()
	if err := eng.SetSideConfig(White, AbilityList{AbilityDoOver}, ElementLight); err != nil {
		t.Fatalf("configure white: %v", err)
	}
	if err := eng.SetSideConfig(Black, AbilityList{AbilityDoOver}, ElementShadow); err != nil {
		t.Fatalf("configure black: %v", err)
	}

	clearBoard(eng)

	whiteKing := mustSquare(t, "g6")
	whiteQueenStart := mustSquare(t, "e7")
	whiteQueenMate := mustSquare(t, "g7")
	blackKing := mustSquare(t, "h8")

	eng.placePiece(White, King, whiteKing)
	eng.placePiece(White, Queen, whiteQueenStart)
	eng.placePiece(Black, King, blackKing)
	eng.board.turn = White

	if err := eng.Move(MoveRequest{From: whiteQueenStart, To: whiteQueenMate, Dir: DirNone}); err != nil {
		t.Fatalf("mate move failed: %v", err)
	}

	if !eng.board.GameOver {
		t.Fatalf("expected game over after checkmate")
	}
	if eng.board.Status != "checkmate" {
		t.Fatalf("expected status 'checkmate', got %q", eng.board.Status)
	}
	if !eng.board.InCheck {
		t.Fatalf("expected side to move to be flagged in check")
	}
	if !eng.board.HasWinner || eng.board.Winner != White {
		t.Fatalf("expected white to be recorded as winner")
	}
	if !strings.Contains(strings.ToLower(eng.board.lastNote), "checkmate") {
		t.Fatalf("expected last note to mention checkmate, got %q", eng.board.lastNote)
	}
}

func TestStalemateDetection(t *testing.T) {
	eng := NewEngine()
	if err := eng.SetSideConfig(White, AbilityList{AbilityDoOver}, ElementLight); err != nil {
		t.Fatalf("configure white: %v", err)
	}
	if err := eng.SetSideConfig(Black, AbilityList{AbilityDoOver}, ElementShadow); err != nil {
		t.Fatalf("configure black: %v", err)
	}

	clearBoard(eng)

	whiteKing := mustSquare(t, "c6")
	whiteQueenStart := mustSquare(t, "c8")
	whiteQueenFinal := mustSquare(t, "c7")
	blackKing := mustSquare(t, "a8")

	eng.placePiece(White, King, whiteKing)
	eng.placePiece(White, Queen, whiteQueenStart)
	eng.placePiece(Black, King, blackKing)
	eng.board.turn = White

	if err := eng.Move(MoveRequest{From: whiteQueenStart, To: whiteQueenFinal, Dir: DirNone}); err != nil {
		t.Fatalf("stalemate move failed: %v", err)
	}

	if !eng.board.GameOver {
		t.Fatalf("expected game over after stalemate setup")
	}
	if eng.board.Status != "stalemate" {
		t.Fatalf("expected status 'stalemate', got %q", eng.board.Status)
	}
	if eng.board.InCheck {
		t.Fatalf("expected side to move not to be in check for stalemate")
	}
	if eng.board.HasWinner {
		t.Fatalf("expected no winner to be recorded for stalemate")
	}
	if !strings.Contains(strings.ToLower(eng.board.lastNote), "stalemate") {
		t.Fatalf("expected last note to mention stalemate, got %q", eng.board.lastNote)
	}
}

func TestFloodWakePushAfterSlide(t *testing.T) {
	eng := NewEngine()
	if err := eng.SetSideConfig(White, AbilityList{AbilityDoOver, AbilityFloodWake}, ElementWater); err != nil {
		t.Fatalf("configure white: %v", err)
	}
	if err := eng.SetSideConfig(Black, AbilityList{AbilityDoOver}, ElementShadow); err != nil {
		t.Fatalf("configure black: %v", err)
	}

	if err := removePieceAt(eng, "a2"); err != nil {
		t.Fatalf("remove a2: %v", err)
	}

	from, _ := CoordToSquare("a1")
	slide, _ := CoordToSquare("a4")
	push, _ := CoordToSquare("a5")

	if err := eng.Move(MoveRequest{From: from, To: slide, Dir: DirNone}); err != nil {
		t.Fatalf("first slide: %v", err)
	}
	if eng.currentMove == nil {
		t.Fatalf("expected Flood Wake push to keep move active")
	}
	if got := eng.currentMove.RemainingSteps; got != 0 {
		t.Fatalf("expected 0 steps after slide, got %d", got)
	}

	if err := eng.Move(MoveRequest{From: slide, To: push, Dir: DirNone}); err != nil {
		t.Fatalf("push move: %v", err)
	}

	pushSq := push
	pc := eng.board.pieceAt[pushSq]
	if pc == nil || pc.Type != Rook {
		t.Fatalf("expected rook on a5 after Flood Wake push")
	}
	if !strings.Contains(eng.board.lastNote, "Flood Wake push (free)") {
		t.Fatalf("expected note for Flood Wake push, got %q", eng.board.lastNote)
	}
}

func TestQuantumStepBlinkConsumesStepAndBlocksSecondUse(t *testing.T) {
	eng := NewEngine()
	if err := eng.SetSideConfig(White, AbilityList{AbilityQuantumStep, AbilitySchrodingersLaugh}, ElementLight); err != nil {
		t.Fatalf("configure white: %v", err)
	}
	if err := eng.SetSideConfig(Black, AbilityList{AbilityDoOver}, ElementShadow); err != nil {
		t.Fatalf("configure black: %v", err)
	}

	clearBoard(eng)

	start := mustSquare(t, "b1")
	firstHop := mustSquare(t, "c3")
	blinkDest := mustSquare(t, "c4")
	secondTarget := mustSquare(t, "d5")
	whiteKing := mustSquare(t, "h1")
	blackKing := mustSquare(t, "h8")

	eng.placePiece(White, King, whiteKing)
	eng.placePiece(Black, King, blackKing)
	eng.placePiece(White, Knight, start)
	eng.placePiece(White, Bishop, secondTarget)
	eng.board.turn = White

	if err := eng.Move(MoveRequest{From: start, To: firstHop, Dir: DirNone}); err != nil {
		t.Fatalf("initial knight move: %v", err)
	}
	if eng.currentMove == nil {
		t.Fatalf("expected move to remain active after first segment")
	}

	remainingBefore := eng.currentMove.RemainingSteps
	if remainingBefore == 0 {
		t.Fatalf("expected steps remaining before quantum step")
	}

	if err := eng.Move(MoveRequest{From: firstHop, To: blinkDest, Dir: DirNone}); err != nil {
		t.Fatalf("quantum step blink: %v", err)
	}

	knight := eng.board.pieceAt[blinkDest]
	if knight == nil || knight.Type != Knight || knight.Color != White {
		t.Fatalf("expected white knight to blink to %s", blinkDest)
	}
	if got := eng.board.pieceAt[firstHop]; got != nil {
		t.Fatalf("expected square %s to be vacated after blink", firstHop)
	}
	if eng.currentMove.RemainingSteps != remainingBefore-1 {
		t.Fatalf("expected steps to decrease by 1, got %d from %d", eng.currentMove.RemainingSteps, remainingBefore)
	}
	if !eng.currentMove.QuantumStepUsed {
		t.Fatalf("expected quantum step flag to be set after blink")
	}

	stepsAfterBlink := eng.currentMove.RemainingSteps

	if err := eng.Move(MoveRequest{From: blinkDest, To: secondTarget, Dir: DirNone}); err == nil {
		t.Fatalf("expected second quantum step attempt to be rejected")
	} else if !strings.Contains(err.Error(), "illegal") {
		t.Fatalf("expected illegal move error, got %v", err)
	}

	if got := eng.board.pieceAt[blinkDest]; got == nil || got.Type != Knight {
		t.Fatalf("expected knight to remain on %s after failed attempt", blinkDest)
	}
	if ally := eng.board.pieceAt[secondTarget]; ally == nil || ally.Type != Bishop {
		t.Fatalf("expected bishop to remain on %s after failed attempt", secondTarget)
	}
	if eng.currentMove.RemainingSteps != stepsAfterBlink {
		t.Fatalf("expected remaining steps to stay at %d after rejection, got %d", stepsAfterBlink, eng.currentMove.RemainingSteps)
	}
}

func TestQuantumStepSwapMovesFriendlyPiece(t *testing.T) {
	eng := NewEngine()
	if err := eng.SetSideConfig(White, AbilityList{AbilityQuantumStep, AbilitySchrodingersLaugh}, ElementLight); err != nil {
		t.Fatalf("configure white: %v", err)
	}
	if err := eng.SetSideConfig(Black, AbilityList{AbilityDoOver}, ElementShadow); err != nil {
		t.Fatalf("configure black: %v", err)
	}

	clearBoard(eng)

	start := mustSquare(t, "b1")
	firstHop := mustSquare(t, "c3")
	swapSquare := mustSquare(t, "c4")
	whiteKing := mustSquare(t, "g1")
	blackKing := mustSquare(t, "g8")

	eng.placePiece(White, King, whiteKing)
	eng.placePiece(Black, King, blackKing)
	eng.placePiece(White, Knight, start)
	eng.placePiece(White, Rook, swapSquare)
	eng.board.turn = White

	if err := eng.Move(MoveRequest{From: start, To: firstHop, Dir: DirNone}); err != nil {
		t.Fatalf("initial knight move: %v", err)
	}
	if eng.currentMove == nil {
		t.Fatalf("expected move to remain active after first segment")
	}

	remainingBefore := eng.currentMove.RemainingSteps
	if remainingBefore == 0 {
		t.Fatalf("expected steps remaining before swap")
	}

	if err := eng.Move(MoveRequest{From: firstHop, To: swapSquare, Dir: DirNone}); err != nil {
		t.Fatalf("quantum step swap: %v", err)
	}

	knight := eng.board.pieceAt[swapSquare]
	if knight == nil || knight.Type != Knight || knight.Color != White {
		t.Fatalf("expected knight to occupy %s after swap", swapSquare)
	}
	rook := eng.board.pieceAt[firstHop]
	if rook == nil || rook.Type != Rook || rook.Color != White {
		t.Fatalf("expected rook to move to %s after swap", firstHop)
	}
	if eng.currentMove.RemainingSteps != remainingBefore-1 {
		t.Fatalf("expected steps to decrease by 1 after swap, got %d from %d", eng.currentMove.RemainingSteps, remainingBefore)
	}
	if !eng.currentMove.QuantumStepUsed {
		t.Fatalf("expected quantum step flag to be set after swap")
	}
}

func TestQuantumKillRemoteRemovalHonorsRank(t *testing.T) {
	eng := NewEngine()
	if err := eng.SetSideConfig(White, AbilityList{AbilityQuantumKill}, ElementLight); err != nil {
		t.Fatalf("configure white: %v", err)
	}
	if err := eng.SetSideConfig(Black, AbilityList{AbilitySideStep}, ElementShadow); err != nil {
		t.Fatalf("configure black: %v", err)
	}

	clearBoard(eng)

	queenSq := mustSquare(t, "d4")
	victimSq := mustSquare(t, "d6")
	bishopSq := mustSquare(t, "a8")
	pawnSq := mustSquare(t, "b8")
	queenGuardSq := mustSquare(t, "h8")

	eng.placePiece(White, Queen, queenSq)
	eng.placePiece(Black, Rook, victimSq)
	eng.placePiece(Black, Bishop, bishopSq)
	eng.placePiece(Black, Pawn, pawnSq)
	eng.placePiece(Black, Queen, queenGuardSq)
	eng.board.turn = White

	if err := eng.Move(MoveRequest{From: queenSq, To: victimSq, Dir: DirNone}); err != nil {
		t.Fatalf("move failed: %v", err)
	}

	if eng.board.pieceAt[bishopSq] != nil {
		t.Fatalf("expected remote bishop at %s to be removed", bishopSq)
	}
	if eng.board.pieceAt[pawnSq] != nil {
		t.Fatalf("expected echo pawn at %s to be removed", pawnSq)
	}
	if eng.board.pieceAt[queenGuardSq] == nil {
		t.Fatalf("expected high-rank queen at %s to survive", queenGuardSq)
	}
}

func TestScatterShotAllowsSideCapture(t *testing.T) {
	eng := NewEngine()
	if err := eng.SetSideConfig(White, AbilityList{AbilityScatterShot}, ElementAir); err != nil {
		t.Fatalf("configure white: %v", err)
	}
	if err := eng.SetSideConfig(Black, AbilityList{AbilitySideStep}, ElementShadow); err != nil {
		t.Fatalf("configure black: %v", err)
	}

	clearBoard(eng)

	attackerSq := mustSquare(t, "d4")
	targetSq := mustSquare(t, "e4")

	eng.placePiece(White, Pawn, attackerSq)
	eng.placePiece(Black, Pawn, targetSq)
	eng.board.turn = White

	if err := eng.Move(MoveRequest{From: attackerSq, To: targetSq, Dir: DirNone}); err != nil {
		t.Fatalf("scatter shot capture failed: %v", err)
	}

	if pc := eng.board.pieceAt[targetSq]; pc == nil || pc.Color != White {
		t.Fatalf("expected white pawn to occupy %s after capture", targetSq)
	}
}

func TestSideStepNudge(t *testing.T) {
	t.Run("white diagonal nudge spends step", func(t *testing.T) {
		eng := NewEngine()
		if err := eng.SetSideConfig(White, AbilityList{AbilitySideStep, AbilitySchrodingersLaugh}, ElementLight); err != nil {
			t.Fatalf("configure white: %v", err)
		}
		if err := eng.SetSideConfig(Black, AbilityList{AbilitySideStep}, ElementShadow); err != nil {
			t.Fatalf("configure black: %v", err)
		}

		clearBoard(eng)

		whiteKing := mustSquare(t, "e1")
		blackKing := mustSquare(t, "e8")
		start := mustSquare(t, "d4")
		first := mustSquare(t, "f5")
		diagonal := mustSquare(t, "g6")

		eng.placePiece(White, King, whiteKing)
		eng.placePiece(Black, King, blackKing)
		eng.placePiece(White, Knight, start)
		eng.board.turn = White

		if err := eng.Move(MoveRequest{From: start, To: first, Dir: DirNone}); err != nil {
			t.Fatalf("first move: %v", err)
		}
		if eng.currentMove == nil {
			t.Fatalf("expected move to remain active after first step")
		}

		stepsBefore := eng.currentMove.RemainingSteps
		if stepsBefore <= 0 {
			t.Fatalf("expected spare steps before side step, got %d", stepsBefore)
		}

		if err := eng.Move(MoveRequest{From: first, To: diagonal, Dir: DirNone}); err != nil {
			t.Fatalf("side step nudge: %v", err)
		}

		if eng.currentMove == nil {
			t.Fatalf("expected move to remain active after diagonal nudge")
		}
		if got := eng.currentMove.RemainingSteps; got != stepsBefore-1 {
			t.Fatalf("expected remaining steps to drop from %d to %d, got %d", stepsBefore, stepsBefore-1, got)
		}
		if !eng.currentMove.SideStepUsed {
			t.Fatalf("expected side step usage to latch")
		}
		if pc := eng.board.pieceAt[diagonal]; pc == nil || pc.Color != White {
			t.Fatalf("expected white knight to occupy %s", diagonal.String())
		}
	})

	t.Run("black orthogonal nudge ends turn", func(t *testing.T) {
		eng := NewEngine()
		if err := eng.SetSideConfig(White, AbilityList{AbilitySideStep}, ElementLight); err != nil {
			t.Fatalf("configure white: %v", err)
		}
		if err := eng.SetSideConfig(Black, AbilityList{AbilitySideStep, AbilityBelligerent}, ElementShadow); err != nil {
			t.Fatalf("configure black: %v", err)
		}

		clearBoard(eng)

		whiteKing := mustSquare(t, "d1")
		blackKing := mustSquare(t, "d8")
		start := mustSquare(t, "e5")
		first := mustSquare(t, "c4")
		orth := mustSquare(t, "c3")

		eng.placePiece(White, King, whiteKing)
		eng.placePiece(Black, King, blackKing)
		eng.placePiece(Black, Knight, start)
		eng.board.turn = Black

		if err := eng.Move(MoveRequest{From: start, To: first, Dir: DirNone}); err != nil {
			t.Fatalf("first move: %v", err)
		}
		if eng.currentMove == nil {
			t.Fatalf("expected move to remain active after first step")
		}
		if got := eng.currentMove.RemainingSteps; got != 1 {
			t.Fatalf("expected exactly one step remaining before nudge, got %d", got)
		}

		if err := eng.Move(MoveRequest{From: first, To: orth, Dir: DirNone}); err != nil {
			t.Fatalf("side step nudge: %v", err)
		}

		if eng.currentMove != nil {
			t.Fatalf("expected turn to end once steps were exhausted")
		}
		if eng.board.turn != White {
			t.Fatalf("expected turn to pass to white, got %s", eng.board.turn)
		}
		if pc := eng.board.pieceAt[orth]; pc == nil || pc.Color != Black {
			t.Fatalf("expected black knight to occupy %s", orth.String())
		}
	})
}

func TestStalwartBlocksLowerRankCapture(t *testing.T) {
	eng := NewEngine()
	if err := eng.SetSideConfig(White, AbilityList{AbilitySideStep}, ElementLight); err != nil {
		t.Fatalf("configure white: %v", err)
	}
	if err := eng.SetSideConfig(Black, AbilityList{AbilityStalwart}, ElementEarth); err != nil {
		t.Fatalf("configure black: %v", err)
	}

	clearBoard(eng)

	knightSq := mustSquare(t, "c3")
	rookSq := mustSquare(t, "d5")

	eng.placePiece(White, Knight, knightSq)
	eng.placePiece(Black, Rook, rookSq)
	eng.board.turn = White

	if err := eng.Move(MoveRequest{From: knightSq, To: rookSq, Dir: DirNone}); err == nil {
		t.Fatalf("expected stalwart to block lower-rank capture")
	}
}

func TestIndomitableBlocksAbilityRemoval(t *testing.T) {
	eng := NewEngine()
	if err := eng.SetSideConfig(White, AbilityList{AbilityQuantumKill}, ElementLight); err != nil {
		t.Fatalf("configure white: %v", err)
	}
	if err := eng.SetSideConfig(Black, AbilityList{AbilityIndomitable}, ElementShadow); err != nil {
		t.Fatalf("configure black: %v", err)
	}

	clearBoard(eng)

	queenSq := mustSquare(t, "d4")
	victimSq := mustSquare(t, "d6")
	targetSq := mustSquare(t, "a8")

	eng.placePiece(White, Queen, queenSq)
	eng.placePiece(Black, Rook, victimSq)
	eng.placePiece(Black, Bishop, targetSq)
	eng.board.turn = White

	if err := eng.Move(MoveRequest{From: queenSq, To: victimSq, Dir: DirNone}); err != nil {
		t.Fatalf("move failed: %v", err)
	}

	if eng.board.pieceAt[targetSq] == nil {
		t.Fatalf("expected indomitable piece at %s to survive ability removal", targetSq)
	}
}

func TestResurrectionCaptureWindowExpires(t *testing.T) {
	eng := NewEngine()
	if err := eng.SetSideConfig(White, AbilityList{AbilityResurrection, AbilitySchrodingersLaugh, AbilityTailwind, AbilityChainKill}, ElementAir); err != nil {
		t.Fatalf("configure white: %v", err)
	}
	if err := eng.SetSideConfig(Black, AbilityList{AbilitySideStep}, ElementShadow); err != nil {
		t.Fatalf("configure black: %v", err)
	}

	clearBoard(eng)

	start := mustSquare(t, "c3")
	firstCapture := mustSquare(t, "e5")
	verticalCapture := mustSquare(t, "e6")
	pivot := mustSquare(t, "f7")
	blocked := mustSquare(t, "f8")

	eng.placePiece(White, Bishop, start)
	eng.placePiece(Black, Knight, firstCapture)
	eng.placePiece(Black, Rook, verticalCapture)
	eng.placePiece(Black, Queen, blocked)
	eng.board.turn = White

	if steps := eng.calculateStepBudget(eng.board.pieceAt[start]); steps < 3 {
		t.Fatalf("expected at least 3 steps with buffs, got %d", steps)
	}
	if eng.abilities[Black].Contains(AbilityDoOver) {
		t.Fatalf("black side unexpectedly configured with Do-Over")
	}
	if pc := eng.board.pieceAt[firstCapture]; pc == nil {
		t.Fatalf("expected black piece at %s", firstCapture)
	} else if pc.Abilities.Contains(AbilityDoOver) {
		t.Fatalf("test setup gave victim Do-Over unexpectedly")
	}

	if err := eng.Move(MoveRequest{From: start, To: firstCapture, Dir: DirNone}); err != nil {
		t.Fatalf("first capture: %v", err)
	}

	if eng.board.turn != White {
		t.Fatalf("expected white to retain turn after capture, got %s", eng.board.turn)
	}

	if err := eng.Move(MoveRequest{From: firstCapture, To: verticalCapture, Dir: DirNone}); err != nil {
		t.Fatalf("vertical capture: %v", err)
	}

	if err := eng.Move(MoveRequest{From: verticalCapture, To: pivot, Dir: DirNone}); err != nil {
		t.Fatalf("pivot move: %v", err)
	}

	if err := eng.Move(MoveRequest{From: pivot, To: blocked, Dir: DirNone}); err == nil {
		t.Fatalf("expected vertical capture window to expire after one segment")
	}
}

func TestTemporalLockAppliesSlowToNextMover(t *testing.T) {
	eng := NewEngine()
	if err := eng.SetSideConfig(White, AbilityList{AbilityTemporalLock}, ElementFire); err != nil {
		t.Fatalf("configure white: %v", err)
	}
	if err := eng.SetSideConfig(Black, AbilityList{AbilitySchrodingersLaugh}, ElementLight); err != nil {
		t.Fatalf("configure black: %v", err)
	}

	clearBoard(eng)

	whiteStart := mustSquare(t, "e4")
	whiteLanding := mustSquare(t, "f6")
	blackStart := mustSquare(t, "a7")
	blackLanding := mustSquare(t, "a6")

	eng.placePiece(White, Knight, whiteStart)
	eng.placePiece(Black, Rook, blackStart)
	eng.board.turn = White

	if err := eng.Move(MoveRequest{From: whiteStart, To: whiteLanding, Dir: DirNone}); err != nil {
		t.Fatalf("white move: %v", err)
	}

	if eng.board.turn != Black {
		t.Fatalf("expected turn to pass to black after white move")
	}

	if err := eng.Move(MoveRequest{From: blackStart, To: blackLanding, Dir: DirNone}); err != nil {
		t.Fatalf("black move: %v", err)
	}

	if eng.board.turn != White {
		t.Fatalf("expected Temporal Lock slow to end black's turn immediately")
	}
	if eng.currentMove != nil {
		t.Fatalf("expected no active move after slow applied")
	}
}

func removePieceAt(eng *Engine, coord string) error {
	sq, ok := CoordToSquare(coord)
	if !ok {
		return fmt.Errorf("invalid square %s", coord)
	}
	pc := eng.board.pieceAt[sq]
	if pc == nil {
		return fmt.Errorf("no piece at %s", coord)
	}
	eng.removePiece(pc, sq)
	return nil
}

func clearBoard(eng *Engine) {
	for idx, pc := range eng.board.pieceAt {
		if pc != nil {
			eng.removePiece(pc, Square(idx))
		}
	}
}

func mustSquare(t *testing.T, coord string) Square {
	t.Helper()
	sq, ok := CoordToSquare(coord)
	if !ok {
		t.Fatalf("invalid coordinate %s", coord)
	}
	return sq
}
