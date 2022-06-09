package minmax

// MinInt64 returns the smallest of a and b
func MinInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// MaxInt64 returns the largest of a and b
func MaxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// MinUint returns the smallest of a and b
func MinUint(a, b uint) uint {
	if a < b {
		return a
	}
	return b
}

// MaxUint returns the largest of a and b
func MaxUint(a, b uint) uint {
	if a > b {
		return a
	}
	return b
}

// MinInt returns the smallest of a and b
func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// MaxInt returns the largest of a and b
func MaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
