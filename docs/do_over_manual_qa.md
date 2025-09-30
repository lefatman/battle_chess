<!-- path: docs/do_over_manual_qa.md -->
# Do-Over Resync Smoke Checklist

- Launch the battle chess web UI and start a new match.
- Configure both sides with the Do-Over ability and any elements so the match locks in.
- Play two standard opening moves (e.g., White pawn e2→e4, Black pawn d7→d5).
- As White, capture the d5 pawn (e4→d5) to trigger the Do-Over ability.
- Verify the board rewinds to the pre-move state, the turn indicator resets to White, and a toast/log entry displays the Do-Over warning message.
- Submit another legal move and confirm the board remains in sync with the server state.
