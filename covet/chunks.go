package covet

import (
	"bufio"
	"io"

	"github.com/johnstarich/go/covet/internal/span"
)

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
		overlappingSpans = append(overlappingSpans, span.Span{
			Start: max(1, int64(line.LineNumber)-int64(contextLines)),
			End:   int64(line.LineNumber + contextLines + 1), // may be beyond end of file, but can stop line iteration at EOF
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

type DiffChunk struct {
	FirstLine, LastLine uint
	Lines               []string
}

// DiffChunks return diff-like Chunks from a covet.File and the file contents' Reader.
func DiffChunks(file File, fileReader io.Reader) ([]DiffChunk, error) {
	var chunks []DiffChunk
	iter := newLineIterator(fileReader)
	const contextLines = 2
	var lineNumber int64 = 1
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
				if lineNumber == int64(diffLine.LineNumber) {
					op = diffLine.diffOpPrefix()
					diffLineIndex++
				}
			}
			lines[i] = op + lines[i]
			lineNumber++
		}
		chunks = append(chunks, DiffChunk{
			FirstLine: uint(s.Start),
			LastLine:  uint(lineNumber - 1),
			Lines:     lines,
		})
	}
	return chunks, nil
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

type lineIterator struct {
	scanner *bufio.Scanner
}

func newLineIterator(r io.Reader) *lineIterator {
	return &lineIterator{
		scanner: bufio.NewScanner(r),
	}
}

func (l *lineIterator) SkipLines(n int64) error {
	more := true
	for i := int64(0); i < n && more; i++ {
		more = l.scanner.Scan()
	}
	return l.scanner.Err()
}

func (l *lineIterator) NextLines(n int64) ([]string, error) {
	lines := make([]string, 0, n)
	for i := int64(0); i < n; i++ {
		more := l.scanner.Scan()
		if !more {
			return lines, l.scanner.Err()
		}
		lines = append(lines, l.scanner.Text())
	}
	return lines, nil
}
