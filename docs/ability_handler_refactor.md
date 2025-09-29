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

