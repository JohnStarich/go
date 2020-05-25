package flags

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	args, output, err := Parse("-help")
	assert.Equal(t, Args{
		OutputPath: "dist",
	}, args)
	assert.NotEmpty(t, output)
	assert.Equal(t, flag.ErrHelp, err)
}
