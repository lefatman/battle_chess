// path: chessTest/internal/httpx/server_test.go
package httpx

import (
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
