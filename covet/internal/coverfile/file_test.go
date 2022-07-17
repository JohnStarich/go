package coverfile

import (
	"strings"
	"testing"

	"github.com/johnstarich/go/covet/internal/span"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileFindContextSpans(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		description  string
		file         File
		contextLines uint
		expectSpans  []span.Span
	}{
		{
			description: "no lines",
		},
		{
			description: "1 line, 0 context",
			file: File{Lines: []Line{
				{LineNumber: 1},
			}},
			expectSpans: []span.Span{
				{Start: 1, End: 2},
			},
		},
		{
			description: "2 adjacent lines, 0 context",
			file: File{Lines: []Line{
				{LineNumber: 1},
				{LineNumber: 2},
			}},
			expectSpans: []span.Span{
				{Start: 1, End: 2},
				{Start: 2, End: 3},
			},
		},
		{
			description: "2 disjoint lines, 0 context",
			file: File{Lines: []Line{
				{LineNumber: 1},
				{LineNumber: 3},
			}},
			expectSpans: []span.Span{
				{Start: 1, End: 2},
				{Start: 3, End: 4},
			},
		},
		{
			description: "2 disjoint lines, 2 context",
			file: File{Lines: []Line{
				{LineNumber: 1},
				{LineNumber: 3},
			}},
			contextLines: 2,
			expectSpans: []span.Span{
				{Start: 1, End: 6},
			},
		},
		{
			description: "2 adjacent lines, 1 disjoint line, 0 context",
			file: File{Lines: []Line{
				{LineNumber: 1},
				{LineNumber: 2},
				{LineNumber: 4},
			}},
			expectSpans: []span.Span{
				{Start: 1, End: 2},
				{Start: 2, End: 3},
				{Start: 4, End: 5},
			},
		},
		{
			description: "2 adjacent lines, 2 disjoint lines, 1 context",
			file: File{Lines: []Line{
				{LineNumber: 1},
				{LineNumber: 2},
				{LineNumber: 4},
				{LineNumber: 7},
			}},
			contextLines: 1,
			expectSpans: []span.Span{
				{Start: 1, End: 6},
				{Start: 6, End: 9},
			},
		},
	} {
		tc := tc // enable parallel sub-tests
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			spans := tc.file.findContextSpans(tc.contextLines)
			assert.Equal(t, tc.expectSpans, spans)
		})
	}
}

func TestDiffChunks(t *testing.T) {
	t.Parallel()
	diff := strings.TrimSpace(`
1: first line
2:
3:
4:
5:
6:
7:
8: last line
`)
	file := File{
		Lines: []Line{
			{LineNumber: 5, Covered: true},
			{LineNumber: 8, Covered: false},
		},
	}
	chunks, err := DiffChunks(file, strings.NewReader(diff))
	assert.NoError(t, err)
	assert.Equal(t, []DiffChunk{
		{
			FirstLine: 3,
			LastLine:  8,
			Lines: []string{
				" 3:",
				" 4:",
				"+5:",
				" 6:",
				" 7:",
				"-8: last line",
			},
		},
	}, chunks)
}

func TestLineIterator(t *testing.T) {
	t.Parallel()
	const threeLines = `line 1
line 2
line 3
`
	t.Run("skip and next", func(t *testing.T) {
		t.Parallel()
		li := newLineIterator(strings.NewReader(threeLines))
		require.NoError(t, li.SkipLines(1))
		lines, err := li.NextLines(1)
		require.NoError(t, err)
		assert.Equal(t, []string{"line 2"}, lines)
	})

	t.Run("skip past end", func(t *testing.T) {
		t.Parallel()
		li := newLineIterator(strings.NewReader(threeLines))
		assert.NoError(t, li.SkipLines(4))
	})

	t.Run("read past end", func(t *testing.T) {
		t.Parallel()
		li := newLineIterator(strings.NewReader(threeLines))
		lines, err := li.NextLines(4)
		assert.NoError(t, err)
		assert.Equal(t, []string{
			"line 1",
			"line 2",
			"line 3",
		}, lines)
	})
}
