// path: chessTest/internal/game/ability_resolver.go
package game

import "math/bits"

const (
	abilityCountInt  = int(abilityCount)
	phaseCount       = 5
	maxPhaseEntries  = 16
	overloadCapacity = 16
)

type abilityPhase uint8

const (
	phaseElemental abilityPhase = iota
	phaseAugmentor
	phaseOffense
	phaseTemporal
	phaseResolution
)

type rngState struct{ seed uint64 }

func newRNG(seed uint64) rngState {
	if seed == 0 {
		seed = 1
	}
	return rngState{seed: seed}
}

func (r *rngState) next() uint32 {
	s := r.seed
	s ^= s << 7
	s ^= s >> 9
	s ^= s << 8
	r.seed = s
	return uint32(s)
}

type phaseScratch struct {
	ability  [maxPhaseEntries]Ability
	owner    [maxPhaseEntries]Color
	piece    [maxPhaseEntries]int8
	priority [maxPhaseEntries]uint8
	count    uint8
}

func (p *phaseScratch) reset() { p.count = 0 }

func (p *phaseScratch) push(id Ability, owner Color, piece int, pri uint8) {
	if p.count >= maxPhaseEntries {
		return
	}
	p.ability[p.count] = id
	p.owner[p.count] = owner
	p.piece[p.count] = int8(piece)
	p.priority[p.count] = pri
	p.count++
}

func (p *phaseScratch) sort() {
	for i := 1; i < int(p.count); i++ {
		pri := p.priority[i]
		ability := p.ability[i]
		owner := p.owner[i]
		piece := p.piece[i]
		j := i - 1
		for j >= 0 {
			if p.priority[j] <= pri {
				break
			}
			p.priority[j+1] = p.priority[j]
			p.ability[j+1] = p.ability[j]
			p.owner[j+1] = p.owner[j]
			p.piece[j+1] = p.piece[j]
			j--
		}
		p.priority[j+1] = pri
		p.ability[j+1] = ability
		p.owner[j+1] = owner
		p.piece[j+1] = piece
	}
}

type phaseExecution struct {
	abilities [maxPhaseEntries]Ability
	owners    [maxPhaseEntries]Color
	count     uint8
}

func (p *phaseExecution) record(id Ability, owner Color) {
	if p.count >= maxPhaseEntries {
		return
	}
	p.abilities[p.count] = id
	p.owners[p.count] = owner
	p.count++
}

type resolveTelemetry struct {
	phaseLogs           [phaseCount]phaseExecution
	firewallSquares     [4]Square
	firewallCount       uint8
	blazeDK             uint8
	blazeQK             uint8
	floodWakePersistent bool
	mistShroudQS        uint8
	bastion             bool
	sturdy              bool
	gale                bool
	tailwindBridge      bool
	scatterHits         uint8
	overload            [overloadCapacity]Ability
	overloadCount       uint8
	raijinFollow        bool
	radiantVision       bool
	blindingSkipped     bool
	anarchist           Ability
	sadist              Ability
	ck                  PieceType
	qk                  PieceType
	dk                  PieceType
}

type resolveContext struct {
	board        *boardSoA
	mover        int
	target       Square
	captureIdx   int
	sideMask     AbilitySet
	enemyMask    AbilitySet
	doOverUsed   *[2]bool
	requestedDir Direction
	sideElement  Element
	enemyElement Element
	seed         uint64
	elemental    phaseScratch
	augmentor    phaseScratch
	offense      phaseScratch
	temporal     phaseScratch
	resolution   phaseScratch
	rng          rngState
}

type resolveResult struct {
	doOver    bool
	blockDir  Direction
	setBlock  bool
	telemetry resolveTelemetry
}

type abilitySource struct {
	color Color
	mask  AbilitySet
	piece int
}

type sideState struct {
	mask     AbilitySet
	piece    AbilitySet
	combined AbilitySet
	element  Element
}

type resolveState struct {
	moverColor Color
	enemyColor Color
	sides      [2]sideState
	floodWake  [2]bool
	tailwind   [2]bool
	mist       [2]bool
	lightSpeed [2]bool
	bastion    [2]bool
	sturdy     [2]bool
	override   Ability
}

type abilityHandler func(*resolveContext, *resolveResult, *resolveState, abilitySource)

type abilityMeta struct {
	phase        abilityPhase
	basePriority uint8
	handler      abilityHandler
}

