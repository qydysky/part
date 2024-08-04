package tool

// unsigned integer Exp-Golomb-coded syntax element with the left bit first
func ue(d int) (r, move int) {
	for d&0b10000000 == 0b00000000 {
		move += 1
		d = d << 1
	}
	for i := move; i >= 0; i-- {
		r |= d & 0b10000000 >> (7 - i)
		d = d << 1
	}
	return r - 1, 2*move + 1
}

// signed integer Exp-Golomb-coded syntax element with the left bit first
func se(d int) (r, move int) {
	for d&0b10000000 == 0b00000000 {
		move += 1
		d = d << 1
	}
	for i := move; i > 0; i-- {
		r |= d & 0b10000000 >> (8 - i)
		d = d << 1
	}
	if d&0b10000000 == 0b00000000 {
		r = -r
	}
	return r, 2*move + 1
}
