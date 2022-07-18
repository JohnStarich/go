package covet

import (
	"io"

	"github.com/johnstarich/go/covet/internal/coverfile"
)

// File represents a file parsed in a Covet report. Includes the file name and which lines are covered.
type File = coverfile.File

// Line represents a line in a file parsed in a Covet report
type Line = coverfile.Line

// DiffChunk is a chunk of lines parsed from a Covet report.
// Includes the associated beginning and end line numbers of the chunk and the diff-like lines of text from the file.
type DiffChunk = coverfile.DiffChunk

// DiffChunks return diff-like Chunks from a covet.File and the file contents' Reader.
func DiffChunks(file File, fileReader io.Reader) ([]DiffChunk, error) {
	return coverfile.DiffChunks(file, fileReader)
}
