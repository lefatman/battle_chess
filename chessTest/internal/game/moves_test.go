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
			if err := eng.SetSideConfig(White, AbilityList{AbilityDoOver}, ElementLight); err != nil {
				t.Fatalf("configure white: %v", err)
			}
			if err := eng.SetSideConfig(Black, AbilityList{AbilityDoOver}, ElementShadow); err != nil {
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
