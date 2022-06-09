package minmax

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMin(t *testing.T) {
	t.Parallel()
	testMin(t, "int64", func(a, b int) int {
		return int(MinInt64(int64(a), int64(b)))
	})
	testMin(t, "uint", func(a, b int) int {
		return int(MinUint(uint(a), uint(b)))
	})
	testMin(t, "int", MinInt)
}

func TestMax(t *testing.T) {
	t.Parallel()
	testMax(t, "int64", func(a, b int) int {
		return int(MaxInt64(int64(a), int64(b)))
	})
	testMax(t, "uint", func(a, b int) int {
		return int(MaxUint(uint(a), uint(b)))
	})
	testMax(t, "int", MaxInt)
}

func testMin(t *testing.T, description string, min func(a, b int) int) {
	t.Helper()
	t.Run(description, func(t *testing.T) {
		assert.EqualValues(t, 1, min(1, 2))
		assert.EqualValues(t, 1, min(2, 1))
		assert.EqualValues(t, 0, min(0, 0))
	})
}

func testMax(t *testing.T, description string, max func(a, b int) int) {
	t.Helper()
	t.Run(description, func(t *testing.T) {
		assert.EqualValues(t, 2, max(1, 2))
		assert.EqualValues(t, 2, max(2, 1))
		assert.EqualValues(t, 0, max(0, 0))
	})
}