var abilityMetaTable = [abilityCountInt]abilityMeta{
	AbilityDoOver:        {phaseTemporal, 1, handleDoOver},
	AbilityBlockPath:     {phaseResolution, 2, handleBlockPath},
	AbilityMistShroud:    {phaseAugmentor, 1, handleMistShroud},
	AbilityTailwind:      {phaseAugmentor, 2, handleTailwind},
	AbilityScatterShot:   {phaseOffense, 2, handleScatterShot},
	AbilityOverload:      {phaseAugmentor, 3, handleOverload},
	AbilityRadiantVision: {phaseElemental, 2, handleRadiantVision},
	AbilityLightSpeed:    {phaseOffense, 0, handleLightSpeed},
	AbilityScorch:        {phaseElemental, 1, handleScorch},
	AbilityBlazeRush:     {phaseOffense, 1, handleBlazeRush},
	AbilityFloodWake:     {phaseElemental, 0, handleFloodWake},
	AbilityBastion:       {phaseAugmentor, 0, handleBastion},
	AbilitySturdy:        {phaseAugmentor, 1, handleSturdy},
	AbilityGaleLift:      {phaseAugmentor, 2, handleGaleLift},
	AbilityRaijin:        {phaseOffense, 3, handleRaijin},
	AbilityBlinding:      {phaseTemporal, 2, handleBlinding},
	AbilityAnarchist:     {phaseResolution, 0, handleAnarchist},
	AbilitySadist:        {phaseResolution, 1, handleSadist},
}

type abilityResolver struct{}

func newAbilityResolver() abilityResolver { return abilityResolver{} }

func (r abilityResolver) resolve(ctx resolveContext) (resolveResult, error) {
	var res resolveResult
	state, err := r.initState(&ctx)
	if err != nil {
		return res, err
	}
	r.collect(&ctx, &state)
	r.runPhase(&ctx, &state, phaseElemental, &ctx.elemental, &res)
	r.runPhase(&ctx, &state, phaseAugmentor, &ctx.augmentor, &res)
	r.runPhase(&ctx, &state, phaseOffense, &ctx.offense, &res)
	r.runPhase(&ctx, &state, phaseTemporal, &ctx.temporal, &res)
	r.runPhase(&ctx, &state, phaseResolution, &ctx.resolution, &res)
	r.finalize(&ctx, &state, &res)
	return res, nil
}

func (abilityResolver) initState(ctx *resolveContext) (resolveState, error) {
	var state resolveState
	state.moverColor = ctx.board.colors[ctx.mover]
	state.enemyColor = state.moverColor.Opposite()
	moverIdx := state.moverColor.Index()
	enemyIdx := state.enemyColor.Index()
	rawMoverMask := ctx.board.ability[ctx.mover]
	moverPieceMask := rawMoverMask
	if moverPieceMask == 0 {
		moverPieceMask = ctx.sideMask
	}
	enemyPieceMask := AbilitySet(0)
	if ctx.captureIdx >= 0 {
		enemyPieceMask = ctx.board.ability[ctx.captureIdx]
	}
	if enemyPieceMask == 0 {
		enemyPieceMask = ctx.enemyMask
	}
	state.sides[moverIdx] = sideState{
		mask:     ctx.sideMask,
		piece:    moverPieceMask,
		combined: moverPieceMask | ctx.sideMask,
		element:  ctx.sideElement,
	}
	state.sides[enemyIdx] = sideState{
		mask:     ctx.enemyMask,
		piece:    enemyPieceMask,
		combined: enemyPieceMask | ctx.enemyMask,
		element:  ctx.enemyElement,
	}
	moverCombined := state.sides[moverIdx].combined
	if moverCombined.Has(AbilityMistShroud) && moverCombined.Has(AbilityRadiantVision) {
		return state, ErrConflictingAugmentors
	}
	if moverCombined.Has(AbilityOverload) && rawMoverMask == 0 {
		return state, ErrInvalidOverload
	}
	ctx.rng = newRNG(ctx.seed ^ uint64(ctx.board.ids[ctx.mover])<<1 ^ uint64(ctx.target))
	state.floodWake[moverIdx] = moverCombined.Has(AbilityFloodWake)
	state.floodWake[enemyIdx] = state.sides[enemyIdx].combined.Has(AbilityFloodWake)
	state.tailwind[moverIdx] = moverCombined.Has(AbilityTailwind)
	state.tailwind[enemyIdx] = state.sides[enemyIdx].combined.Has(AbilityTailwind)
	state.mist[moverIdx] = moverCombined.Has(AbilityMistShroud)
	state.mist[enemyIdx] = state.sides[enemyIdx].combined.Has(AbilityMistShroud)
	state.lightSpeed[moverIdx] = moverCombined.Has(AbilityLightSpeed)
	state.lightSpeed[enemyIdx] = state.sides[enemyIdx].combined.Has(AbilityLightSpeed)
	return state, nil
}

