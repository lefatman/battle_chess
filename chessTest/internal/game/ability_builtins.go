// path: chessTest/internal/game/ability_builtins.go
package game

func init() {
	mustRegisterAbilityHandler(AbilityDoubleKill, NewDoubleKillHandler)
	mustRegisterAbilityHandler(AbilityScorch, NewScorchHandler)
	mustRegisterAbilityHandler(AbilityTailwind, NewTailwindHandler)
	mustRegisterAbilityHandler(AbilityRadiantVision, NewRadiantVisionHandler)
	mustRegisterAbilityHandler(AbilityUmbralStep, NewUmbralStepHandler)
	mustRegisterAbilityHandler(AbilitySchrodingersLaugh, NewSchrodingersLaughHandler)
	mustRegisterAbilityHandler(AbilityGaleLift, NewGaleLiftHandler)
	mustRegisterAbilityHandler(AbilityQuantumKill, NewQuantumKillHandler)
	mustRegisterAbilityHandler(AbilityChainKill, NewChainKillHandler)
	mustRegisterAbilityHandler(AbilityPoisonousMeat, NewPoisonousMeatHandler)
	mustRegisterAbilityHandler(AbilityOverload, NewOverloadHandler)
	mustRegisterAbilityHandler(AbilityBastion, NewBastionHandler)
	mustRegisterAbilityHandler(AbilityTemporalLock, NewTemporalLockHandler)
	mustRegisterAbilityHandler(AbilityResurrection, NewResurrectionHandler)
}

func mustRegisterAbilityHandler(id Ability, ctor func() AbilityHandler) {
	if err := RegisterAbilityHandler(id, ctor); err != nil {
		panic(err)
	}
}
