package pipe

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChain(t *testing.T) {
	results := make([]bool, 3)
	c := Chain(
		OpFunc(func() error {
			results[0] = true
			return nil
		}),
		OpFunc(func() error {
			results[1] = true
			return errors.New("some error")
		}),
		OpFunc(func() error {
			results[2] = true
			return nil
		}),
	)
	assert.EqualError(t, c.Do(), "some error")
	assert.Equal(t, []bool{true, true, false}, results)
}

func TestChainFuncs(t *testing.T) {
	t.Run("error early return", func(t *testing.T) {
		results := make([]bool, 3)
		c := ChainFuncs(
			func() error {
				results[0] = true
				return nil
			},
			func() error {
				results[1] = true
				return errors.New("some error")
			},
			func() error {
				results[2] = true
				return nil
			},
		)
		assert.EqualError(t, c.Do(), "some error")
		assert.Equal(t, []bool{true, true, false}, results)
	})

	t.Run("no errors", func(t *testing.T) {
		results := make([]bool, 3)
		c := ChainFuncs(
			func() error {
				results[0] = true
				return nil
			},
			func() error {
				results[1] = true
				return nil
			},
			func() error {
				results[2] = true
				return nil
			},
		)
		assert.NoError(t, c.Do())
		assert.Equal(t, []bool{true, true, true}, results)
	})
}

func TestErrIf(t *testing.T) {
	assert.EqualError(t, ErrIf(true, errors.New("some error")), "some error")
	assert.NoError(t, ErrIf(true, nil))
	assert.NoError(t, ErrIf(false, errors.New("some error")))
	assert.NoError(t, ErrIf(false, nil))
}