func (abilityResolver) collect(ctx *resolveContext, state *resolveState) {
	ctx.elemental.reset()
	ctx.augmentor.reset()
	ctx.offense.reset()
	ctx.temporal.reset()
	ctx.resolution.reset()
	moverPiece := ctx.mover
	enemyPiece := -1
	if ctx.captureIdx >= 0 {
		enemyPiece = ctx.captureIdx
	}
	abilityResolverQueue(ctx, state, state.moverColor, moverPiece, state.sides[state.moverColor.Index()].combined)
	abilityResolverQueue(ctx, state, state.enemyColor, enemyPiece, state.sides[state.enemyColor.Index()].combined)
}

func abilityResolverQueue(ctx *resolveContext, state *resolveState, color Color, piece int, mask AbilitySet) {
	if mask == 0 {
		return
	}
	idx := color.Index()
	for ability := Ability(1); ability < abilityCount; ability++ {
		if !mask.Has(ability) {
			continue
		}
		meta := abilityMetaTable[int(ability)]
		if meta.handler == nil {
			continue
		}
		priority := meta.basePriority
		if state.lightSpeed[idx] {
			priority = 0
		}
		switch meta.phase {
		case phaseElemental:
			ctx.elemental.push(ability, color, piece, priority)
		case phaseAugmentor:
			ctx.augmentor.push(ability, color, piece, priority)
		case phaseOffense:
			ctx.offense.push(ability, color, piece, priority)
		case phaseTemporal:
			ctx.temporal.push(ability, color, piece, priority)
		case phaseResolution:
			ctx.resolution.push(ability, color, piece, priority)
		}
	}
}

func (r abilityResolver) runPhase(ctx *resolveContext, state *resolveState, phase abilityPhase, scratch *phaseScratch, res *resolveResult) {
	if scratch.count == 0 {
		return
	}
	scratch.sort()
	for i := uint8(0); i < scratch.count; i++ {
		ability := scratch.ability[i]
		owner := scratch.owner[i]
		idx := owner.Index()
		piece := int(scratch.piece[i])
		meta := abilityMetaTable[int(ability)]
		if meta.handler != nil {
			src := abilitySource{color: owner, mask: state.sides[idx].combined, piece: piece}
			meta.handler(ctx, res, state, src)
		}
		res.telemetry.phaseLogs[int(phase)].record(ability, owner)
	}
}

func (abilityResolver) finalize(ctx *resolveContext, state *resolveState, res *resolveResult) {
	moverIdx := state.moverColor.Index()
	enemyIdx := state.enemyColor.Index()
	if state.tailwind[moverIdx] && state.mist[moverIdx] {
		res.telemetry.tailwindBridge = true
	}
	if state.tailwind[enemyIdx] && state.mist[enemyIdx] {
		res.telemetry.tailwindBridge = true
	}
	if res.telemetry.floodWakePersistent || state.floodWake[moverIdx] {
		res.telemetry.floodWakePersistent = true
	}
	if state.lightSpeed[enemyIdx] && !state.lightSpeed[moverIdx] {
		res.telemetry.ck = Knight
		res.telemetry.qk = Queen
		res.telemetry.dk = King
	} else {
		res.telemetry.ck = King
		res.telemetry.qk = Queen
		res.telemetry.dk = Knight
	}
	if res.doOver {
		res.telemetry.floodWakePersistent = res.telemetry.floodWakePersistent || state.floodWake[enemyIdx]
	}
}

func handleDoOver(ctx *resolveContext, res *resolveResult, state *resolveState, src abilitySource) {
	if ctx.captureIdx < 0 {
		return
	}
	color := ctx.board.colors[ctx.captureIdx]
	idx := color.Index()
	if (*ctx.doOverUsed)[idx] {
		return
	}
	(*ctx.doOverUsed)[idx] = true
	res.doOver = true
	if state.floodWake[idx] {
		res.telemetry.floodWakePersistent = true
	}
}

func handleBlockPath(ctx *resolveContext, res *resolveResult, _ *resolveState, src abilitySource) {
	if src.color != ctx.board.colors[ctx.mover] {
		return
	}
	if ctx.requestedDir == DirNone {
		return
	}
	res.blockDir = ctx.requestedDir
	res.setBlock = true
}

func handleMistShroud(_ *resolveContext, res *resolveResult, state *resolveState, src abilitySource) {
	idx := src.color.Index()
	state.mist[idx] = true
	res.telemetry.mistShroudQS++
}

func handleTailwind(_ *resolveContext, res *resolveResult, state *resolveState, src abilitySource) {
	idx := src.color.Index()
	state.tailwind[idx] = true
	if state.mist[idx] {
		res.telemetry.tailwindBridge = true
	}
}

func handleScatterShot(ctx *resolveContext, res *resolveResult, state *resolveState, src abilitySource) {
	enemy := src.color.Opposite()
	dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	start := int(ctx.rng.next() % uint32(len(dirs)))
	for n := 0; n < len(dirs); n++ {
		dir := dirs[(start+n)%len(dirs)]
		sq := offsetSquare(ctx.target, dir[1], dir[0])
		if sq == SquareInvalid {
			continue
		}
		idx := ctx.board.pieceIndexBySquare(sq)
		if idx < 0 || ctx.board.colors[idx] != enemy {
			continue
		}
		ctx.board.removePiece(idx)
		if res.telemetry.scatterHits < 4 {
			res.telemetry.scatterHits++
		}
	}
	_ = state
}

