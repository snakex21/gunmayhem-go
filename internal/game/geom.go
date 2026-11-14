package game

type Rect struct {
	X, Y, W, H float64
}

func (r Rect) ContainsX(x float64) bool {
	return x >= r.X && x <= r.X+r.W
}

func (r Rect) Contains(x, y float64) bool {
	return x >= r.X && x <= r.X+r.W && y >= r.Y && y <= r.Y+r.H
}
