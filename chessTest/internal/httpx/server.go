// path: chessTest/internal/httpx/server.go
package httpx

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"battle_chess_poc/internal/game"
)

// Server wires the HTTP layer to the chess engine and templates.
type Server struct {
	engineMu  sync.RWMutex
	engine    *game.Engine
	tmpl      *template.Template
	abilities []string
	elements  []string
}

// NewServer builds a Server, parses templates, and precomputes option lists.
func NewServer(engine *game.Engine) (*Server, error) {
	if engine == nil {
		return nil, fmt.Errorf("nil engine")
	}
	tmpl, err := template.ParseFiles("web/templates/index.html")
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}
	return &Server{
		engine:    engine,
		tmpl:      tmpl,
		abilities: game.AbilityStrings(),
		elements:  game.ElementStrings(),
	}, nil
}

// Listen starts the HTTP server.
func (s *Server) Listen(addr string) error {
	server := &http.Server{
		Addr:              addr,
		Handler:           s.routes(),
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 12,
	}
	return server.ListenAndServe()
}

// routes configures the ServeMux with UI, JSON APIs, static files.
func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)

	// JSON APIs
	mux.HandleFunc("/api/state", s.withJSON(s.handleState))
	mux.HandleFunc("/api/move", s.withJSON(s.handleMove))
	mux.HandleFunc("/api/config", s.withJSON(s.handleConfig))
	mux.HandleFunc("/api/reset", s.withJSON(s.handleReset)) // <-- NEW

	// Static assets under /static/
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	// Health
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	return mux
}

// ---- UI ----

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	s.engineMu.RLock()
	state := s.engine.State()
	s.engineMu.RUnlock()
	init := struct {
		State     game.BoardState `json:"state"`
		Abilities []string        `json:"abilities"`
		Elements  []string        `json:"elements"`
	}{
		State:     state,
		Abilities: s.abilities,
		Elements:  s.elements,
	}
	payload, err := marshalInit(init)
	if err != nil {
		log.Printf("init marshal: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	data := map[string]any{"Init": payload}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Execute template "index" from parsed file.
	if err := s.tmpl.ExecuteTemplate(w, "index", data); err != nil {
		log.Printf("template exec: %v", err)
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
}

// ---- JSON helpers ----

const maxBodyBytes = 64 << 10

func (s *Server) withJSON(h func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
			defer r.Body.Close()
		}
		h(w, r)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		log.Printf("json encode: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	writeJSON(w, map[string]string{"error": msg})
}

func marshalInit(v any) (template.JS, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return template.JS(b), nil
}

// ---- API: state ----

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	s.engineMu.RLock()
	state := s.engine.State()
	s.engineMu.RUnlock()
	writeJSON(w, map[string]any{"state": state})
}

// ---- API: move ----

type moveBody struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Dir       string `json:"dir"` // optional: N,NE,E,SE,S,SW,W,NW or "" (auto)
	Promotion string `json:"promotion"`
}

func (s *Server) handleMove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body moveBody
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	from, ok := game.CoordToSquare(toLower(body.From))
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid from square")
		return
	}
	to, ok := game.CoordToSquare(toLower(body.To))
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid to square")
		return
	}
	dir := parseDirection(body.Dir)

	req := game.MoveRequest{From: from, To: to, Dir: dir}
	if promotion := trim(body.Promotion); promotion != "" {
		pt, ok := game.ParsePromotionPiece(promotion)
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid promotion choice")
			return
		}
		req.Promotion = pt
		req.HasPromotion = true
	}

	s.engineMu.Lock()
	err := s.engine.Move(req)
	state := s.engine.State()
	s.engineMu.Unlock()

	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, map[string]any{"state": state})
}

// ---- API: config ----

type configBody struct {
	Color     string   `json:"color"`
	Abilities []string `json:"abilities"`
	Element   string   `json:"element"`
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body configBody
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	color, ok := parseColor(body.Color)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid color")
		return
	}
	abilityList, err := parseAbilities(body.Abilities)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	element, ok := parseElement(body.Element)
	if !ok {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid element %q", body.Element))
		return
	}

	s.engineMu.Lock()
	err = s.engine.SetSideConfig(color, abilityList, element)
	state := s.engine.State()
	s.engineMu.Unlock()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, map[string]any{"state": state})
}

// ---- API: reset (NEW) ----

func (s *Server) handleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	s.engineMu.Lock()
	err := s.engine.Reset()
	state := s.engine.State()
	s.engineMu.Unlock()

	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, map[string]any{"state": state})
}

// ---- parsing helpers ----

func parseColor(s string) (game.Color, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "white", "w":
		return game.White, true
	case "black", "b":
		return game.Black, true
	default:
		return 0, false
	}
}

func parseElement(s string) (game.Element, bool) {
	elem, ok := game.ParseElement(s)
	return elem, ok
}

func parseDirection(s string) game.Direction {
	dir := trim(s)
	if dir == "" || strings.EqualFold(dir, "auto") {
		return game.DirNone
	}
	if parsed, ok := game.ParseDirection(strings.ToUpper(dir)); ok {
		return parsed
	}
	return game.DirNone
}

func parseAbilities(list []string) (game.AbilityList, error) {
	if len(list) == 0 {
		return nil, fmt.Errorf("abilities cannot be empty")
	}
	abilities := make(game.AbilityList, 0, len(list))
	for _, item := range list {
		ability, ok := game.ParseAbility(item)
		if !ok {
			return nil, fmt.Errorf("invalid ability %q", item)
		}
		abilities = append(abilities, ability)
	}
	return abilities, nil
}

func trim(s string) string { return strings.TrimSpace(s) }

func toLower(s string) string { return strings.ToLower(trim(s)) }
