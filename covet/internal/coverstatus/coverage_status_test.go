package coverstatus

import (
	"fmt"
	"testing"

	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
)

func TestNewCoverageStatus(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		f      float64
		status Status
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
		t.Run(fmt.Sprint(tc.f, tc.status), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.status, New(tc.f))
		})
	}
}

func TestCoverageStatusWorkflowCommand(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		status  Status
		command string
	}{
		{coverageExcellent, "notice"},
		{coverageGood, "notice"},
		{coverageOK, "warning"},
		{coverageWarning, "warning"},
		{coverageError, "error"},
		{Status(-1), "error"},
	} {
		t.Run(fmt.Sprint(tc.status, tc.command), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.command, tc.status.WorkflowCommand())
		})
	}
}

func TestCoverageStatusEmoji(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		status Status
		emoji  string
	}{
		{coverageExcellent, "🟢"},
		{coverageGood, "🟢"},
		{coverageOK, "🟡"},
		{coverageWarning, "🟠"},
		{coverageError, "🔴"},
		{Status(-1), "🔴"},
	} {
		t.Run(fmt.Sprint(tc.status, tc.emoji), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.emoji, tc.status.Emoji())
		})
	}
}

func TestCoverageStatusColor(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		status Status
		expect *color.Color
	}{
		{coverageExcellent, boldGreen()},
		{coverageGood, green()},
		{coverageOK, yellow()},
		{coverageWarning, red()},
		{coverageError, boldRed()},
		{Status(-1), boldRed()},
	} {
		t.Run(fmt.Sprint(tc.status, tc.expect), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expect, tc.status.color())
		})
	}
}
