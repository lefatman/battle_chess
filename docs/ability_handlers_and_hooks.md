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
Determines whether the mover may pass through blockers by combining personal abilities with player-wide grants. The routine inspects the following toggles before caching the outcome as `UsedPhasing`:

* **Flood Wake (piece or side ability)** – Presence on the acting piece (`pc.Abilities`) or the mover's color entry in `e.abilities[color]` sets `hasFlood`, which immediately vetoes phasing. 【F:chessTest/internal/game/piece_ops.go†L205-L219】
* **Bastion (piece or side ability)** – Mirrors Flood Wake's veto, disabling phasing for the entire move whenever found either on the piece or the color ability list. 【F:chessTest/internal/game/piece_ops.go†L205-L219】
* **Gale Lift (piece or side ability)** – Grants phasing when present anywhere on the mover or the side list, provided neither Flood Wake nor Bastion blocked it first. 【F:chessTest/internal/game/piece_ops.go†L205-L224】
* **Umbral Step (piece or side ability)** – Offers a fallback grant: if Gale Lift was absent but no vetoes fired, the piece or side owning Umbral Step still turns phasing on for the move. 【F:chessTest/internal/game/piece_ops.go†L224-L231】

The checks give side-level abilities (`e.abilities[color].Contains`) equal priority with piece abilities, meaning a single team-wide veto (Flood Wake/Bastion) suppresses every mover, while a single team-wide grant (Gale Lift/Umbral Step) can elevate otherwise mundane pieces into phasing riders. 【F:chessTest/internal/game/piece_ops.go†L205-L231】

#### Handler responsibilities for phasing

In the handler model, phasing logic should surface through an `OnQueryPathingFlags` hook that runs **once per move** before path validation:

* **Precompute grants and vetoes.** The handler must aggregate both piece-owned and side-owned sources into a single flag so per-segment path checks can consult a cached `UsedPhasing` value instead of re-evaluating abilities for every square traversed. 【F:chessTest/internal/game/moves.go†L134-L152】【F:chessTest/internal/game/piece_ops.go†L197-L231】
* **Respect priority ordering.** Veto abilities (Flood Wake, Bastion) must override grants, while Gale Lift and Umbral Step may independently supply the pass-through permission. 【F:chessTest/internal/game/piece_ops.go†L205-L231】
* **Mirror side-ability reach.** Side-level toggles need to be queried alongside the piece so team-wide buffs or bans continue to propagate to every mover in the turn. 【F:chessTest/internal/game/piece_ops.go†L205-L231】

By collapsing the decision into the move-start snapshot, path validation can continue to rely on the inexpensive `UsedPhasing` check while allowing ability-owned handlers to determine the value. 【F:chessTest/internal/game/moves.go†L134-L152】

### `handlePostSegment`

After every executed segment the engine funnels through `handlePostSegment`, which performs four distinct side effects before any capture aftermath or continuation logic runs:

1. **Capture memory.** Writes `LastSegmentCaptured` so the next Blaze Rush option check knows whether a capture just occurred. 【F:chessTest/internal/game/moves.go†L682-L689】
2. **Flood Wake bookkeeping.** When `isFloodWakePushAvailable` confirms the move was an orthogonal, non-capturing shove by a Water-aspected Flood Wake piece, the handler toggles `FloodWakePushUsed` and emits "Flood Wake push (free)". 【F:chessTest/internal/game/moves.go†L689-L692】
3. **Blaze Rush bookkeeping.** When `isBlazeRushDash` reports the segment as a valid free dash, it marks `BlazeRushUsed` and posts "Blaze Rush dash (free)". 【F:chessTest/internal/game/moves.go†L694-L697】
4. **Direction logging.** Delegates to `logDirectionChange` so Mist Shroud can consume its free pivot or so the engine can note the extra step cost. 【F:chessTest/internal/game/moves.go†L699-L700】

Any refactor that moves these abilities into handlers must preserve the ordering so later checks observe the updated flags and notes before deciding whether the turn can continue.

#### `logDirectionChange`

This helper only runs for sliding pieces with at least two prior path entries. It compares the previous and current move directions and, when a pivot occurs, offers Mist Shroud the once-per-turn freebie by incrementing `FreeTurnsUsed` and appending "Mist Shroud free pivot". If the free pivot is unavailable, it instead appends "Direction change cost +1 step" to surface the penalty. 【F:chessTest/internal/game/moves.go†L702-L727】

#### Free continuation gate helpers

`hasFreeContinuation` short-circuits the step-exhaustion turn ending check by asking two ability-specific predicates in order:

* `hasBlazeRushOption` – Requires an unused Blaze Rush on the active move (`BlazeRushUsed == false`), no capture on the last segment (`LastSegmentCaptured == false`), a sliding mover, and at least one prior path entry to derive the direction vector. It then peeks one square further along that direction and returns true only if the destination is empty. 【F:chessTest/internal/game/moves.go†L742-L769】
* `hasFloodWakePushOption` – Requires an unused Flood Wake push (`FloodWakePushUsed == false`), Water alignment, and an empty orthogonal adjacent square around the piece’s current location. 【F:chessTest/internal/game/moves.go†L789-L814】

These checks read the flags written in `handlePostSegment`, tying the free-move availability directly to the bookkeeping above.

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

