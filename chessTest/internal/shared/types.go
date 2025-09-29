package shared

import (
	"fmt"
	"strings"
)

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

// ---------------------------
// Castling Rights
// ---------------------------

type CastlingRights uint8

const (
	CastlingNone          CastlingRights = 0
	CastlingWhiteKingside CastlingRights = 1 << iota
	CastlingWhiteQueenside
	CastlingBlackKingside
	CastlingBlackQueenside
	CastlingAll = CastlingWhiteKingside | CastlingWhiteQueenside | CastlingBlackKingside | CastlingBlackQueenside
)

type CastlingSide uint8

const (
	CastleKingside CastlingSide = iota
	CastleQueenside
)

func (cs CastlingSide) String() string {
	switch cs {
	case CastleKingside:
		return "kingside"
	case CastleQueenside:
		return "queenside"
	default:
		return "?"
	}
}

func CastlingRight(color Color, side CastlingSide) CastlingRights {
	switch color {
	case White:
		if side == CastleQueenside {
			return CastlingWhiteQueenside
		}
		return CastlingWhiteKingside
	case Black:
		if side == CastleQueenside {
			return CastlingBlackQueenside
		}
		return CastlingBlackKingside
	default:
		return CastlingNone
	}
}

func CastlingRightsForColor(color Color) CastlingRights {
	switch color {
	case White:
		return CastlingWhiteKingside | CastlingWhiteQueenside
	case Black:
		return CastlingBlackKingside | CastlingBlackQueenside
	default:
		return CastlingNone
	}
}

func (cr CastlingRights) Has(right CastlingRights) bool { return cr&right != 0 }

func (cr CastlingRights) HasSide(color Color, side CastlingSide) bool {
	return cr.Has(CastlingRight(color, side))
}

func (cr CastlingRights) With(right CastlingRights) CastlingRights { return cr | right }

func (cr CastlingRights) Without(right CastlingRights) CastlingRights { return cr &^ right }

func (cr CastlingRights) WithoutColor(color Color) CastlingRights {
	return cr.Without(CastlingRightsForColor(color))
}

func (cr CastlingRights) String() string {
	if cr == CastlingNone {
		return "-"
	}
	var b strings.Builder
	if cr.Has(CastlingWhiteKingside) {
		b.WriteByte('K')
	}
	if cr.Has(CastlingWhiteQueenside) {
		b.WriteByte('Q')
	}
	if cr.Has(CastlingBlackKingside) {
		b.WriteByte('k')
	}
	if cr.Has(CastlingBlackQueenside) {
		b.WriteByte('q')
	}
	if b.Len() == 0 {
		return "-"
	}
	return b.String()
}

func ParseCastlingRights(s string) (CastlingRights, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" || trimmed == "-" {
		return CastlingNone, nil
	}
	var rights CastlingRights
	for _, r := range trimmed {
		switch r {
		case 'K':
			rights |= CastlingWhiteKingside
		case 'Q':
			rights |= CastlingWhiteQueenside
		case 'k':
			rights |= CastlingBlackKingside
		case 'q':
			rights |= CastlingBlackQueenside
		default:
			return CastlingNone, fmt.Errorf("invalid castling flag %q", string(r))
		}
	}
	return rights, nil
}

func (cr CastlingRights) MarshalText() ([]byte, error) { return []byte(cr.String()), nil }

func (cr *CastlingRights) UnmarshalText(text []byte) error {
	parsed, err := ParseCastlingRights(string(text))
	if err != nil {
		return err
	}
	*cr = parsed
	return nil
}

// ---------------------------
// En-passant targets
// ---------------------------

type EnPassantTarget struct {
	square Square
	valid  bool
}

func NewEnPassantTarget(sq Square) EnPassantTarget { return EnPassantTarget{square: sq, valid: true} }

func NoEnPassantTarget() EnPassantTarget { return EnPassantTarget{} }

func (e EnPassantTarget) Valid() bool { return e.valid }

func (e EnPassantTarget) Square() (Square, bool) {
	if !e.valid {
		return 0, false
	}
	return e.square, true
}

func (e EnPassantTarget) String() string {
	if !e.valid {
		return "-"
	}
	return e.square.String()
}

