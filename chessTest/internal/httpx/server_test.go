// path: chessTest/internal/httpx/server_test.go
package httpx

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"battle_chess_poc/internal/game"
)

func TestHandleMoveDoOverReturnsState(t *testing.T) {
	eng := game.NewEngine()
	if err := eng.SetSideConfig(game.White, game.AbilityList{game.AbilityDoOver}, game.ElementLight); err != nil {
		t.Fatalf("configure white: %v", err)
	}
	if err := eng.SetSideConfig(game.Black, game.AbilityList{game.AbilityDoOver}, game.ElementShadow); err != nil {
		t.Fatalf("configure black: %v", err)
	}

	e2, _ := game.CoordToSquare("e2")
	e4, _ := game.CoordToSquare("e4")
	if err := eng.Move(game.MoveRequest{From: e2, To: e4, Dir: game.DirNone}); err != nil {
		t.Fatalf("white opening move: %v", err)
	}
	d7, _ := game.CoordToSquare("d7")
	d5, _ := game.CoordToSquare("d5")
	if err := eng.Move(game.MoveRequest{From: d7, To: d5, Dir: game.DirNone}); err != nil {
		t.Fatalf("black reply move: %v", err)
	}

	srv := &Server{engine: eng}

	reqBody := `{"from":"e4","to":"d5","dir":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/move", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	srv.handleMove(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var payload struct {
		State   game.BoardState `json:"state"`
		Message string          `json:"message"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if payload.Message != game.ErrDoOverActivated.Error() {
		t.Fatalf("expected DoOver message, got %q", payload.Message)
	}
	if len(payload.State.Pieces) == 0 {
		t.Fatalf("expected non-empty state payload")
	}
	if payload.State.Turn != game.White {
		t.Fatalf("expected rewound turn to white, got %s", payload.State.Turn)
	}
	if !strings.Contains(strings.ToLower(payload.State.LastNote), "doover") &&
		!strings.Contains(strings.ToLower(payload.State.LastNote), "do over") {
		t.Fatalf("expected last note to mention do-over, got %q", payload.State.LastNote)
	}
}

func TestHandleConfigAbilityAliases(t *testing.T) {
	cases := []struct {
		name      string
		abilities []string
	}{
		{
			name:      "CamelCase",
			abilities: []string{"DoOver", "MistShroud"},
		},
		{
			name:      "SpacedNames",
			abilities: []string{"Do Over", "Mist Shroud"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			eng := game.NewEngine()
			srv := &Server{engine: eng}

			body, err := json.Marshal(configBody{
				Color:     "white",
				Abilities: tc.abilities,
				Element:   "light",
			})
			if err != nil {
				t.Fatalf("marshal body: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			srv.handleConfig(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d", rr.Code)
			}

			var resp struct {
				State game.BoardState `json:"state"`
			}
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Fatalf("decode body: %v", err)
			}

			expected := []string{"DoOver", "MistShroud"}
			if got := resp.State.Abilities[game.White.String()]; !equalStringSlices(got, expected) {
				t.Fatalf("unexpected ability list: got %v want %v", got, expected)
			}

			state := srv.engine.State()
			if got := state.Abilities[game.White.String()]; !equalStringSlices(got, expected) {
				t.Fatalf("engine state abilities mismatch: got %v want %v", got, expected)
			}
		})
	}
}

func TestFirstMoveAllowsBlockPathDirectionBeforeLock(t *testing.T) {
	eng := game.NewEngine()
	srv := &Server{engine: eng}

	applyConfig := func(color string, abilities []string, element string) {
		body, err := json.Marshal(configBody{Color: color, Abilities: abilities, Element: element})
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		srv.handleConfig(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("/api/config %s status = %d", color, rr.Code)
		}
	}

	applyConfig("white", []string{"BlockPath"}, "light")
	applyConfig("black", []string{"DoOver"}, "shadow")

	if srv.engine.State().Locked {
		t.Fatal("engine locked before first move")
	}

	moveBody := `{"from":"e2","to":"e4","dir":"N"}`
	req := httptest.NewRequest(http.MethodPost, "/api/move", strings.NewReader(moveBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	srv.handleMove(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("/api/move status = %d", rr.Code)
	}

	var payload struct {
		State game.BoardState `json:"state"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode move response: %v", err)
	}

	e4, _ := game.CoordToSquare("e4")
	var moved game.PieceState
	for _, pc := range payload.State.Pieces {
		if pc.Square == e4 {
			moved = pc
			break
		}
	}
	if moved.ID == 0 {
		t.Fatal("moved piece not found in response state")
	}
	if dir, ok := payload.State.BlockFacing[moved.ID]; !ok || dir != game.DirN {
		t.Fatalf("expected block direction N, got %v (ok=%v)", dir, ok)
	}
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
