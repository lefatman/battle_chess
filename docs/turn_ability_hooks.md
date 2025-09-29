# Turn Ability Hook Review

## Ability-sensitive work performed by `endTurn`
- `resolvePromotion` finalizes pending pawn promotions before the turn flips, ensuring the upgraded piece type and note are applied immediately after the moving piece finishes. 【F:chessTest/internal/game/moves.go†L496-L521】【F:chessTest/internal/game/engine.go†L322-L339】
- `applyTemporalLockSlow` checks for the Temporal Lock ability and, if present, applies a slow penalty (doubled for Fire element pieces) to the opponent and records the note before control passes. 【F:chessTest/internal/game/moves.go†L496-L521】【F:chessTest/internal/game/moves.go†L523-L537】
- The resulting turn summary note is appended after the ability hooks run so that ability notes (promotion, Temporal Lock, etc.) appear before the generic status message in the final log. 【F:chessTest/internal/game/moves.go†L508-L520】

## Ability-specific turn ending or notifications
- `applyTemporalLockSlow` stores a slow penalty for the opposing color when the acting piece has Temporal Lock, with Fire-aligned pieces inflicting a value of two; it also appends an ability note announcing the slow. 【F:chessTest/internal/game/moves.go†L523-L537】
- `shouldEndTurnAfterCapture` forces the turn to end (and appends a note) when the capturing piece has any of:
  - Poisonous Meat (always ends the turn). 【F:chessTest/internal/game/moves.go†L858-L890】
  - Overload while aligned to Lightning. 【F:chessTest/internal/game/moves.go†L858-L890】
  - Bastion while aligned to Earth. 【F:chessTest/internal/game/moves.go†L858-L890】
- `checkPostMoveAbilities` emits reminder notes when Side Step or Quantum Step are still available and the piece retains steps this turn. 【F:chessTest/internal/game/moves.go†L892-L902】

## Required handler hooks and ordering
To replicate the current behavior inside an ability handler system, the engine would need the following hooks executed in this order after each segment/turn:
1. **Post-capture enforcement (`ShouldForceTurnEnd`)** – queried immediately after processing capture abilities but before other post-move state so that forced turn endings (Poisonous Meat, Overload, Bastion) can short-circuit the remainder of the segment. 【F:chessTest/internal/game/moves.go†L181-L204】
2. **Post-move notifications (`PostMoveNotes`)** – triggered after standard post-move state like promotions and facing adjustments so ability reminders (Side Step, Quantum Step) can append to the note log without preventing normal cleanup. 【F:chessTest/internal/game/moves.go†L204-L218】【F:chessTest/internal/game/moves.go†L892-L902】
3. **Turn finalization (`OnTurnEnd`)** – fired once per completed turn; it must first resolve promotions, then apply Temporal Lock slow effects, and only afterwards flip the turn and update the game status to preserve the correct actor/opponent context for ability effects and notes. 【F:chessTest/internal/game/moves.go†L496-L520】
