package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCoverageStatus(t *testing.T) {
	for f, status := range map[float64]coverageStatus{
		-10: coverageError,
		0.0: coverageError,
		0.1: coverageError,
		0.5: coverageWarning,
		0.7: coverageOK,
		0.8: coverageGood,
		0.9: coverageExcellent,
		1.0: coverageExcellent,
		10:  coverageExcellent,
	} {
		t.Run(fmt.Sprint(f, status), func(t *testing.T) {
			assert.Equal(t, status, newCoverageStatus(f))
		})
	}
}

func TestCoverageStatusWorkflowCommand(t *testing.T) {
	for status, command := range map[coverageStatus]string{
		coverageExcellent:  "notice",
		coverageGood:       "notice",
		coverageOK:         "warning",
		coverageWarning:    "warning",
		coverageError:      "error",
		coverageStatus(-1): "error",
	} {
		t.Run(fmt.Sprint(status, command), func(t *testing.T) {
			assert.Equal(t, command, status.WorkflowCommand())
		})
	}
}

func TestCoverageStatusEmoji(t *testing.T) {
	for status, emoji := range map[coverageStatus]string{
		coverageExcellent:  "🟢",
		coverageGood:       "🟢",
		coverageOK:         "🟡",
		coverageWarning:    "🟠",
		coverageError:      "🔴",
		coverageStatus(-1): "🔴",
	} {
		t.Run(fmt.Sprint(status, emoji), func(t *testing.T) {
			assert.Equal(t, emoji, status.Emoji())
		})
	}
}
