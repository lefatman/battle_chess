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

## Capture ability resolution

### `ResolveCaptureAbility` branch walk-through
The capture resolver evaluates ability-specific follow-ups in a strict order so each effect can short-circuit later ones when it succeeds:
- **Do-Over interrupt.** Immediately after a victim is identified the engine calls `maybeTriggerDoOver`; a triggered rewind returns `ErrDoOverActivated`, aborting the rest of the capture handling. 【F:chessTest/internal/game/ability_resolver.go†L10-L33】【F:chessTest/internal/game/ability_resolver.go†L102-L123】
- **Double Kill sweep.** If the attacker has Double Kill the resolver looks for the best adjacent, lower-ranked enemy around the capture square and attempts to remove it. On success it records an ability note and suppresses later Scorch processing. 【F:chessTest/internal/game/ability_resolver.go†L21-L33】【F:chessTest/internal/game/ability_resolver.go†L69-L100】
- **Fire Scorch fallback.** When no Double Kill extra removal happened, Fire-aligned attackers with Scorch reuse the same adjacency scan to burn an additional enemy and emit the Fire Scorch note. 【F:chessTest/internal/game/ability_resolver.go†L34-L43】【F:chessTest/internal/game/ability_resolver.go†L69-L100】
- **Quantum Kill chain.** Once-per-turn Quantum Kill consumes its flag, removes the best-ranked enemy at or below the primary victim’s rank anywhere on the board, and appends a note. It then runs a secondary adjacency search from the remote square to support the echo removal and note when possible. 【F:chessTest/internal/game/ability_resolver.go†L45-L63】【F:chessTest/internal/game/ability_resolver.go†L144-L180】
- **Capture penalties.** After all extra removals (or immediately if none fire) the resolver delegates to `applyCapturePenalties` so lingering effects can drain steps before post-capture continuation checks run. 【F:chessTest/internal/game/ability_resolver.go†L64-L66】【F:chessTest/internal/game/ability_resolver.go†L182-L201】

Every removal path calls `attemptAbilityRemoval`, which respects King immunity, Stalwart/Belligerent rank gating, and can recursively trigger additional Do-Over interrupts on secondary victims. 【F:chessTest/internal/game/ability_resolver.go†L125-L155】

### Lingering capture penalties
`applyCapturePenalties` enforces two drain mechanics that must run even if the attacker keeps moving:
- **Poisonous Meat** subtracts one remaining step unless the attacker is Shadow-aligned, logging the drain as an ability note. 【F:chessTest/internal/game/ability_resolver.go†L187-L193】
- **Overload + Stalwart** together on a Lightning attacker cost one step and log their own note, reflecting the compounded strain. 【F:chessTest/internal/game/ability_resolver.go†L195-L200】

These drains rely on the active `MoveState` (`currentMove`) still being present so the handler needs access to step counts and the shared note buffer when invoked.

### Capture hook responsibilities
To migrate capture processing into discrete handlers, two hook points cover the existing behaviors:
- **`OnCapture(attacker, victim, square, moveCtx)`** – invoked immediately after the primary removal so it can raise Resurrection windows, bump Chain Kill counters via `registerCapture`, and then orchestrate ordered ability resolution (Do-Over → Double Kill/Scorch → Quantum Kill/Echo). The hook needs:
  - the attacking `Piece` (may be `nil` for ability-only removals),
  - the captured `Piece` (for Do-Over, rank/color comparisons, and undo bookkeeping),
  - the capture `Square` (seed for adjacency scans), and
  - the active `MoveState`/engine context (to append notes, update Quantum Kill usage, and recurse into `attemptAbilityRemoval`). 【F:chessTest/internal/game/moves.go†L174-L186】【F:chessTest/internal/game/ability_resolver.go†L10-L201】【F:chessTest/internal/game/moves.go†L48-L58】
- **`OnCaptureAftermath(attacker, moveCtx)`** – runs once the capture resolver finishes to apply lingering step drains and evaluate forced turn endings. It requires the attacker and current move so it can invoke penalty drains, consult `shouldEndTurnAfterCapture`, and end the turn when necessary. 【F:chessTest/internal/game/ability_resolver.go†L182-L201】【F:chessTest/internal/game/moves.go†L188-L207】【F:chessTest/internal/game/moves.go†L858-L878】

This division keeps interrupt-style abilities (Do-Over) and cascading removals in the first hook while concentrating exhaustion and termination checks in the aftermath stage that immediately precedes promotion and post-move notifications.
