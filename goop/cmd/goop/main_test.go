package main

import (
	"bytes"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
)

//nolint:gochecknoinits // This minimal init enables testing of main().
func init() {
	osArgs = []string{"goop", "-not-a-flag"}
	osExiter = func(code int) {
		panic(errors.Errorf("exited with code: %d", code))
	}
	osOut = bytes.NewBuffer(nil)
	osErr = bytes.NewBuffer(nil)
}

func TestMain(t *testing.T) {
	t.Parallel()
	assert.PanicsWithError(t, "exited with code: 1", func() {
		main()
	})
	assert.Equal(t, "flag provided but not defined: -not-a-flag\n", osErr.(*bytes.Buffer).String())
	output := osOut.(*bytes.Buffer).String()
	assert.Contains(t, output, "Incorrect Usage: flag provided but not defined: -not-a-flag")
	assert.Contains(t, output, "COMMANDS:")
}

func TestExitCode(t *testing.T) {
	t.Parallel()
	err := cli.Exit("failed", 2)
	assert.Equal(t, 2, exitCode(err))
}
