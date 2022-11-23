package main

import (
	"bytes"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

//nolint:gochecknoinits // This minimal init enables testing of main().
func init() {
	osArgs = []string{"goop", "--not-a-flag"}
	osExiter = func(code int) {
		panic(errors.Errorf("exited with code: %d", code))
	}
	osOut = bytes.NewBuffer(nil)
	osErr = bytes.NewBuffer(nil)
}

func TestMain(t *testing.T) {
	t.Parallel()
	assert.PanicsWithError(t, "exited with code: 2", func() {
		main()
	})
	stdout := osOut.(*bytes.Buffer).String()
	stderr := osErr.(*bytes.Buffer).String()
	assert.Contains(t, stdout, "Usage:")
	assert.Equal(t, "unknown flag: --not-a-flag\n", stderr)
}

func TestExitCode(t *testing.T) {
	t.Parallel()
	err := wrapExitCode(errors.New("failed"), 2)
	assert.Equal(t, 2, exitCode(err))
}
