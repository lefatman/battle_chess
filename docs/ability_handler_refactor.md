# MoveState Ability-Specific Runtime Inventory

The table below records each `MoveState` field in `internal/game/moves.go` that exists to support a
specific ability or rule. For each entry the owning mechanic and the scope of the state are noted so
that future refactors can relocate the data into ability-owned handlers.

| Field | Ability / Rule | Scope | Notes |
| --- | --- | --- | --- |
| `UsedPhasing` | Gale Lift / Umbral Step phasing | Per turn | Snapshot of whether the current mover can phase through blockers; derived from `canPhaseThrough` and carried for the whole move. 【F:chessTest/internal/game/moves.go†L12-L33】【F:chessTest/internal/game/piece_ops.go†L191-L233】
| `HasChainKill` | Chain Kill | Per turn | Cached presence of Chain Kill to loosen continuation legality and enable extra captures. 【F:chessTest/internal/game/moves.go†L18-L33】【F:chessTest/internal/game/moves.go†L56-L58】【F:chessTest/internal/game/moves.go†L234-L318】【F:chessTest/internal/game/moves.go†L1088-L1104】
| `HasQuantumKill` | Quantum Kill | Per turn | Cached ability check enabling post-capture removal logic. 【F:chessTest/internal/game/moves.go†L19-L33】【F:chessTest/internal/game/ability_resolver.go†L45-L63】
| `HasDoubleKill` | Double Kill | Per turn | Cached ability flag to drive extra removal on captures. 【F:chessTest/internal/game/moves.go†L20-L33】【F:chessTest/internal/game/ability_resolver.go†L21-L32】
| `HasResurrection` | Resurrection | Per turn | Cached ability presence to open a resurrection window after captures. 【F:chessTest/internal/game/moves.go†L21-L57】
| `ResurrectionWindow` | Resurrection | Per capture segment | Raised when a capture occurs and cleared on the next action or special move. 【F:chessTest/internal/game/moves.go†L22-L55】【F:chessTest/internal/game/moves.go†L249-L355】【F:chessTest/internal/game/moves.go†L392-L416】
| `FreeTurnsUsed` | Mist Shroud | Per turn | Counts the once-per-turn free direction change provided by Mist Shroud. 【F:chessTest/internal/game/moves.go†L24-L33】【F:chessTest/internal/game/moves.go†L609-L726】
| `BlazeRushUsed` | Blaze Rush | Per turn | Tracks whether the free dash has been consumed this move so further checks disable it. 【F:chessTest/internal/game/moves.go†L25-L33】【F:chessTest/internal/game/moves.go†L648-L770】
| `QuantumStepUsed` | Quantum Step | Per turn | Marks use of the once-per-turn blink/swap to suppress repeated activation and hints. 【F:chessTest/internal/game/moves.go†L26-L33】【F:chessTest/internal/game/moves.go†L370-L419】【F:chessTest/internal/game/moves.go†L880-L890】
| `SideStepUsed` | Side Step | Per turn | Marks use of the per-turn nudge action so it cannot repeat and so UI notes know when to show. 【F:chessTest/internal/game/moves.go†L27-L33】【F:chessTest/internal/game/moves.go†L325-L368】【F:chessTest/internal/game/moves.go†L880-L888】
| `TemporalLockUsed` | Temporal Lock | Per turn | Reserved for tracking the once-per-turn slowdown interaction of Temporal Lock (not yet consumed elsewhere). 【F:chessTest/internal/game/moves.go†L28-L33】
| `ChainKillCaptureCount` | Chain Kill | Per turn (capture counter) | Counts additional captures taken to enforce Chain Kill's two extra capture limit. 【F:chessTest/internal/game/moves.go†L29-L58】
| `FloodWakePushUsed` | Flood Wake | Per turn | Ensures the free orthogonal push only fires once each move. 【F:chessTest/internal/game/moves.go†L30-L33】【F:chessTest/internal/game/moves.go†L627-L737】
| `MaxCaptures` | Chain Kill | Per turn | Computed capture budget including Chain Kill's +2 allowance. 【F:chessTest/internal/game/moves.go†L31-L66】
| `LastSegmentCaptured` | Blaze Rush interaction | Per segment | Remembers whether the previous segment was a capture so Blaze Rush dashes are disallowed immediately afterward. 【F:chessTest/internal/game/moves.go†L32-L33】【F:chessTest/internal/game/moves.go†L648-L770】
| `QuantumKillUsed` | Quantum Kill | Per turn | Locks out the once-per-capture removal burst after it fires. 【F:chessTest/internal/game/moves.go†L33-L33】【F:chessTest/internal/game/ability_resolver.go†L45-L63】

