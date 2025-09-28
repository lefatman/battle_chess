package game

import "battle_chess_poc/internal/shared"

type (
	Color     = shared.Color
	PieceType = shared.PieceType
	Element   = shared.Element
	Ability   = shared.Ability
	Square    = shared.Square
	Direction = shared.Direction

	AbilityList = shared.AbilityList
)

const (
	White = shared.White
	Black = shared.Black

	Pawn   = shared.Pawn
	Knight = shared.Knight
	Bishop = shared.Bishop
	Rook   = shared.Rook
	Queen  = shared.Queen
	King   = shared.King

	ElementLight     = shared.ElementLight
	ElementShadow    = shared.ElementShadow
	ElementFire      = shared.ElementFire
	ElementWater     = shared.ElementWater
	ElementEarth     = shared.ElementEarth
	ElementAir       = shared.ElementAir
	ElementLightning = shared.ElementLightning
	ElementNone      = shared.ElementNone

	AbilityNone              = shared.AbilityNone
	AbilityDoOver            = shared.AbilityDoOver
	AbilityBlockPath         = shared.AbilityBlockPath
	AbilityDoubleKill        = shared.AbilityDoubleKill
	AbilityObstinant         = shared.AbilityObstinant
	AbilityScorch            = shared.AbilityScorch
	AbilityBlazeRush         = shared.AbilityBlazeRush
	AbilityFloodWake         = shared.AbilityFloodWake
	AbilityMistShroud        = shared.AbilityMistShroud
	AbilityBastion           = shared.AbilityBastion
	AbilityGaleLift          = shared.AbilityGaleLift
	AbilityTailwind          = shared.AbilityTailwind
	AbilityScatterShot       = shared.AbilityScatterShot
	AbilityOverload          = shared.AbilityOverload
	AbilityRadiantVision     = shared.AbilityRadiantVision
	AbilityUmbralStep        = shared.AbilityUmbralStep
	AbilitySideStep          = shared.AbilitySideStep
	AbilityQuantumStep       = shared.AbilityQuantumStep
	AbilityStalwart          = shared.AbilityStalwart
	AbilityBelligerent       = shared.AbilityBelligerent
	AbilityIndomitable       = shared.AbilityIndomitable
	AbilityQuantumKill       = shared.AbilityQuantumKill
	AbilityChainKill         = shared.AbilityChainKill
	AbilityPoisonousMeat     = shared.AbilityPoisonousMeat
	AbilityResurrection      = shared.AbilityResurrection
	AbilityTemporalLock      = shared.AbilityTemporalLock
	AbilitySchrodingersLaugh = shared.AbilitySchrodingersLaugh

	DirN    = shared.DirN
	DirNE   = shared.DirNE
	DirE    = shared.DirE
	DirSE   = shared.DirSE
	DirS    = shared.DirS
	DirSW   = shared.DirSW
	DirW    = shared.DirW
	DirNW   = shared.DirNW
	DirNone = shared.DirNone
)

var (
	AllAbilities = shared.AllAbilities
	AllElements  = shared.AllElements
)

type SideConfig = struct {
	Ability Ability
	Element Element
}

func AbilityStrings() []string                  { return shared.AbilityStrings() }
func ElementStrings() []string                  { return shared.ElementStrings() }
func ParseAbility(s string) (Ability, bool)     { return shared.ParseAbility(s) }
func ParseElement(s string) (Element, bool)     { return shared.ParseElement(s) }
func ParseDirection(s string) (Direction, bool) { return shared.ParseDirection(s) }
func CoordToSquare(coord string) (Square, bool) { return shared.CoordToSquare(coord) }
