// path: chessTest/internal/game/abilities/handler_test.go
package abilities

import (
	"errors"
	"testing"

	"battle_chess_poc/internal/game"
)

func resetRegistry(t *testing.T) {
	t.Helper()
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = make(map[game.Ability]HandlerFactory)
}

func TestRegisterAndNew(t *testing.T) {
	resetRegistry(t)

	handler := HandlerFuncs{
		OnMoveStartFunc: func(MoveLifecycleContext) error {
			return nil
		},
	}

	if err := Register(game.AbilityScorch, func() AbilityHandler { return handler }); err != nil {
		t.Fatalf("register ability: %v", err)
	}

	instance, err := New(game.AbilityScorch)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	if _, err := instance.StepBudgetModifier(StepBudgetContext{}); err != nil {
		t.Fatalf("step budget modifier: %v", err)
	}
	if err := instance.OnMoveStart(MoveLifecycleContext{}); err != nil {
		t.Fatalf("on move start: %v", err)
	}
}

func TestRegisterDuplicate(t *testing.T) {
	resetRegistry(t)

	ctor := func() AbilityHandler { return HandlerFuncs{} }
	if err := Register(game.AbilityScorch, ctor); err != nil {
		t.Fatalf("first registration: %v", err)
	}
	if err := Register(game.AbilityScorch, ctor); !errors.Is(err, ErrDuplicateRegistration) {
		t.Fatalf("expected ErrDuplicateRegistration, got %v", err)
	}
}

func TestRegisterNilFactory(t *testing.T) {
	resetRegistry(t)
	if err := Register(game.AbilityScorch, nil); !errors.Is(err, ErrNilFactory) {
		t.Fatalf("expected ErrNilFactory, got %v", err)
	}
}

func TestRegisterInvalidAbility(t *testing.T) {
	resetRegistry(t)
	if err := Register(game.AbilityNone, func() AbilityHandler { return HandlerFuncs{} }); !errors.Is(err, ErrInvalidAbility) {
		t.Fatalf("expected ErrInvalidAbility, got %v", err)
	}
}

func TestNewMissingAbility(t *testing.T) {
	resetRegistry(t)
	if _, err := New(game.AbilityScorch); !errors.Is(err, ErrUnknownAbility) {
		t.Fatalf("expected ErrUnknownAbility, got %v", err)
	}
}

func TestNewNilHandler(t *testing.T) {
	resetRegistry(t)
	if err := Register(game.AbilityScorch, func() AbilityHandler { return nil }); err != nil {
		t.Fatalf("register ability: %v", err)
	}
	if _, err := New(game.AbilityScorch); !errors.Is(err, ErrNilHandler) {
		t.Fatalf("expected ErrNilHandler, got %v", err)
	}
}

func TestRegisteredAbilitiesIsCopy(t *testing.T) {
	resetRegistry(t)
	if err := Register(game.AbilityScorch, func() AbilityHandler { return HandlerFuncs{} }); err != nil {
		t.Fatalf("register ability: %v", err)
	}

	ids := registeredAbilities()
	if len(ids) != 1 || ids[0] != game.AbilityScorch {
		t.Fatalf("unexpected ids: %v", ids)
	}

	// Mutating the returned slice should not affect the registry.
	ids[0] = game.AbilityDoOver

	registryMu.RLock()
	_, exists := registry[game.AbilityScorch]
	_, mutated := registry[game.AbilityDoOver]
	registryMu.RUnlock()

	if !exists {
		t.Fatalf("ability scorch missing after mutation")
	}
	if mutated {
		t.Fatalf("registry mutated after slice modification")
	}
}

// Ensure the handler functions compile against the expected runtime types.
func TestContextTypes(t *testing.T) {
	t.Helper()
	_ = StepBudgetContext{Engine: (*game.Engine)(nil), Piece: (*game.Piece)(nil), Move: (*game.MoveState)(nil)}
	_ = PhaseContext{Engine: (*game.Engine)(nil), Piece: (*game.Piece)(nil)}
	_ = MoveLifecycleContext{Engine: (*game.Engine)(nil), Move: (*game.MoveState)(nil), Request: game.MoveRequest{}, Segment: SegmentMetadata{}}
	_ = SegmentContext{Engine: (*game.Engine)(nil), Move: (*game.MoveState)(nil), Segment: SegmentMetadata{}}
	_ = CaptureContext{Engine: (*game.Engine)(nil), Move: (*game.MoveState)(nil), Attacker: (*game.Piece)(nil), Victim: (*game.Piece)(nil)}
	_ = TurnEndContext{Engine: (*game.Engine)(nil), Move: (*game.MoveState)(nil), Reason: TurnEndForced}
}