## Function field usage audit

### `MoveState.canCaptureMore`
* Reads `Captures` to check how many pieces have been logged this turn before allowing further captures. 【F:chessTest/internal/game/moves.go†L38-L45】
* Reads `MaxCaptures` so the capture budget computed at move start can end the turn when exhausted. 【F:chessTest/internal/game/moves.go†L38-L45】

The method is called after every capture from both `startNewMove` and `continueMove`; in each case a `false` result ends the turn immediately, so callers depend on the capture tally staying in sync with the limit. 【F:chessTest/internal/game/moves.go†L174-L185】【F:chessTest/internal/game/moves.go†L301-L310】

### `MoveState.registerCapture`
* Appends the captured piece to `Captures`, which in turn feeds later `canCaptureMore` checks. 【F:chessTest/internal/game/moves.go†L48-L58】
* Reads `HasResurrection` to decide whether to raise `ResurrectionWindow` for the follow-up action window. 【F:chessTest/internal/game/moves.go†L48-L58】【F:chessTest/internal/game/moves.go†L249-L250】
* Reads `HasChainKill` and the mover's `Piece` color so it can increment `ChainKillCaptureCount` for hostile captures, enforcing the Chain Kill limit. 【F:chessTest/internal/game/moves.go†L48-L58】

Both `startNewMove` and `continueMove` invoke `registerCapture` immediately after executing a capture. The resulting state enables Resurrection follow-ups to be detected on the next segment and ensures the capture budget reflects the new removal before `canCaptureMore` decides whether the player may continue. 【F:chessTest/internal/game/moves.go†L174-L185】【F:chessTest/internal/game/moves.go†L249-L310】

## `startNewMove` ability checkpoints

The entrypoint for a turn front-loads several ability- and flag-aware decisions before executing the first segment. Each numbered block below corresponds to a contiguous section in `startNewMove` and highlights the ability or `MoveState` concern that would need a handler hook in a refactor.

1. **Capture blockers and notes.** `captureBlockedByBlockPath` inspects defender abilities and elemental overrides, and posts an ability note if the attempt is vetoed. 【F:chessTest/internal/game/moves.go†L129-L133】【F:chessTest/internal/game/ability_resolver.go†L203-L215】
2. **Step budget calculation.** `calculateStepBudget` aggregates per-ability step bonuses and Temporal Lock slow tokens before the move begins. The result seeds `RemainingSteps`. 【F:chessTest/internal/game/moves.go†L134-L145】【F:chessTest/internal/game/moves.go†L545-L589】
3. **Phasing capability snapshot.** `canPhaseThrough` collapses rider and team ability grants into a `UsedPhasing` flag that persists for path passability checks. 【F:chessTest/internal/game/moves.go†L134-L152】【F:chessTest/internal/game/piece_ops.go†L197-L231】
4. **Per-turn ability caches.** The `MoveState` initializer queries Chain/Quantum/Double Kill helpers and Resurrection to cache turn-wide flags and capture limits. 【F:chessTest/internal/game/moves.go†L147-L160】
5. **Post-segment toggles.** Immediately after executing the segment, `handlePostSegment` toggles Flood Wake/Blaze Rush booleans and logs Mist Shroud pivots, emitting ability notes as it goes. 【F:chessTest/internal/game/moves.go†L171-L199】【F:chessTest/internal/game/moves.go†L682-L727】
6. **Capture window bookkeeping.** On captures the engine raises the Resurrection window, tallies Chain Kill, and re-evaluates capture budgets via `registerCapture` and `canCaptureMore`. 【F:chessTest/internal/game/moves.go†L174-L186】【F:chessTest/internal/game/moves.go†L38-L58】
7. **Capture aftermath abilities.** `ResolveCaptureAbility` chains DoOver, Double Kill, Scorch, and Quantum Kill routines, including penalty drains that mutate `RemainingSteps`. 【F:chessTest/internal/game/moves.go†L174-L186】【F:chessTest/internal/game/ability_resolver.go†L9-L201】
8. **Forced turn endings.** `shouldEndTurnAfterCapture` scans attacker abilities for Poisonous Meat, Overload, and Bastion triggers, appending notes when they fire. 【F:chessTest/internal/game/moves.go†L188-L199】【F:chessTest/internal/game/moves.go†L858-L890】
9. **Post-move ability hints.** `resolveBlockPathFacing` and `checkPostMoveAbilities` post ability notes, while the final branch checks `RemainingSteps` alongside `hasFreeContinuation` to decide whether to end the turn. 【F:chessTest/internal/game/moves.go†L194-L209】【F:chessTest/internal/game/moves.go†L699-L769】【F:chessTest/internal/game/moves.go†L729-L818】