### Per-ability handler responsibilities

Breaking `handlePostSegment` into discrete ability handlers implies the following dedicated duties:

* **Flood Wake handler** – Detect whether the latest segment was an orthogonal, non-capturing shove, then spend the move’s Flood Wake push token (`FloodWakePushUsed`) and announce the free action. Its option check must mirror `hasFloodWakePushOption` so free-continuation logic stays consistent. 【F:chessTest/internal/game/moves.go†L689-L704】【F:chessTest/internal/game/moves.go†L789-L814】
* **Blaze Rush handler** – Track when a segment qualifies as the free dash (`BlazeRushUsed`) and surface the matching note. Its availability check must respect `LastSegmentCaptured`, direction continuity, path history, and board emptiness just like `hasBlazeRushOption`/`isBlazeRushDash`. 【F:chessTest/internal/game/moves.go†L694-L705】【F:chessTest/internal/game/moves.go†L742-L787】
* **Mist Shroud handler** – Monitor direction pivots for sliders, consume the once-per-turn `FreeTurnsUsed` credit, and emit either the free-pivot or penalty note. 【F:chessTest/internal/game/moves.go†L702-L727】

Centralising these behaviors inside ability-owned hooks ensures the core engine no longer needs to mutate `MoveState` fields directly for Flood Wake, Blaze Rush, or Mist Shroud while still providing `hasFreeContinuation` the same data it reads today.

## Step budget ability audit

### Ability + element modifiers

`calculateStepBudget` grants or removes steps only when specific ability and element pairings are present on the acting piece. The current combinations are:

| Ability | Required element | Net change | Notes |
| --- | --- | --- | --- |
| Scorch | Fire | +1 | Flat bonus with no additional interactions. 【F:chessTest/internal/game/moves.go†L552-L554】 |
| Tailwind | Air | +2 | Applies before Temporal Lock’s slowdown check. 【F:chessTest/internal/game/moves.go†L555-L560】 |
| Tailwind + Temporal Lock | Air | −1 (in addition to Tailwind’s +2) | Temporal Lock partially suppresses Tailwind, resulting in a net +1. 【F:chessTest/internal/game/moves.go†L555-L559】 |
| Radiant Vision | Light | +1 | Grants a light-aligned bonus and enables the Mist Shroud combo. 【F:chessTest/internal/game/moves.go†L561-L565】 |
| Radiant Vision + Mist Shroud | Light | +1 (stacking with Radiant Vision) | Adds a further bonus for the paired abilities, yielding +2 total. 【F:chessTest/internal/game/moves.go†L561-L565】 |
| Umbral Step | Shadow | +2 | Shadow-aligned step surge that is later dampened by Radiant Vision. 【F:chessTest/internal/game/moves.go†L567-L571】 |
| Umbral Step + Radiant Vision | Shadow | −1 (in addition to Umbral Step’s +2) | Cross-polarity pairing reduces the umbral bonus, leaving a net +1. 【F:chessTest/internal/game/moves.go†L567-L571】 |
| Schrödinger’s Laugh | Any | +2 | Universal step bonus available without an element check. 【F:chessTest/internal/game/moves.go†L573-L575】 |
| Schrödinger’s Laugh + Side Step | Any | +1 (stacking with Laugh) | Extra momentum when both abilities are present for +3 total. 【F:chessTest/internal/game/moves.go†L573-L577】 |

All bonuses are summed with the one-step baseline, then the color’s stored Temporal Lock slow penalty is subtracted and cleared. The minimum budget is clamped to one step. 【F:chessTest/internal/game/moves.go†L547-L588】

### Stacking vs. overrides in the handler model

* **Tailwind × Temporal Lock** – Temporal Lock should continue to *stack* as an additive −1 modifier that applies only when Tailwind triggered. This preserves the existing net +1 and allows future slow sources to combine coherently.
* **Radiant Vision × Mist Shroud** – The +1 combo should stack with Radiant Vision’s base bonus rather than overriding it so Mist Shroud can contribute meaningfully when both abilities are owned.
* **Umbral Step × Radiant Vision** – Radiant Vision should remain an additional −1 modifier against Umbral Step’s +2 so the net benefit reflects the present day polarity tension.
* **Schrödinger’s Laugh × Side Step** – Side Step should continue to provide an extra +1 on top of the universal +2 instead of replacing it, keeping the playful +3 spike intact.

Future handlers should express the relationships above through additive contributions rather than mutually exclusive overrides so additional abilities can compose cleanly.

### Handler step contribution data model

To mirror today’s arithmetic, the step budget hook should expose a minimal accumulator structure:

```go
type StepBudget struct {
    Base int // The guaranteed starting point (defaults to 1).
    Bonus int // Sum of additive modifiers from abilities/elements.
    GlobalPenalty int // Aggregate of stored slow tokens or other cross-piece taxes.
}
```

Handlers would receive a mutable `StepBudget` and adjust `Bonus` (or `GlobalPenalty` for effects like Temporal Lock) while optionally editing `Base` if a future mechanic changes the minimum. After all handlers run, the engine computes `max(1, Base+Bonus-GlobalPenalty)` to obtain the final budget, matching the current clamp semantics. 【F:chessTest/internal/game/moves.go†L547-L588】

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
