// path: chessTest/internal/game/ability_resolver_test.go
package game

import "testing"

func newEmptyBoard() boardSoA {
	var b boardSoA
	for i := range b.alive {
		b.alive[i] = false
	}
	return b
}

func addPiece(b *boardSoA, idx int, id int, color Color, typ PieceType, sq Square) {
	b.ids[idx] = id
	b.alive[idx] = true
	b.colors[idx] = color
	b.types[idx] = typ
	b.squares[idx] = sq
	bit := uint64(1) << uint(sq)
	b.occupancy[color.Index()] |= bit
	b.pieceMask[color.Index()][typ] |= bit
}

func TestAbilityResolverPhases(t *testing.T) {
	cases := []struct {
		name   string
		side   AbilitySet
		enemy  AbilitySet
		setup  func(*boardSoA, *resolveContext)
		expect func(*testing.T, resolveResult, error, *boardSoA, *resolveContext)
	}{
		{
			name: "scorch firewall",
			side: NewAbilitySet(AbilityScorch),
			expect: func(t *testing.T, res resolveResult, err error, _ *boardSoA, _ *resolveContext) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res.telemetry.firewallCount != 4 {
					t.Fatalf("expected 4 firewalls, got %d", res.telemetry.firewallCount)
				}
				want := map[Square]struct{}{SquareF4: {}, SquareD4: {}, SquareE5: {}, SquareE3: {}}
				for i := uint8(0); i < res.telemetry.firewallCount; i++ {
					sq := res.telemetry.firewallSquares[i]
					if _, ok := want[sq]; !ok {
						t.Fatalf("unexpected firewall square %v", sq)
					}
				}
			},
		},
		{
			name: "blaze rush quick knight",
			side: NewAbilitySet(AbilityBlazeRush),
			expect: func(t *testing.T, res resolveResult, err error, _ *boardSoA, _ *resolveContext) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res.telemetry.blazeQK != 1 {
					t.Fatalf("expected quick knight bonus, got %d", res.telemetry.blazeQK)
				}
			},
		},
		{
			name: "blaze rush dark queen",
			side: NewAbilitySet(AbilityBlazeRush),
			setup: func(b *boardSoA, _ *resolveContext) {
				b.types[0] = Queen
			},
			expect: func(t *testing.T, res resolveResult, err error, _ *boardSoA, _ *resolveContext) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res.telemetry.blazeDK != 1 {
					t.Fatalf("expected dark knight bonus, got %d", res.telemetry.blazeDK)
				}
			},
		},
		{
			name:  "flood wake do-over",
			side:  NewAbilitySet(AbilityFloodWake),
			enemy: NewAbilitySet(AbilityDoOver),
			setup: func(b *boardSoA, ctx *resolveContext) {
				addPiece(b, 1, 2, Black, Pawn, ctx.target)
				ctx.captureIdx = 1
				b.ability[1] = ctx.enemyMask
				b.removePiece(1)
			},
			expect: func(t *testing.T, res resolveResult, err error, _ *boardSoA, ctx *resolveContext) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !res.doOver {
					t.Fatalf("expected do-over trigger")
				}
				if !res.telemetry.floodWakePersistent {
					t.Fatalf("expected flood wake persistence")
				}
				if !ctx.doOverUsed[Black.Index()] {
					t.Fatalf("expected do-over usage recorded")
				}
			},
		},
		{
			name: "mist tailwind synergy",
			side: NewAbilitySet(AbilityMistShroud, AbilityTailwind),
			expect: func(t *testing.T, res resolveResult, err error, _ *boardSoA, _ *resolveContext) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res.telemetry.mistShroudQS == 0 {
					t.Fatalf("expected mist shroud grant")
				}
				if !res.telemetry.tailwindBridge {
					t.Fatalf("expected tailwind synergy")
				}
			},
		},
		{
			name: "bastion sturdy gale",
			side: NewAbilitySet(AbilityBastion, AbilitySturdy, AbilityGaleLift),
			expect: func(t *testing.T, res resolveResult, err error, _ *boardSoA, _ *resolveContext) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !res.telemetry.bastion || !res.telemetry.sturdy || !res.telemetry.gale {
					t.Fatalf("expected bastion/sturdy/gale interaction")
				}
			},
		},
		{
			name: "scatter shot captures",
			side: NewAbilitySet(AbilityScatterShot),
			setup: func(b *boardSoA, ctx *resolveContext) {
				addPiece(b, 1, 3, Black, Pawn, SquareF4)
				addPiece(b, 2, 4, Black, Pawn, SquareE5)
			},
			expect: func(t *testing.T, res resolveResult, err error, b *boardSoA, _ *resolveContext) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res.telemetry.scatterHits != 2 {
					t.Fatalf("expected 2 scatter captures, got %d", res.telemetry.scatterHits)
				}
				if b.alive[1] || b.alive[2] {
					t.Fatalf("expected adjacent enemies removed")
				}
			},
		},
		{
			name: "overload assignments",
			side: NewAbilitySet(AbilityOverload),
			setup: func(b *boardSoA, _ *resolveContext) {
				addPiece(b, 1, 5, White, Pawn, SquareC3)
				b.ability[0] = NewAbilitySet(AbilityTailwind)
				b.ability[1] = NewAbilitySet(AbilityScorch)
			},
			expect: func(t *testing.T, res resolveResult, err error, _ *boardSoA, _ *resolveContext) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res.telemetry.overloadCount != 2 {
					t.Fatalf("expected overload count 2, got %d", res.telemetry.overloadCount)
				}
				if res.telemetry.overload[0] != AbilityTailwind || res.telemetry.overload[1] != AbilityScorch {
					t.Fatalf("unexpected overload assignments %v %v", res.telemetry.overload[0], res.telemetry.overload[1])
				}
			},
		},
		{
			name: "raijin follow",
			side: NewAbilitySet(AbilityRaijin),
			expect: func(t *testing.T, res resolveResult, err error, _ *boardSoA, _ *resolveContext) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !res.telemetry.raijinFollow {
					t.Fatalf("expected raijin follow-up")
				}
			},
		},
		{
			name: "radiant vision",
			side: NewAbilitySet(AbilityRadiantVision),
			expect: func(t *testing.T, res resolveResult, err error, _ *boardSoA, _ *resolveContext) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !res.telemetry.radiantVision {
					t.Fatalf("expected radiant vision active")
				}
			},
		},
		{
			name:  "blinding tempo",
			enemy: NewAbilitySet(AbilityBlinding),
			expect: func(t *testing.T, res resolveResult, err error, _ *boardSoA, _ *resolveContext) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !res.telemetry.blindingSkipped {
					t.Fatalf("expected blinding tempo")
				}
			},
		},
		{
			name:  "light speed mover priority",
			side:  NewAbilitySet(AbilityLightSpeed, AbilityBlazeRush),
			enemy: NewAbilitySet(AbilityBlazeRush),
			expect: func(t *testing.T, res resolveResult, err error, _ *boardSoA, ctx *resolveContext) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				log := res.telemetry.phaseLogs[int(phaseOffense)]
				if log.count < 2 {
					t.Fatalf("expected both sides in offense phase, got %d", log.count)
				}
				if log.owners[0] != ctx.board.colors[ctx.mover] {
					t.Fatalf("expected mover to act first")
				}
				if res.telemetry.ck != King || res.telemetry.dk != Knight {
					t.Fatalf("expected default CK/QK/DK ordering")
				}
			},
		},
		{
			name:  "light speed enemy priority",
			side:  NewAbilitySet(AbilityBlazeRush),
			enemy: NewAbilitySet(AbilityLightSpeed, AbilityBlazeRush),
			expect: func(t *testing.T, res resolveResult, err error, _ *boardSoA, ctx *resolveContext) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				log := res.telemetry.phaseLogs[int(phaseOffense)]
				if log.count < 2 {
					t.Fatalf("expected both sides in offense phase, got %d", log.count)
				}
				if log.owners[0] != ctx.board.colors[ctx.mover].Opposite() {
					t.Fatalf("expected enemy to act first")
				}
				if res.telemetry.ck != Knight || res.telemetry.dk != King {
					t.Fatalf("expected reversed CK/QK/DK ordering")
				}
			},
		},
		{
			name: "anarchist sadist overrides",
			side: NewAbilitySet(AbilityAnarchist, AbilitySadist),
			expect: func(t *testing.T, res resolveResult, err error, _ *boardSoA, _ *resolveContext) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res.telemetry.anarchist != AbilityScatterShot {
					t.Fatalf("expected anarchist override")
				}
				if res.telemetry.sadist != AbilityScatterShot {
					t.Fatalf("expected sadist override to follow anarchist")
				}
			},
		},
		{
			name: "block path directive",
			side: NewAbilitySet(AbilityBlockPath),
			setup: func(_ *boardSoA, ctx *resolveContext) {
				ctx.requestedDir = DirN
			},
			expect: func(t *testing.T, res resolveResult, err error, _ *boardSoA, ctx *resolveContext) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !res.setBlock || res.blockDir != ctx.requestedDir {
					t.Fatalf("expected block direction honored")
				}
			},
		},
		{
			name: "conflicting augmentors",
			side: NewAbilitySet(AbilityMistShroud, AbilityRadiantVision),
			expect: func(t *testing.T, _ resolveResult, err error, _ *boardSoA, _ *resolveContext) {
				if err != ErrConflictingAugmentors {
					t.Fatalf("expected conflicting augmentors error, got %v", err)
				}
			},
		},
		{
			name: "invalid overload",
			side: NewAbilitySet(AbilityOverload),
			setup: func(b *boardSoA, _ *resolveContext) {
				b.ability[0] = 0
			},
			expect: func(t *testing.T, _ resolveResult, err error, _ *boardSoA, _ *resolveContext) {
				if err != ErrInvalidOverload {
					t.Fatalf("expected invalid overload error, got %v", err)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			board := newEmptyBoard()
			addPiece(&board, 0, 1, White, Knight, SquareD4)
			board.turn = White
			board.ability[0] = tc.side
			doOver := [2]bool{}
			ctx := resolveContext{
				board:        &board,
				mover:        0,
				target:       SquareE4,
				captureIdx:   -1,
				sideMask:     tc.side,
				enemyMask:    tc.enemy,
				doOverUsed:   &doOver,
				requestedDir: DirE,
				sideElement:  ElementFire,
				enemyElement: ElementWater,
				seed:         0xACE0FACE,
			}
			if tc.setup != nil {
				tc.setup(&board, &ctx)
			}
			resolver := newAbilityResolver()
			res, err := resolver.resolve(ctx)
			tc.expect(t, res, err, &board, &ctx)
		})
	}
}
