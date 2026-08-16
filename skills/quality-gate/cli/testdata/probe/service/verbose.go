package service

// Rank scores a card.
//
// The scoring is described here at length so the block runs past the budget a
// function doc is allowed to spend, which is the point of the fixture: it has
// to be long enough to trip CMT-01 and no rule other than CMT-01, so the
// assertion stays about the budget and nothing else creeps in alongside it,
// which needs one line more than the budget a function doc is given.
func Rank(a, b, c, d, e, f, g int) int {
	total := 0
	for i := 0; i < a; i++ {
		if i%2 == 0 {
			for j := 0; j < b; j++ {
				if j%3 == 0 {
					for k := 0; k < c; k++ {
						if k > d {
							total += e + f + g
						}
					}
				}
			}
		}
	}
	return total
}
