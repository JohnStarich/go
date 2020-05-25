package main

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/johnstarich/go/gopages/cmd"
	"github.com/johnstarich/go/gopages/internal/flags"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(t *testing.T) {
	cmd.SetupTestExiter(t)
	assert.Panics(t, main)
}

func TestMainArgs(t *testing.T) {
	cmd.SetupTestExiter(t)
	tmp, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(tmp)

	for _, tc := range []struct {
		description string
		runnerErr   error
		wdErr       error
		args        []string
		expectErr   string
	}{
		{
			description: "bad flag usage",
			args:        []string{"-not-a-flag"},
			expectErr:   "Attempted to exit with exit code 2",
		},
		{
			description: "request usage",
			args:        []string{"-help"},
		},
		{
			description: "getwd error",
			wdErr:       errors.New("some error"),
			expectErr:   "Failed to get current directory: some error",
		},
		{
			description: "runner failed",
			runnerErr:   errors.New("some error"),
			expectErr:   "Attempted to exit with exit code 1",
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			runner := func(string, flags.Args) error {
				return tc.runnerErr
			}
			getWD := func() (string, error) {
				return tmp, tc.wdErr
			}

			runTest := func() {
				mainArgs(runner, getWD, tc.args...)
			}
			if tc.expectErr != "" {
				assert.PanicsWithError(t, tc.expectErr, runTest)
				return
			}
			assert.NotPanics(t, runTest)
		})
	}
}
