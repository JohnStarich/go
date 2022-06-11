package cmd

import (
	"os"
	"sync"
	"testing"

	"github.com/pkg/errors"
)

const (
	// ExitCodeInvalidUsage is an exit code indicating invalid command line arguments were provided
	ExitCodeInvalidUsage = 2
)

// nolint:gochecknoglobals // Enables os.Exit() to be swapped out in tests for a slightly safer variant. Required for sane test results in older Go versions.
var (
	exiter          = os.Exit
	setupExiterOnce sync.Once
)

// Exit runs os.Exit(). If SetupTestExiter() has been called, it panics instead.
func Exit(code int) {
	exiter(code)
}

// SetupTestExiter changes the exiter to panic instead of exiting
func SetupTestExiter(t *testing.T) {
	// require testing.T to ensure this is a test and not real code
	t.Log("Setting up exiter")
	setupExiterOnce.Do(func() {
		exiter = func(code int) {
			panic(errors.Errorf("Attempted to exit with exit code %d", code))
		}
	})
}
