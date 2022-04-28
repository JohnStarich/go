package diffcover

import (
	"strings"
	"testing"

	"github.com/johnstarich/go/diffcover/internal/span"
	"github.com/stretchr/testify/assert"
)

func TestFileFindContextSpans(t *testing.T) {
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
		t.Run(tc.description, func(t *testing.T) {
			spans := tc.file.findContextSpans(tc.contextLines)
			assert.Equal(t, tc.expectSpans, spans)
		})
	}
}

func TestDiffChunks(t *testing.T) {
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
	assert.Equal(t, []Chunk{
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
