package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCoverageStatus(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		f      float64
		status coverageStatus
	}{
		{-10, coverageError},
		{0.0, coverageError},
		{0.1, coverageError},
		{0.5, coverageWarning},
		{0.7, coverageOK},
		{0.8, coverageGood},
		{0.9, coverageExcellent},
		{1.0, coverageExcellent},
		{10, coverageExcellent},
	} {
		tc := tc // enable parallel sub-tests
		t.Run(fmt.Sprint(tc.f, tc.status), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.status, newCoverageStatus(tc.f))
		})
	}
}

func TestCoverageStatusWorkflowCommand(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		status  coverageStatus
		command string
	}{
		{coverageExcellent, "notice"},
		{coverageGood, "notice"},
		{coverageOK, "warning"},
		{coverageWarning, "warning"},
		{coverageError, "error"},
		{coverageStatus(-1), "error"},
	} {
		tc := tc // enable parallel sub-tests
		t.Run(fmt.Sprint(tc.status, tc.command), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.command, tc.status.WorkflowCommand())
		})
	}
}

func TestCoverageStatusEmoji(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		status coverageStatus
		emoji  string
	}{
		{coverageExcellent, "🟢"},
		{coverageGood, "🟢"},
		{coverageOK, "🟡"},
		{coverageWarning, "🟠"},
		{coverageError, "🔴"},
		{coverageStatus(-1), "🔴"},
	} {
		tc := tc // enable parallel sub-tests
		t.Run(fmt.Sprint(tc.status, tc.emoji), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.emoji, tc.status.Emoji())
		})
	}
}
