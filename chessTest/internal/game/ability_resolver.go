// path: chessTest/internal/game/ability_resolver.go
package game

type resolveContext struct {
	board        *boardSoA
	mover        int
	target       Square
	captureIdx   int
	sideMask     AbilitySet
	enemyMask    AbilitySet
	doOverUsed   *[2]bool
	requestedDir Direction
}

type resolveResult struct {
	doOver   bool
	blockDir Direction
	setBlock bool
}

type abilityResolver struct{}

func newAbilityResolver() abilityResolver {
	return abilityResolver{}
}

func (r abilityResolver) resolve(ctx resolveContext) resolveResult {
	if ctx.captureIdx >= 0 && ctx.enemyMask.Has(AbilityDoOver) {
		color := ctx.board.colors[ctx.captureIdx]
		if !ctx.doOverUsed[color.Index()] {
			ctx.doOverUsed[color.Index()] = true
			return resolveResult{doOver: true}
		}
	}
	if ctx.sideMask.Has(AbilityBlockPath) && ctx.requestedDir != DirNone {
		return resolveResult{blockDir: ctx.requestedDir, setBlock: true}
	}
	return resolveResult{}
}