## Ability-driven helper behavior

### `calculateStepBudget`
Aggregates turn-start step counts by layering elemental + ability synergies and clearing any stored Temporal Lock slows. Each ability contribution is additive or subtractive: Scorch(+1), Tailwind(+2, -1 with Temporal Lock), Radiant Vision(+1, +1 more with Mist Shroud), Umbral Step(+2, -1 with Radiant Vision), and Schrodinger's Laugh(+2, +1 with Side Step). The routine also consumes the color's queued slow penalty. 【F:chessTest/internal/game/moves.go†L545-L589】

### `canPhaseThrough`
Determines whether the mover may pass through blockers by combining personal abilities with player-wide grants. Flood Wake or Bastion anywhere on the team disables phasing; Gale Lift or Umbral Step (piece or global) enables it. The result is cached as `UsedPhasing`. 【F:chessTest/internal/game/piece_ops.go†L197-L231】

### `handlePostSegment`
Updates per-turn flags after each executed segment. If the move qualified as a Flood Wake push or Blaze Rush dash, it marks the usage and emits corresponding notes. It also calls `logDirectionChange`, which either consumes Mist Shroud's free pivot or reports the standard extra step cost. 【F:chessTest/internal/game/moves.go†L682-L727】

### `ResolveCaptureAbility`
On every capture the resolver:
* Runs DoOver rewinds before any further processing.
* Attempts Double Kill extra removals, falling back to Fire-Scorch splash if Double Kill failed.
* Spends the cached Quantum Kill charge to remove another victim and potentially echo a second removal.
* Applies capture penalties such as Poisonous Meat and Overload + Stalwart drains, appending ability notes whenever a branch fires.
Each step may mutate `RemainingSteps` or remove extra pieces, so the hook must run before `canCaptureMore` reevaluates continuation rights. 【F:chessTest/internal/game/ability_resolver.go†L9-L201】

## Hook surface summary

To migrate these mechanics into ability-owned handlers, the refactor will need dedicated extension points:

* **Budget contributions:** `OnComputeStepBudget(piece, ctx)` allowing abilities to add/remove step modifiers and consume stored slows before `RemainingSteps` is set. 【F:chessTest/internal/game/moves.go†L134-L145】【F:chessTest/internal/game/moves.go†L545-L589】
* **Phasing eligibility:** `OnQueryPathingFlags(piece, segment)` letting abilities advertise or veto pass-through rights before path validation caches `UsedPhasing`. 【F:chessTest/internal/game/moves.go†L134-L152】【F:chessTest/internal/game/piece_ops.go†L197-L231】
* **Capture aftermath:** `OnCaptureResolved(attacker, victim, ctx)` combining extra removals, resurrection windows, capture counts, and penalty drains prior to continuation checks. 【F:chessTest/internal/game/moves.go†L174-L199】【F:chessTest/internal/game/ability_resolver.go†L9-L201】
* **Post-segment toggles:** `OnSegmentResolved(piece, segment, ctx)` updating once-per-turn flags (Flood Wake, Blaze Rush, Mist Shroud) and announcing ability hints for subsequent actions. 【F:chessTest/internal/game/moves.go†L171-L209】【F:chessTest/internal/game/moves.go†L682-L890】

