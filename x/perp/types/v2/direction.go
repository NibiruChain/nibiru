package v2

func (dir Direction) ToMultiplier() int64 {
	var dirMult int64
	switch dir {
	case Direction_LONG, Direction_DIRECTION_UNSPECIFIED:
		dirMult = 1
	case Direction_SHORT:
		dirMult = -1
	}
	return dirMult
}
