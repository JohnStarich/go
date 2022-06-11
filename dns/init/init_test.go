package init

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInit(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, net.DefaultResolver)
}
