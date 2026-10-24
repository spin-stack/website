package consts

import "time"

const (
	Size   = 1 << 30
	Window = 90 * 24 * time.Hour
	Half   = Size / 2
	Pair   = (Size + 2) - 2
	Other  = time.Now
)
