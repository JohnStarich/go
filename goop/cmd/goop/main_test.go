package main

import (
	"bytes"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
)

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
	assert.Equal(t, `Incorrect Usage: flag provided but not defined: -not-a-flag

NAME:
   goop - A new cli application

USAGE:
   goop [global options] command [command options] [arguments...]

COMMANDS:
   info     
   install  
   rm       

GLOBAL OPTIONS:
   --help, -h  show help (default: false)
`, osOut.(*bytes.Buffer).String())
}

func TestExitCode(t *testing.T) {
	t.Parallel()
	err := cli.Exit("failed", 2)
	assert.Equal(t, 2, exitCode(err))
}
