package covet

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPriorityUncoveredFiles(t *testing.T) {
	t.Parallel()
	t.Run("sort and filter just enough files", func(t *testing.T) {
		t.Parallel()
		files := []File{
			{Name: "foo", Covered: 2, Uncovered: 0},
			{Name: "bar", Covered: 1, Uncovered: 2},
			{Name: "baz", Covered: 1, Uncovered: 2},
			{Name: "biff", Covered: 0, Uncovered: 2},
		}
		reportable := findReportableUncoveredFiles(files, 0.75, 0.4)
		assert.Equal(t, []File{
			{Name: "bar", Covered: 1, Uncovered: 2},
			{Name: "baz", Covered: 1, Uncovered: 2},
			{Name: "biff", Covered: 0, Uncovered: 2},
		}, reportable)
	})

	t.Run("include more small files if the biggest chunks are not close enough to target", func(t *testing.T) {
		t.Parallel()
		files := []File{
			{Name: "foo", Covered: 0, Uncovered: 1},
			{Name: "bar", Covered: 0, Uncovered: 1},
			{Name: "baz", Covered: 0, Uncovered: 1},
			{Name: "biff", Covered: 0, Uncovered: 7},
		}
		reportable := findReportableUncoveredFiles(files, 0.8, 0)
		assert.Equal(t, []File{
			{Name: "biff", Covered: 0, Uncovered: 7},
			{Name: "bar", Covered: 0, Uncovered: 1},
			{Name: "baz", Covered: 0, Uncovered: 1},
			{Name: "foo", Covered: 0, Uncovered: 1},
		}, reportable)
	})
}
