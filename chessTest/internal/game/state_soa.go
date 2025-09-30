// path: chessTest/internal/game/state_soa.go
package game

type boardSoA struct {
	ids       [32]int
	squares   [32]Square
	types     [32]PieceType
	colors    [32]Color
	alive     [32]bool
	ability   [32]AbilitySet
	occupancy [2]uint64
	pieceMask [2][6]uint64
	turn      Color
	ply       uint32
}

func newBoard() boardSoA {
	var b boardSoA
	idx := 0
	add := func(color Color, typ PieceType, sq Square) {
		b.ids[idx] = idx + 1
		b.squares[idx] = sq
		b.types[idx] = typ
		b.colors[idx] = color
		b.alive[idx] = true
		bit := uint64(1) << uint(sq)
		b.occupancy[color.Index()] |= bit
		b.pieceMask[color.Index()][typ] |= bit
		idx++
	}
	for file := 0; file < 8; file++ {
		add(White, Pawn, Square(uint8(SquareA2)+uint8(file)))
	}
	add(White, Rook, SquareA1)
	add(White, Knight, SquareB1)
	add(White, Bishop, SquareC1)
	add(White, Queen, SquareD1)
	add(White, King, SquareE1)
	add(White, Bishop, SquareF1)
	add(White, Knight, SquareG1)
	add(White, Rook, SquareH1)
	for file := 0; file < 8; file++ {
		add(Black, Pawn, Square(uint8(SquareA7)+uint8(file)))
	}
	add(Black, Rook, SquareA8)
	add(Black, Knight, SquareB8)
	add(Black, Bishop, SquareC8)
	add(Black, Queen, SquareD8)
	add(Black, King, SquareE8)
	add(Black, Bishop, SquareF8)
	add(Black, Knight, SquareG8)
	add(Black, Rook, SquareH8)
	b.turn = White
	return b
}

func (b *boardSoA) clone() boardSoA {
	var out boardSoA
	out = *b
	return out
}

func (b *boardSoA) pieceIndexBySquare(sq Square) int {
	if sq == SquareInvalid {
		return -1
	}
	mask := uint64(1) << uint(sq)
	if b.occupancy[0]&mask == 0 && b.occupancy[1]&mask == 0 {
		return -1
	}
	for i := range b.ids {
		if !b.alive[i] {
			continue
		}
		if b.squares[i] == sq {
			return i
		}
	}
	return -1
}

func (b *boardSoA) movePiece(idx int, to Square) {
	from := b.squares[idx]
	bitFrom := uint64(1) << uint(from)
	bitTo := uint64(1) << uint(to)
	colorIdx := b.colors[idx].Index()
	typ := b.types[idx]
	b.occupancy[colorIdx] &^= bitFrom
	b.occupancy[colorIdx] |= bitTo
	b.pieceMask[colorIdx][typ] &^= bitFrom
	b.pieceMask[colorIdx][typ] |= bitTo
	b.squares[idx] = to
}

func (b *boardSoA) removePiece(idx int) {
	if !b.alive[idx] {
		return
	}
	sq := b.squares[idx]
	bit := uint64(1) << uint(sq)
	colorIdx := b.colors[idx].Index()
	typ := b.types[idx]
	b.occupancy[colorIdx] &^= bit
	b.pieceMask[colorIdx][typ] &^= bit
	b.alive[idx] = false
}

func (b *boardSoA) addAbility(mask AbilitySet, color Color) {
	for i := range b.ids {
		if !b.alive[i] || b.colors[i] != color {
			continue
		}
		b.ability[i] = mask
	}
}

func (b *boardSoA) sideAbility(color Color) AbilitySet {
	for i := range b.ids {
		if !b.alive[i] || b.colors[i] != color {
			continue
		}
		return b.ability[i]
	}
	return 0
}

func (b *boardSoA) squareOccupiedBy(color Color, sq Square) bool {
	if sq == SquareInvalid {
		return false
	}
	return b.occupancy[color.Index()]&(uint64(1)<<uint(sq)) != 0
}

func (b *boardSoA) empty(sq Square) bool {
	if sq == SquareInvalid {
		return false
	}
	return (b.occupancy[0]|b.occupancy[1])&(uint64(1)<<uint(sq)) == 0
}
