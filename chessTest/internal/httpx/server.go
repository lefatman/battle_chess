// path: chessTest/internal/httpx/server.go
package httpx

import (
	"context"
	"encoding/json"
	"errors"
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
	engineMu  sync.Mutex
	engine    *game.Engine
	tmpl      *template.Template
	abilities []string
	elements  []string
	srvMu     sync.Mutex
	srv       *http.Server
}

const (
	maxJSONBodyBytes int64 = 1 << 20
	htmlCSP                = "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'"
	apiCSP                 = "default-src 'none'; frame-ancestors 'none'; base-uri 'none'"
)

// NewServer builds a Server, parses templates, and precomputes option lists.
func NewServer(engine *game.Engine) *Server {
	// Expect file at web/templates/index.html with: {{define "index"}}...{{end}}
	t := template.Must(template.ParseFiles("web/templates/index.html"))

	s := &Server{
		engine:    engine,
		tmpl:      t,
		abilities: abilityNames(),
		elements:  elementNames(),
	}
	return s
}

// Listen starts the HTTP server.
func (s *Server) Listen(addr string) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           s.routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 16,
	}

	s.srvMu.Lock()
	s.srv = srv
	s.srvMu.Unlock()
	defer func() {
		s.srvMu.Lock()
		s.srv = nil
		s.srvMu.Unlock()
	}()

	log.Printf("HTTP listening on %s", addr)
	err := srv.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Close attempts a graceful shutdown of the HTTP server.
func (s *Server) Close(ctx context.Context) error {
	s.srvMu.Lock()
	srv := s.srv
	s.srvMu.Unlock()
	if srv == nil {
		return nil
	}
	return srv.Shutdown(ctx)
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
	applyHTMLSecurityHeaders(w.Header())
	// Build initial payload embedding current engine state and option lists.
	s.engineMu.Lock()
	state := s.engine.State()
	s.engineMu.Unlock()
	init := struct {
		State     game.BoardState `json:"state"`
		Abilities []string        `json:"abilities"`
		Elements  []string        `json:"elements"`
	}{
		State:     state,
		Abilities: s.abilities,
		Elements:  s.elements,
	}
	data := map[string]any{
		"Init": mustJSON(init),
	}
	// Execute template "index" from parsed file.
	if err := s.tmpl.ExecuteTemplate(w, "index", data); err != nil {
		log.Printf("template exec: %v", err)
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
}

// ---- JSON helpers ----

func (s *Server) withJSON(h func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		applyAPISecurityHeaders(w.Header())
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if r.Body != nil && r.Body != http.NoBody {
			r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
		}
		h(w, r)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	writeJSON(w, map[string]string{"error": msg})
}

func mustJSON(v any) template.JS {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return template.JS(b)
}

func applyHTMLSecurityHeaders(h http.Header) {
	h.Set("Content-Security-Policy", htmlCSP)
	h.Set("Cross-Origin-Opener-Policy", "same-origin")
	h.Set("Cross-Origin-Embedder-Policy", "require-corp")
}

func applyAPISecurityHeaders(h http.Header) {
	h.Set("Content-Security-Policy", apiCSP)
	h.Set("Cross-Origin-Opener-Policy", "same-origin")
	h.Set("Cross-Origin-Embedder-Policy", "require-corp")
}

func isBodyTooLarge(err error) bool {
	var maxErr *http.MaxBytesError
	return errors.As(err, &maxErr)
}

// ---- API: state ----

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	s.engineMu.Lock()
	state := s.engine.State()
	s.engineMu.Unlock()
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
	defer r.Body.Close()
	var body moveBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		if isBodyTooLarge(err) {
			writeError(w, http.StatusRequestEntityTooLarge, "request too large")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	from, ok := game.CoordToSquare(strings.ToLower(strings.TrimSpace(body.From)))
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid from square")
		return
	}
	to, ok := game.CoordToSquare(strings.ToLower(strings.TrimSpace(body.To)))
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid to square")
		return
	}
	dir := parseDirection(body.Dir)

	req := game.MoveRequest{From: from, To: to, Dir: dir}
	if promotion := strings.TrimSpace(body.Promotion); promotion != "" {
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
	defer r.Body.Close()
	var body configBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		if isBodyTooLarge(err) {
			writeError(w, http.StatusRequestEntityTooLarge, "request too large")
			return
		}
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
	if r.Body != nil {
		r.Body.Close()
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

func parseAbility(s string) (game.Ability, bool) {
	if s == "" {
		return game.AbilityNone, false
	}
	needle := strings.ToLower(strings.TrimSpace(s))
	for _, a := range game.AllAbilities {
		if strings.ToLower(a.String()) == needle {
			return a, true
		}
	}
	return game.AbilityNone, false
}

func parseElement(s string) (game.Element, bool) {
	if s == "" {
		return game.ElementLight, false
	}
	needle := strings.ToLower(strings.TrimSpace(s))
	for _, e := range game.AllElements {
		if strings.ToLower(e.String()) == needle {
			return e, true
		}
	}
	return game.ElementLight, false
}

func parseDirection(s string) game.Direction {
	needle := strings.ToUpper(strings.TrimSpace(s))
	switch needle {
	case "", "AUTO":
		return game.DirNone
	case "N":
		return game.DirN
	case "NE":
		return game.DirNE
	case "E":
		return game.DirE
	case "SE":
		return game.DirSE
	case "S":
		return game.DirS
	case "SW":
		return game.DirSW
	case "W":
		return game.DirW
	case "NW":
		return game.DirNW
	default:
		return game.DirNone
	}
}

func parseAbilities(list []string) (game.AbilityList, error) {
	abilities := make(game.AbilityList, 0, len(list))
	for _, item := range list {
		ability, ok := parseAbility(item)
		if !ok {
			return nil, fmt.Errorf("invalid ability %q", item)
		}
		abilities = append(abilities, ability)
	}
	return abilities, nil
}

func abilityNames() []string {
	out := make([]string, 0, len(game.AllAbilities))
	for _, a := range game.AllAbilities {
		out = append(out, a.String())
	}
	return out
}

func elementNames() []string {
	out := make([]string, 0, len(game.AllElements))
	for _, e := range game.AllElements {
		out = append(out, e.String())
	}
	return out
}
