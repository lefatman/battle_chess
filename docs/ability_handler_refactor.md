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

## `continueMove` ability checkpoints

1. **Side Step bail-out.** The continuation handler first offers the Side Step nudge before any other validation. `trySideStepNudge` consumes one step, marks `SideStepUsed`, resets the resurrection window, and re-runs the turn-ending checks, so the caller must branch on its return value. 【F:chessTest/internal/game/moves.go†L231-L233】【F:chessTest/internal/game/moves.go†L325-L368】
2. **Quantum Step bail-out.** If Side Step declined, `tryQuantumStep` performs the once-per-turn blink or swap, decrementing `RemainingSteps`, toggling `QuantumStepUsed`, clearing the resurrection window, and then invoking the same post-segment termination checks. 【F:chessTest/internal/game/moves.go†L235-L237】【F:chessTest/internal/game/moves.go†L370-L419】
3. **Resurrection window reset.** Any normal continuation immediately clears `ResurrectionWindow` before evaluating the destination, limiting Resurrection follow-ups to the very next input. 【F:chessTest/internal/game/moves.go†L249-L251】
4. **Block Path veto.** Defender Block Path is consulted before execution; a match appends an ability note and aborts the capture attempt. Water-aligned attackers bypass the veto. 【F:chessTest/internal/game/moves.go†L257-L259】【F:chessTest/internal/game/ability_resolver.go†L203-L214】
5. **Post-segment toggles.** After executing the segment, `handlePostSegment` updates `LastSegmentCaptured`, consumes Flood Wake and Blaze Rush freebies, and logs Mist Shroud pivots. These updates must occur before later continuation logic queries the flags. 【F:chessTest/internal/game/moves.go†L297-L300】【F:chessTest/internal/game/moves.go†L682-L727】
6. **Capture aftermath.** Captures enqueue resurrection and Chain Kill bookkeeping via `registerCapture`, then invoke `ResolveCaptureAbility` for chained removals and penalties before `canCaptureMore` decides whether the move may continue. 【F:chessTest/internal/game/moves.go†L301-L310】
7. **Forced turn endings.** After every segment the engine first asks `shouldEndTurnAfterCapture` (via `checkPostCaptureTermination`) and then checks for step exhaustion, only allowing the turn to continue if a Blaze Rush or Flood Wake free continuation is available. 【F:chessTest/internal/game/moves.go†L313-L318】【F:chessTest/internal/game/moves.go†L729-L739】【F:chessTest/internal/game/moves.go†L858-L890】

### Helper side effects referenced by `continueMove`

* `trySideStepNudge` writes undo history, spends one step, sets `SideStepUsed`, clears the resurrection window, and funnels into `handlePostSegment` so Flood Wake, Blaze Rush, and Mist Shroud hooks fire before the post-action termination logic. 【F:chessTest/internal/game/moves.go†L340-L365】
* `tryQuantumStep` mirrors the Side Step flow while also toggling `QuantumStepUsed`, clamping `RemainingSteps` at zero, and either blinking or swapping pieces before the shared turn-ending checks. 【F:chessTest/internal/game/moves.go†L383-L417】
* `captureBlockedByBlockPath` enforces the defender's facing restriction unless the attacker is Water-aspected, emitting the explanatory note that `continueMove` surfaces to the player. 【F:chessTest/internal/game/ability_resolver.go†L203-L214】
* `hasFreeContinuation` defers to Blaze Rush and Flood Wake option checks; these helpers inspect `BlazeRushUsed`, `LastSegmentCaptured`, `FloodWakePushUsed`, elemental alignment, and future path availability to decide whether step exhaustion should end the turn. 【F:chessTest/internal/game/moves.go†L729-L815】

### Special move flow audit

#### Side Step nudge (`trySideStepNudge`)
* **Eligibility gate:** Requires an active move, a Side Step-enabled piece, unused `SideStepUsed`, at least one remaining step, adjacency between `from`/`to`, and an empty destination. 【F:chessTest/internal/game/moves.go†L325-L338】
* **State mutations:** Writes undo history for both squares, decrements `RemainingSteps`, toggles `SideStepUsed`, appends the destination to the path, clears the Resurrection window, and routes through `handlePostSegment` so downstream toggles fire. 【F:chessTest/internal/game/moves.go†L340-L355】
* **Notes & termination:** Emits "Side Step nudge" with cost, then re-runs post-capture termination, step exhaustion checks, and remaining-step hints. 【F:chessTest/internal/game/moves.go†L357-L365】

