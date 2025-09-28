package game

import "testing"

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
