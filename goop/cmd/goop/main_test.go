package main

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
)

func init() {
	osExiter = func(code int) {
		panic(errors.Errorf("exited with code: %d", code))
	}
}

func TestMain(t *testing.T) {
	t.Parallel()
	assert.PanicsWithError(t, "exited with code: 1", func() {
		main()
	})
}

func TestExitCode(t *testing.T) {
	t.Parallel()
	err := cli.Exit("failed", 2)
	assert.Equal(t, 2, exitCode(err))
}
