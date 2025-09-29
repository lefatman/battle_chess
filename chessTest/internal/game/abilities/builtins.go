// path: chessTest/internal/game/abilities/builtins.go
package abilities

import "battle_chess_poc/internal/game"

func init() {
	mustRegisterBuiltin(game.AbilityDoubleKill, game.NewDoubleKillHandler)
	mustRegisterBuiltin(game.AbilityScorch, game.NewScorchHandler)
	mustRegisterBuiltin(game.AbilityTailwind, game.NewTailwindHandler)
	mustRegisterBuiltin(game.AbilityRadiantVision, game.NewRadiantVisionHandler)
	mustRegisterBuiltin(game.AbilityUmbralStep, game.NewUmbralStepHandler)
	mustRegisterBuiltin(game.AbilitySchrodingersLaugh, game.NewSchrodingersLaughHandler)
	mustRegisterBuiltin(game.AbilityGaleLift, game.NewGaleLiftHandler)
	mustRegisterBuiltin(game.AbilityQuantumKill, game.NewQuantumKillHandler)
	mustRegisterBuiltin(game.AbilityChainKill, game.NewChainKillHandler)
	mustRegisterBuiltin(game.AbilityPoisonousMeat, game.NewPoisonousMeatHandler)
	mustRegisterBuiltin(game.AbilityOverload, game.NewOverloadHandler)
	mustRegisterBuiltin(game.AbilityBastion, game.NewBastionHandler)
	mustRegisterBuiltin(game.AbilityTemporalLock, game.NewTemporalLockHandler)
	mustRegisterBuiltin(game.AbilityResurrection, game.NewResurrectionHandler)
}

func mustRegisterBuiltin(id game.Ability, ctor func() game.AbilityHandler) {
	if err := Register(id, func() AbilityHandler { return ctor() }); err != nil {
		panic(err)
	}
}
