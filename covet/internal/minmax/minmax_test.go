package minmax

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMin(t *testing.T) {
	t.Parallel()
	assert.EqualValues(t, 1, Min(1, 2))
	assert.EqualValues(t, 1, Min(2, 1))
	assert.EqualValues(t, 0, Min(0, 0))
}

func TestMax(t *testing.T) {
	t.Parallel()
	assert.EqualValues(t, 2, Max(1, 2))
	assert.EqualValues(t, 2, Max(2, 1))
	assert.EqualValues(t, 0, Max(0, 0))
}
