// path: chessTest/internal/game/ability_registry.go
package game

import "strings"

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

func (c Color) Index() int { return int(c) }

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

type Square uint8

const (
	SquareA1 Square = iota
	SquareB1
	SquareC1
	SquareD1
	SquareE1
	SquareF1
	SquareG1
	SquareH1
	SquareA2
	SquareB2
	SquareC2
	SquareD2
	SquareE2
	SquareF2
	SquareG2
	SquareH2
	SquareA3
	SquareB3
	SquareC3
	SquareD3
	SquareE3
	SquareF3
	SquareG3
	SquareH3
	SquareA4
	SquareB4
	SquareC4
	SquareD4
	SquareE4
	SquareF4
	SquareG4
	SquareH4
	SquareA5
	SquareB5
	SquareC5
	SquareD5
	SquareE5
	SquareF5
	SquareG5
	SquareH5
	SquareA6
	SquareB6
	SquareC6
	SquareD6
	SquareE6
	SquareF6
	SquareG6
	SquareH6
	SquareA7
	SquareB7
	SquareC7
	SquareD7
	SquareE7
	SquareF7
	SquareG7
	SquareH7
	SquareA8
	SquareB8
	SquareC8
	SquareD8
	SquareE8
	SquareF8
	SquareG8
	SquareH8
	SquareInvalid Square = 255
)

type Direction uint8

const (
	DirNone Direction = iota
	DirN
	DirNE
	DirE
	DirSE
	DirS
	DirSW
	DirW
	DirNW
)

func ParseDirection(s string) Direction {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "N":
		return DirN
	case "NE":
		return DirNE
	case "E":
		return DirE
	case "SE":
		return DirSE
	case "S":
		return DirS
	case "SW":
		return DirSW
	case "W":
		return DirW
	case "NW":
		return DirNW
	default:
		return DirNone
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

var elementNames = [...]string{
	ElementLight:     "Light",
	ElementShadow:    "Shadow",
	ElementFire:      "Fire",
	ElementWater:     "Water",
	ElementEarth:     "Earth",
	ElementAir:       "Air",
	ElementLightning: "Lightning",
}

func (e Element) String() string {
	if int(e) < len(elementNames) {
		if name := elementNames[e]; name != "" {
			return name
		}
	}
	return "None"
}

type Ability uint8

const (
	AbilityNone Ability = iota
	AbilityDoOver
	AbilityBlockPath
	AbilityMistShroud
	AbilityTailwind
	AbilityScatterShot
	AbilityOverload
	AbilityRadiantVision
	AbilityLightSpeed
	AbilityScorch
	AbilityBlazeRush
	AbilityFloodWake
	abilityCount
)

type AbilityList []Ability

type AbilitySet uint64

func abilityBit(id Ability) AbilitySet {
	idx := int(id)
	if idx <= 0 || idx >= int(abilityCount) {
		return 0
	}
	return AbilitySet(1) << uint(idx)
}

func (s AbilitySet) With(id Ability) AbilitySet { return s | abilityBit(id) }

func (s AbilitySet) Has(id Ability) bool { return s&abilityBit(id) != 0 }

func NewAbilitySet(ids ...Ability) AbilitySet {
	var out AbilitySet
	for _, id := range ids {
		out |= abilityBit(id)
	}
	return out
}

type abilityEntry struct {
	id      Ability
	name    string
	aliases []string
}

var abilityCatalog = []abilityEntry{
	{AbilityDoOver, "DoOver", []string{"do over", "doover"}},
	{AbilityBlockPath, "BlockPath", []string{"block path", "blockpiece", "block piece"}},
	{AbilityMistShroud, "MistShroud", []string{"mist shroud"}},
	{AbilityTailwind, "Tailwind", []string{"tail wind"}},
	{AbilityScatterShot, "ScatterShot", []string{"scatter shot"}},
	{AbilityOverload, "Overload", []string{"over load"}},
	{AbilityRadiantVision, "RadiantVision", []string{"radiant vision"}},
	{AbilityLightSpeed, "LightSpeed", []string{"light speed"}},
	{AbilityScorch, "Scorch", nil},
	{AbilityBlazeRush, "BlazeRush", []string{"blaze rush"}},
	{AbilityFloodWake, "FloodWake", []string{"flood wake"}},
}

var abilityNameByID map[Ability]string
var abilityLookup map[string]Ability
var AllAbilities []Ability
var AllElements = []Element{ElementLight, ElementShadow, ElementFire, ElementWater, ElementEarth, ElementAir, ElementLightning}

func init() {
	abilityNameByID = make(map[Ability]string, len(abilityCatalog))
	abilityLookup = make(map[string]Ability, len(abilityCatalog)*2)
	for _, entry := range abilityCatalog {
		abilityNameByID[entry.id] = entry.name
		abilityLookup[strings.ToLower(entry.name)] = entry.id
		for _, alias := range entry.aliases {
			abilityLookup[strings.ToLower(alias)] = entry.id
		}
		AllAbilities = append(AllAbilities, entry.id)
	}
}

func (a Ability) String() string {
	if name, ok := abilityNameByID[a]; ok {
		return name
	}
	return "Unknown"
}

func ParseAbility(s string) (Ability, bool) {
	normalized := strings.ToLower(strings.TrimSpace(s))
	if normalized == "" {
		return AbilityNone, false
	}
	id, ok := abilityLookup[normalized]
	return id, ok
}

func AbilityStrings() []string {
	out := make([]string, 0, len(abilityCatalog))
	for _, entry := range abilityCatalog {
		out = append(out, entry.name)
	}
	return out
}

func ParseElement(s string) (Element, bool) {
	normalized := strings.ToLower(strings.TrimSpace(s))
	switch normalized {
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
	default:
		return ElementNone, false
	}
}

func ElementStrings() []string {
	return []string{"Light", "Shadow", "Fire", "Water", "Earth", "Air", "Lightning"}
}

func (al AbilityList) Strings() []string {
	out := make([]string, 0, len(al))
	for _, id := range al {
		out = append(out, id.String())
	}
	return out
}
