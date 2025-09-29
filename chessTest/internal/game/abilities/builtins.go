package abilities

import (
	"battle_chess_poc/internal/game"
	"battle_chess_poc/internal/shared"
)

func init() {
	mustRegisterBuiltin(shared.AbilityDoubleKill, game.NewDoubleKillHandler)
	mustRegisterBuiltin(shared.AbilityScorch, game.NewScorchHandler)
	mustRegisterBuiltin(shared.AbilityQuantumKill, game.NewQuantumKillHandler)
	mustRegisterBuiltin(shared.AbilityPoisonousMeat, game.NewPoisonousMeatHandler)
	mustRegisterBuiltin(shared.AbilityOverload, game.NewOverloadHandler)
	mustRegisterBuiltin(shared.AbilityBastion, game.NewBastionHandler)
	mustRegisterBuiltin(shared.AbilityTemporalLock, game.NewTemporalLockHandler)
}

func mustRegisterBuiltin(id shared.Ability, ctor func() game.AbilityHandler) {
	if err := Register(id, func() AbilityHandler { return ctor() }); err != nil {
		panic(err)
	}
}