func handleOverload(ctx *resolveContext, res *resolveResult, state *resolveState, src abilitySource) {
	idx := src.color.Index()
	res.telemetry.overloadCount = 0
	for i := range ctx.board.ids {
		if !ctx.board.alive[i] || ctx.board.colors[i] != src.color {
			continue
		}
		mask := ctx.board.ability[i]
		if mask == 0 {
			mask = state.sides[idx].mask
		}
		ability := firstAbility(mask)
		if ability == AbilityNone {
			continue
		}
		if res.telemetry.overloadCount >= overloadCapacity {
			break
		}
		res.telemetry.overload[res.telemetry.overloadCount] = ability
		res.telemetry.overloadCount++
	}
}

func handleRadiantVision(_ *resolveContext, res *resolveResult, _ *resolveState, _ abilitySource) {
	res.telemetry.radiantVision = true
}

func handleLightSpeed(_ *resolveContext, _ *resolveResult, state *resolveState, src abilitySource) {
	state.lightSpeed[src.color.Index()] = true
}

func handleScorch(ctx *resolveContext, res *resolveResult, _ *resolveState, _ abilitySource) {
	offsets := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	count := res.telemetry.firewallCount
	for _, delta := range offsets {
		if count >= uint8(len(res.telemetry.firewallSquares)) {
			break
		}
		sq := offsetSquare(ctx.target, delta[1], delta[0])
		if sq == SquareInvalid {
			continue
		}
		res.telemetry.firewallSquares[count] = sq
		count++
	}
	res.telemetry.firewallCount = count
}

func handleBlazeRush(ctx *resolveContext, res *resolveResult, _ *resolveState, src abilitySource) {
	if src.piece < 0 {
		return
	}
	switch ctx.board.types[src.piece] {
	case Knight:
		if res.telemetry.blazeQK < 3 {
			res.telemetry.blazeQK++
		}
	case Queen:
		if res.telemetry.blazeDK < 3 {
			res.telemetry.blazeDK++
		}
	default:
		if res.telemetry.blazeDK < 3 {
			res.telemetry.blazeDK++
		}
	}
}

func handleFloodWake(_ *resolveContext, res *resolveResult, state *resolveState, src abilitySource) {
	idx := src.color.Index()
	state.floodWake[idx] = true
	res.telemetry.floodWakePersistent = true
}

func handleBastion(_ *resolveContext, res *resolveResult, state *resolveState, src abilitySource) {
	idx := src.color.Index()
	state.bastion[idx] = true
	res.telemetry.bastion = true
}

func handleSturdy(_ *resolveContext, res *resolveResult, state *resolveState, src abilitySource) {
	idx := src.color.Index()
	state.sturdy[idx] = true
	res.telemetry.sturdy = true
}

func handleGaleLift(_ *resolveContext, res *resolveResult, state *resolveState, src abilitySource) {
	idx := src.color.Index()
	if state.bastion[idx] && state.sturdy[idx] {
		res.telemetry.gale = true
	}
}

func handleRaijin(_ *resolveContext, res *resolveResult, _ *resolveState, _ abilitySource) {
	res.telemetry.raijinFollow = true
}

func handleBlinding(_ *resolveContext, res *resolveResult, _ *resolveState, _ abilitySource) {
	res.telemetry.blindingSkipped = true
}

func handleAnarchist(_ *resolveContext, res *resolveResult, state *resolveState, _ abilitySource) {
	state.override = AbilityScatterShot
	res.telemetry.anarchist = AbilityScatterShot
}

func handleSadist(_ *resolveContext, res *resolveResult, state *resolveState, _ abilitySource) {
	if state.override != AbilityNone {
		res.telemetry.sadist = state.override
	} else {
		res.telemetry.sadist = AbilitySadist
	}
}

func firstAbility(set AbilitySet) Ability {
	bitsVal := uint64(set)
	if bitsVal == 0 {
		return AbilityNone
	}
	idx := bits.TrailingZeros64(bitsVal)
	if idx <= 0 {
		return AbilityNone
	}
	return Ability(idx)
}

func offsetSquare(base Square, rankDelta, fileDelta int) Square {
	if base == SquareInvalid {
		return SquareInvalid
	}
	rank := int(base) / 8
	file := int(base) % 8
	rank += rankDelta
	file += fileDelta
	if rank < 0 || rank >= 8 || file < 0 || file >= 8 {
		return SquareInvalid
	}
	return Square(rank*8 + file)
}
