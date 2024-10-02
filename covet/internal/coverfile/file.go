// Package coverfile parses Covet reports and formats files in the diff
package coverfile

import (
	"bufio"
	"io"

	"github.com/johnstarich/go/covet/internal/minmax"
	"github.com/johnstarich/go/covet/internal/span"
)

// File represents a file parsed in a Covet report. Includes the file name and which lines are covered.
//
// NOTE: This struct is in package 'coverfile' to expose in the main 'covet' package while breaking import cycles.
// Avoid exporting any methods or unnecessary data fields.
type File struct {
	Name      string
	Covered   uint
	Uncovered uint
	Lines     []Line
}

func (f File) findContextSpans(contextLines uint) []span.Span {
	if len(f.Lines) == 0 {
		return nil
	}

	var overlappingSpans []span.Span
	for _, line := range f.Lines {
		var start uint = 1
		if contextLines < line.LineNumber { // prevent uint underflow
			start = line.LineNumber - contextLines
		}
		overlappingSpans = append(overlappingSpans, span.Span{
			Start: minmax.Max(1, start),
			End:   line.LineNumber + contextLines + 1, // may be beyond end of file, but can stop line iteration at EOF
		})
	}
	spans := []span.Span{overlappingSpans[0]}
	for _, s := range overlappingSpans {
		lastSpan := &spans[len(spans)-1]
		newSpan, merged := lastSpan.Merge(s)
		if merged {
			*lastSpan = newSpan
		} else {
			spans = append(spans, s)
		}
	}
	return spans
}

// Line represents a line in a file parsed in a Covet report
//
// NOTE: This struct is in package 'coverfile' to expose in the main 'covet' package while breaking import cycles.
// Avoid exporting any methods or unnecessary data fields.
type Line struct {
	Covered    bool
	LineNumber uint
}

const noOpPrefix = " "

func (l Line) diffOpPrefix() string {
	if l.Covered {
		return "+"
	}
	return "-"
}

// DiffChunk is a chunk of lines parsed from a Covet report.
// Includes the associated beginning and end line numbers of the chunk and the diff-like lines of text from the file.
//
// NOTE: This struct is in package 'coverfile' to expose in the main 'covet' package while breaking import cycles.
// Avoid exporting any methods or unnecessary data fields.
type DiffChunk struct {
	FirstLine, LastLine uint
	Lines               []string
}

// DiffChunks return diff-like Chunks from a covet.File and the file contents' Reader.
func DiffChunks(file File, fileReader io.Reader) ([]DiffChunk, error) {
	var chunks []DiffChunk
	iter := newLineIterator(fileReader)
	const contextLines = 2
	var lineNumber uint = 1
	diffLineIndex := 0
	for _, s := range file.findContextSpans(contextLines) {
		if lineNumber < s.Start {
			err := iter.SkipLines(s.Start - lineNumber)
			if err != nil {
				return nil, err
			}
			lineNumber = s.Start
		}
		lines, err := iter.NextLines(s.Len())
		if err != nil {
			return nil, err
		}
		for i := range lines {
			op := noOpPrefix
			if diffLineIndex < len(file.Lines) {
				diffLine := file.Lines[diffLineIndex]
				if lineNumber == diffLine.LineNumber {
					op = diffLine.diffOpPrefix()
					diffLineIndex++
				}
			}
			lines[i] = op + lines[i]
			lineNumber++
		}
		chunks = append(chunks, DiffChunk{
			FirstLine: s.Start,
			LastLine:  lineNumber - 1,
			Lines:     lines,
		})
	}
	return chunks, nil
}

type lineIterator struct {
	scanner *bufio.Scanner
}

func newLineIterator(r io.Reader) *lineIterator {
	return &lineIterator{
		scanner: bufio.NewScanner(r),
	}
}

func (l *lineIterator) SkipLines(n uint) error {
	more := true
	for i := uint(0); i < n && more; i++ {
		more = l.scanner.Scan()
	}
	return l.scanner.Err()
}

func (l *lineIterator) NextLines(n uint) ([]string, error) {
	lines := make([]string, 0, n)
	for i := uint(0); i < n; i++ {
		more := l.scanner.Scan()
		if !more {
			return lines, l.scanner.Err()
		}
		lines = append(lines, l.scanner.Text())
	}
	return lines, nil
}
