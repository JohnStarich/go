package module

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackage(t *testing.T) {
	t.Parallel()
	modulePackage, err := Package("../..")
	require.NoError(t, err)
	assert.Equal(t, "github.com/therve/go/gopages", modulePackage)
}
