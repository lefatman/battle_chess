// path: chessTest/internal/game/bitboard.go
package game

// Bitboard represents a 64-bit set of squares.
type Bitboard uint64

func BB(s Square) Bitboard { return 1 << s }

func (b Bitboard) Empty() bool { return b == 0 }

func (b Bitboard) PopLSB() (Square, Bitboard) {
	lsb := b & -b
	if lsb == 0 {
		return 0, 0
	}
	idx := Square(bitScan(lsb))
	return idx, b ^ Bitboard(lsb)
}

func (b Bitboard) Has(s Square) bool { return b&(1<<s) != 0 }

func (b Bitboard) Add(s Square) Bitboard { return b | (1 << s) }

func (b Bitboard) Remove(s Square) Bitboard { return b &^ (1 << s) }

func (b Bitboard) Iter(fn func(Square)) {
	bb := b
	for bb != 0 {
		sq, rest := bb.PopLSB()
		fn(sq)
		bb = rest
	}
}

func bitScan(x Bitboard) int {
	const debruijn = 0x03f79d71b4cb0a89
	index := ((uint64(x) & -uint64(x)) * debruijn) >> 58
	return debruijnIndex[index]
}

var debruijnIndex = [64]int{
	0, 1, 48, 2, 57, 49, 28, 3,
	61, 58, 50, 42, 38, 29, 17, 4,
	62, 55, 59, 36, 53, 51, 43, 22,
	45, 39, 33, 30, 24, 18, 12, 5,
	63, 47, 56, 27, 60, 41, 37, 16,
	54, 35, 52, 21, 44, 32, 23, 11,
	46, 26, 40, 15, 34, 20, 31, 10,
	25, 14, 19, 9, 13, 8, 7, 6,
}