#### Quantum Step (`tryQuantumStep` & helpers)
* **Activation checks:** Demands an active move, Quantum Step ability, unused `QuantumStepUsed`, positive steps, and a successful `validateQuantumStep` result. 【F:chessTest/internal/game/moves.go†L370-L381】
* **Validation branches:** `validateQuantumStep` insists on adjacency, forbids hostile occupants, rejects standard-legal non-captures (so the special move only fires when normal movement is blocked), and returns a friendly piece pointer to signal swap mode. 【F:chessTest/internal/game/moves.go†L422-L444】
* **State mutations:** Records undo squares, decrements steps (clamping at zero), marks `QuantumStepUsed`, clears the Resurrection window, appends the hop to the path, and funnels into `handlePostSegment`. 【F:chessTest/internal/game/moves.go†L383-L405】
* **Blink vs swap:** Empty targets execute a standard `executeMoveSegment` blink. Friendly targets call `performQuantumSwap`, which zeroes En Passant, swaps board occupants, updates piece bitboards/occupancy, and refreshes castling rights for both pieces. 【F:chessTest/internal/game/moves.go†L395-L483】
* **Notes & termination:** Emits either "blink" or "swap" notes, reuses the post-capture termination flow, and rechecks step exhaustion with free-continuation overrides. 【F:chessTest/internal/game/moves.go†L397-L417】

### Special move hook surface

To migrate Side Step and Quantum Step into handlers, the refactor will need dedicated hook methods in addition to the segment lifecycle callbacks:

* **`OnSpecialMoveStart(piece, request, ctx)`** – decides whether the handler will consume the input before standard continuation validation. Must gate on adjacency, availability flags, open destinations, or friendly swap targets, and is responsible for declining when normal movement should proceed. 【F:chessTest/internal/game/moves.go†L325-L381】
* **`OnSpecialMoveResolve(piece, ctx)`** – spends steps, toggles per-turn flags (`SideStepUsed`, `QuantumStepUsed`), clears `ResurrectionWindow`, performs the blink/swap board mutations, and appends ability notes. It should also populate undo history and extend the current path before control returns to shared post-segment routines. 【F:chessTest/internal/game/moves.go†L340-L417】【F:chessTest/internal/game/moves.go†L446-L483】
* **`OnSegmentStart(piece, segment, ctx)`** – invoked after a special move schedules a new segment so handlers can ensure `handlePostSegment` equivalents toggle Flood Wake, Blaze Rush, Mist Shroud, and other once-per-turn states in sync with the normal pipeline. 【F:chessTest/internal/game/moves.go†L348-L365】【F:chessTest/internal/game/moves.go†L682-L727】
* **`OnSegmentResolved(piece, segment, ctx)`** – continues to cover post-action cleanup (already required for Flood Wake/Blaze Rush/Mist Shroud) so special moves and standard segments share a termination path and ability note reporting. 【F:chessTest/internal/game/moves.go†L351-L365】【F:chessTest/internal/game/moves.go†L682-L739】

Handlers that implement these hooks must directly manage the `MoveState` toggles (`SideStepUsed`, `QuantumStepUsed`, `RemainingSteps`, `ResurrectionWindow`) and emit the corresponding ability hints so the engine's turn-ending logic sees consistent state regardless of whether the action came from a special move or a standard segment. 【F:chessTest/internal/game/moves.go†L325-L417】

### Sequencing requirements for handler hooks

* Special actions (Side Step, Quantum Step) must resolve and run their post-segment cleanup before any standard continuation logic executes, because they mutate step totals, resurrection windows, and ability usage flags that later checks read. 【F:chessTest/internal/game/moves.go†L231-L318】【F:chessTest/internal/game/moves.go†L325-L419】
* Normal continuations clear the resurrection window before looking for a defender, so any Resurrection handler has only the immediately following request to act. 【F:chessTest/internal/game/moves.go†L249-L251】
* Capture handling requires the order `registerCapture` → `ResolveCaptureAbility` → `canCaptureMore`; reordering would break resurrection detection, chained removals, or capture-limit enforcement. 【F:chessTest/internal/game/moves.go†L301-L310】
* Post-segment ability toggles run before capture aftermath and turn-ending checks so that free actions (Flood Wake/Blaze Rush) and Mist Shroud pivots are visible to both the capture budget logic and the final continuation gate. 【F:chessTest/internal/game/moves.go†L297-L318】【F:chessTest/internal/game/moves.go†L682-L739】

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

