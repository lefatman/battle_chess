package shared

import "strings"

type AbilityList []Ability

func (al AbilityList) Contains(target Ability) bool {
	for _, ability := range al {
		if ability == target {
			return true
		}
	}
	return false
}

func (al AbilityList) Clone() AbilityList {
	if len(al) == 0 {
		return nil
	}
	out := make(AbilityList, len(al))
	copy(out, al)
	return out
}

func (al AbilityList) Without(target Ability) AbilityList {
	if len(al) == 0 {
		return nil
	}
	out := make(AbilityList, 0, len(al))
	for _, ability := range al {
		if ability != target {
			out = append(out, ability)
		}
	}
	return out
}

func (al AbilityList) Strings() []string {
	out := make([]string, len(al))
	for i, ability := range al {
		out[i] = ability.String()
	}
	return out
}

func (a Ability) String() string {
	switch a {
	case AbilityNone:
		return "None"
	case AbilityDoOver:
		return "DoOver"
	case AbilityBlockPath:
		return "BlockPath"
	case AbilityDoubleKill:
		return "DoubleKill"
	case AbilityObstinant:
		return "Obstinant"
	case AbilityScorch:
		return "Scorch"
	case AbilityBlazeRush:
		return "BlazeRush"
	case AbilityFloodWake:
		return "FloodWake"
	case AbilityMistShroud:
		return "MistShroud"
	case AbilityBastion:
		return "Bastion"
	case AbilityGaleLift:
		return "GaleLift"
	case AbilityTailwind:
		return "Tailwind"
	case AbilityScatterShot:
		return "ScatterShot"
	case AbilityOverload:
		return "Overload"
	case AbilityRadiantVision:
		return "RadiantVision"
	case AbilityUmbralStep:
		return "UmbralStep"
	case AbilitySideStep:
		return "SideStep"
	case AbilityQuantumStep:
		return "QuantumStep"
	case AbilityStalwart:
		return "Stalwart"
	case AbilityBelligerent:
		return "Belligerent"
	case AbilityIndomitable:
		return "Indomitable"
	case AbilityQuantumKill:
		return "QuantumKill"
	case AbilityChainKill:
		return "ChainKill"
	case AbilityPoisonousMeat:
		return "PoisonousMeat"
	case AbilityResurrection:
		return "Resurrection"
	case AbilityTemporalLock:
		return "TemporalLock"
	case AbilitySchrodingersLaugh:
		return "SchrodingersLaugh"
	default:
		return "?"
	}
}

func AbilityStrings() []string {
	out := make([]string, len(AllAbilities))
	for i, a := range AllAbilities {
		out[i] = a.String()
	}
	return out
}

func ElementStrings() []string {
	out := make([]string, len(AllElements))
	for i, e := range AllElements {
		out[i] = e.String()
	}
	return out
}

var AllAbilities = []Ability{
	AbilityDoOver,
	AbilityBlockPath,
	AbilityDoubleKill,
	AbilityObstinant,
	AbilityScorch,
	AbilityBlazeRush,
	AbilityFloodWake,
	AbilityMistShroud,
	AbilityBastion,
	AbilityGaleLift,
	AbilityTailwind,
	AbilityScatterShot,
	AbilityOverload,
	AbilityRadiantVision,
	AbilityUmbralStep,
	AbilitySideStep,
	AbilityQuantumStep,
	AbilityStalwart,
	AbilityBelligerent,
	AbilityIndomitable,
	AbilityQuantumKill,
	AbilityChainKill,
	AbilityPoisonousMeat,
	AbilityResurrection,
	AbilityTemporalLock,
	AbilitySchrodingersLaugh,
}

var AllElements = []Element{
	ElementLight,
	ElementShadow,
	ElementFire,
	ElementWater,
	ElementEarth,
	ElementAir,
	ElementLightning,
}

func ParseAbility(s string) (Ability, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "doover", "do over":
		return AbilityDoOver, true
	case "blockpath", "block path":
		return AbilityBlockPath, true
	case "doublekill", "double kill":
		return AbilityDoubleKill, true
	case "obstinant":
		return AbilityObstinant, true
	case "scorch":
		return AbilityScorch, true
	case "blazerush", "blaze rush":
		return AbilityBlazeRush, true
	case "floodwake", "flood wake":
		return AbilityFloodWake, true
	case "mistshroud", "mist shroud":
		return AbilityMistShroud, true
	case "bastion":
		return AbilityBastion, true
	case "galelift", "gale lift":
		return AbilityGaleLift, true
	case "tailwind":
		return AbilityTailwind, true
	case "scattershot", "scatter shot":
		return AbilityScatterShot, true
	case "overload":
		return AbilityOverload, true
	case "radiantvision", "radiant vision":
		return AbilityRadiantVision, true
	case "umbralstep", "umbral step":
		return AbilityUmbralStep, true
	case "sidestep", "side step":
		return AbilitySideStep, true
	case "quantumstep", "quantum step":
		return AbilityQuantumStep, true
	case "stalwart":
		return AbilityStalwart, true
	case "belligerent":
		return AbilityBelligerent, true
	case "indomitable":
		return AbilityIndomitable, true
	case "quantumkill", "quantum kill":
		return AbilityQuantumKill, true
	case "chainkill", "chain kill":
		return AbilityChainKill, true
	case "poisonousmeat", "poisonous meat":
		return AbilityPoisonousMeat, true
	case "resurrection":
		return AbilityResurrection, true
	case "temporallock", "temporal lock":
		return AbilityTemporalLock, true
	case "schrodingerslaugh", "schrodinger's laugh":
		return AbilitySchrodingersLaugh, true
	default:
		return AbilityNone, false
	}
}

func ParseElement(s string) (Element, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "light":
		return ElementLight, true
	case "shadow":
		return ElementShadow, true
	case "fire":
		return ElementFire, true
	case "water":
		return ElementWater, true
	case "earth":
		return ElementEarth, true
	case "air":
		return ElementAir, true
	case "lightning":
		return ElementLightning, true
	case "none":
		return ElementNone, true
	default:
		return ElementLight, false
	}
}