func ParseEnPassantTarget(s string) (EnPassantTarget, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" || trimmed == "-" {
		return EnPassantTarget{}, nil
	}
	sq, ok := CoordToSquare(strings.ToLower(trimmed))
	if !ok {
		return EnPassantTarget{}, fmt.Errorf("invalid en-passant square %q", s)
	}
	return NewEnPassantTarget(sq), nil
}

func (e EnPassantTarget) MarshalText() ([]byte, error) { return []byte(e.String()), nil }

func (e *EnPassantTarget) UnmarshalText(text []byte) error {
	parsed, err := ParseEnPassantTarget(string(text))
	if err != nil {
		return err
	}
	*e = parsed
	return nil
}

// ---------------------------
// Promotion choices
// ---------------------------

type PromotionChoices uint8

const (
	PromotionNone  PromotionChoices = 0
	PromoteToQueen PromotionChoices = 1 << iota
	PromoteToRook
	PromoteToBishop
	PromoteToKnight
	PromotionAll = PromoteToQueen | PromoteToRook | PromoteToBishop | PromoteToKnight
)

func PromotionChoicesFromTypes(types ...PieceType) PromotionChoices {
	var choices PromotionChoices
	for _, pt := range types {
		choices = choices.WithPiece(pt)
	}
	return choices
}

func (pc PromotionChoices) WithPiece(pt PieceType) PromotionChoices {
	switch pt {
	case Queen:
		return pc | PromoteToQueen
	case Rook:
		return pc | PromoteToRook
	case Bishop:
		return pc | PromoteToBishop
	case Knight:
		return pc | PromoteToKnight
	default:
		return pc
	}
}

func (pc PromotionChoices) WithoutPiece(pt PieceType) PromotionChoices {
	switch pt {
	case Queen:
		return pc &^ PromoteToQueen
	case Rook:
		return pc &^ PromoteToRook
	case Bishop:
		return pc &^ PromoteToBishop
	case Knight:
		return pc &^ PromoteToKnight
	default:
		return pc
	}
}

func (pc PromotionChoices) Contains(pt PieceType) bool {
	switch pt {
	case Queen:
		return pc&PromoteToQueen != 0
	case Rook:
		return pc&PromoteToRook != 0
	case Bishop:
		return pc&PromoteToBishop != 0
	case Knight:
		return pc&PromoteToKnight != 0
	default:
		return false
	}
}

func (pc PromotionChoices) Types() []PieceType {
	var out []PieceType
	for _, pt := range []PieceType{Queen, Rook, Bishop, Knight} {
		if pc.Contains(pt) {
			out = append(out, pt)
		}
	}
	return out
}

func (pc PromotionChoices) Default() PieceType {
	for _, pt := range []PieceType{Queen, Rook, Bishop, Knight} {
		if pc.Contains(pt) {
			return pt
		}
	}
	return Queen
}

func (pc PromotionChoices) String() string {
	if pc == PromotionNone {
		return "-"
	}
	var b strings.Builder
	if pc.Contains(Queen) {
		b.WriteByte('Q')
	}
	if pc.Contains(Rook) {
		b.WriteByte('R')
	}
	if pc.Contains(Bishop) {
		b.WriteByte('B')
	}
	if pc.Contains(Knight) {
		b.WriteByte('N')
	}
	if b.Len() == 0 {
		return "-"
	}
	return b.String()
}

func ParsePromotionPiece(s string) (PieceType, bool) {
	trimmed := strings.TrimSpace(strings.ToLower(s))
	switch trimmed {
	case "q", "queen":
		return Queen, true
	case "r", "rook":
		return Rook, true
	case "b", "bishop":
		return Bishop, true
	case "n", "knight":
		return Knight, true
	default:
		return 0, false
	}
}

func ParsePromotionChoices(s string) (PromotionChoices, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" || trimmed == "-" {
		return PromotionNone, nil
	}
	var choices PromotionChoices
	for _, r := range trimmed {
		if r == ',' || r == ' ' || r == '/' {
			continue
		}
		pt, ok := ParsePromotionPiece(string(r))
		if !ok {
			return PromotionNone, fmt.Errorf("invalid promotion piece %q", string(r))
		}
		choices = choices.WithPiece(pt)
	}
	return choices, nil
}

func (pc PromotionChoices) MarshalText() ([]byte, error) { return []byte(pc.String()), nil }

func (pc *PromotionChoices) UnmarshalText(text []byte) error {
	parsed, err := ParsePromotionChoices(string(text))
	if err != nil {
		return err
	}
	*pc = parsed
	return nil
}
