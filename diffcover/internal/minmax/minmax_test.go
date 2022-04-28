package minmax

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMin(t *testing.T) {
	assert.EqualValues(t, 0, MinInt64(0, 1))
	assert.EqualValues(t, 0, MinInt64(0, 0))
	assert.EqualValues(t, -1, MinInt64(-1, 0))
}

func TestMax(t *testing.T) {
	assert.EqualValues(t, 1, MaxInt64(1, 0))
	assert.EqualValues(t, 0, MaxInt64(0, 0))
	assert.EqualValues(t, 0, MaxInt64(-1, 0))
}
