package shared

import "fmt"

type Color uint8

const (
	White Color = iota
	Black
)

func (c Color) Opposite() Color {
	if c == White {
		return Black
	}
	return White
}

func (c Color) String() string {
	if c == White {
		return "white"
	}
	return "black"
}

type PieceType uint8

const (
	Pawn PieceType = iota
	Knight
	Bishop
	Rook
	Queen
	King
)

func (p PieceType) String() string {
	switch p {
	case Pawn:
		return "P"
	case Knight:
		return "N"
	case Bishop:
		return "B"
	case Rook:
		return "R"
	case Queen:
		return "Q"
	case King:
		return "K"
	default:
		return fmt.Sprintf("piece(%d)", p)
	}
}

type Element uint8

const (
	ElementLight Element = iota
	ElementShadow
	ElementFire
	ElementWater
	ElementEarth
	ElementAir
	ElementLightning
	ElementNone Element = 255
)

func (e Element) String() string {
	switch e {
	case ElementLight:
		return "Light"
	case ElementShadow:
		return "Shadow"
	case ElementFire:
		return "Fire"
	case ElementWater:
		return "Water"
	case ElementEarth:
		return "Earth"
	case ElementAir:
		return "Air"
	case ElementLightning:
		return "Lightning"
	case ElementNone:
		return "None"
	default:
		return "?"
	}
}

type Ability uint8

const (
	AbilityNone Ability = iota
	AbilityDoOver
	AbilityBlockPath
	AbilityDoubleKill
	AbilityObstinant
	AbilityScorch
	AbilityBlazeRush
	AbilityFloodWake
	AbilityMistShroud
	AbilityBastion
	AbilityGaleLift
	AbilityTailwind
	AbilityScatterShot
	AbilityOverload
	AbilityRadiantVision
	AbilityUmbralStep
	AbilitySideStep
	AbilityQuantumStep
	AbilityStalwart
	AbilityBelligerent
	AbilityIndomitable
	AbilityQuantumKill
	AbilityChainKill
	AbilityPoisonousMeat
	AbilityResurrection
	AbilityTemporalLock
	AbilitySchrodingersLaugh
)

type Square uint8

func (s Square) Rank() int { return int(s) >> 3 }
func (s Square) File() int { return int(s) & 7 }

type Direction uint8

const (
	DirN Direction = iota
	DirNE
	DirE
	DirSE
	DirS
	DirSW
	DirW
	DirNW
	DirNone Direction = 255
)

func (d Direction) String() string {
	switch d {
	case DirN:
		return "N"
	case DirNE:
		return "NE"
	case DirE:
		return "E"
	case DirSE:
		return "SE"
	case DirS:
		return "S"
	case DirSW:
		return "SW"
	case DirW:
		return "W"
	case DirNW:
		return "NW"
	case DirNone:
		return "?"
	default:
		return "?"
	}
}

func ParseDirection(s string) (Direction, bool) {
	switch s {
	case "", "?", "NONE", "None":
		return DirNone, true
	case "N":
		return DirN, true
	case "NE":
		return DirNE, true
	case "E":
		return DirE, true
	case "SE":
		return DirSE, true
	case "S":
		return DirS, true
	case "SW":
		return DirSW, true
	case "W":
		return DirW, true
	case "NW":
		return DirNW, true
	default:
		return DirNone, false
	}
}

func (s Square) String() string {
	file := byte('a' + s.File())
	rank := byte('1' + s.Rank())
	return string([]byte{file, rank})
}

func CoordToSquare(coord string) (Square, bool) {
	if len(coord) != 2 {
		return 0, false
	}
	file := coord[0]
	rank := coord[1]
	if file < 'a' || file > 'h' || rank < '1' || rank > '8' {
		return 0, false
	}
	r := int(rank - '1')
	c := int(file - 'a')
	return Square(r*8 + c), true
}
