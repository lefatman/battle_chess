package shared

func DirectionOf(from, to Square) Direction {
	fr, ff := from.Rank(), from.File()
	tr, tf := to.Rank(), to.File()
	dr := tr - fr
	df := tf - ff

	nr := normalize(dr)
	nf := normalize(df)

	switch {
	case nr == -1 && nf == 0:
		return DirN
	case nr == -1 && nf == 1:
		return DirNE
	case nr == 0 && nf == 1:
		return DirE
	case nr == 1 && nf == 1:
		return DirSE
	case nr == 1 && nf == 0:
		return DirS
	case nr == 1 && nf == -1:
		return DirSW
	case nr == 0 && nf == -1:
		return DirW
	case nr == -1 && nf == -1:
		return DirNW
	default:
		return DirNone
	}
}

func normalize(v int) int {
	if v > 0 {
		return 1
	}
	if v < 0 {
		return -1
	}
	return 0
}

func SameFileNeighborhood(from, to Square, d int) bool {
	ff := from.File()
	tf := to.File()
	switch d {
	case -1, 7, -9:
		return tf == ff-1
	case 1, -7, 9:
		return tf == ff+1
	default:
		return true
	}
}

func Line(from, to Square) []Square {
	dr := to.Rank() - from.Rank()
	df := to.File() - from.File()
	stepR := normalize(dr)
	stepF := normalize(df)

	aligned := false
	switch {
	case dr == 0 && df != 0:
		stepR = 0
		aligned = true
	case df == 0 && dr != 0:
		stepF = 0
		aligned = true
	case abs(dr) == abs(df) && dr != 0:
		aligned = true
	}

	if !aligned {
		return nil
	}

	distance := max(abs(dr), abs(df)) - 1
	if distance <= 0 {
		return nil
	}

	squares := make([]Square, 0, distance)
	rank := from.Rank()
	file := from.File()
	for i := 0; i < distance; i++ {
		rank += stepR
		file += stepF
		sq, ok := SquareFromCoords(rank, file)
		if !ok {
			return nil
		}
		squares = append(squares, sq)
	}
	return squares
}

func SquareFromCoords(rank, file int) (Square, bool) {
	if rank < 0 || rank > 7 || file < 0 || file > 7 {
		return 0, false
	}
	return Square(rank*8 + file), true
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
