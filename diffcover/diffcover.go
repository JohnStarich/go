package diffcover

import (
	"io"
	"path/filepath"
	"sort"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/johnstarich/go/diffcover/internal/span"
	"golang.org/x/tools/cover"
)

// DiffCoverage generates reports for a diff and coverage combination
type DiffCoverage struct {
	addedLines,
	coveredLines,
	uncoveredLines map[string][]span.Span
}

// Options contains parse options
type Options struct {
	// Diff is a reader with patch or diff formatted contents
	Diff io.Reader
	// GoCoverage is a reader with Go coverage formatted contents
	GoCoverage io.Reader
}

// Parse reads and parses both a diff file and Go coverage file, then returns a DiffCoverage instance to render reports
func Parse(options Options) (*DiffCoverage, error) {
	diffFiles, _, err := gitdiff.Parse(options.Diff)
	if err != nil {
		return nil, err
	}

	coverageFiles, err := cover.ParseProfilesFromReader(options.GoCoverage)
	if err != nil {
		return nil, err
	}

	diffcov := &DiffCoverage{
		addedLines:     make(map[string][]span.Span),
		coveredLines:   make(map[string][]span.Span),
		uncoveredLines: make(map[string][]span.Span),
	}
	diffcov.addDiff(diffFiles)
	diffcov.addCoverage(coverageFiles)

	return diffcov, nil
}

func (c *DiffCoverage) addDiff(diffFiles []*gitdiff.File) {
	for _, file := range diffFiles {
		spans := findDiffAddSpans(file.TextFragments)
		c.addedLines[file.NewName] = append(c.addedLines[file.NewName], spans...)
	}
}

func findDiffAddSpans(fragments []*gitdiff.TextFragment) []span.Span {
	var spans []span.Span
	for _, fragment := range fragments {
		lineNumber := fragment.NewPosition
		for _, line := range fragment.Lines {
			if line.Op == gitdiff.OpAdd {
				if len(spans) == 0 || spans[len(spans)-1].End < lineNumber {
					spans = append(spans, span.Span{Start: lineNumber, End: lineNumber + 1})
				} else {
					spans[len(spans)-1].End++
				}
			}
			if line.New() {
				lineNumber++
			}
		}
	}
	return spans
}

func (c *DiffCoverage) addCoverage(coverageFiles []*cover.Profile) {
	for _, file := range coverageFiles {
		for _, block := range file.Blocks {
			coverageFile := filepath.Clean(file.FileName)
			if block.Count > 0 {
				c.coveredLines[coverageFile] = append(c.coveredLines[coverageFile], span.Span{
					Start: int64(block.StartLine),
					End:   int64(block.EndLine),
				})
			} else {
				c.uncoveredLines[coverageFile] = append(c.uncoveredLines[coverageFile], span.Span{
					Start: int64(block.StartLine),
					End:   int64(block.EndLine),
				})
			}
		}
	}
}

func (c *DiffCoverage) coveredAndUncovered() (coveredDiff, uncoveredDiff map[string][]span.Span) {
	coveredDiff = make(map[string][]span.Span)
	uncoveredDiff = make(map[string][]span.Span)
	for file := range c.addedLines {
		for _, added := range c.addedLines[file] {
			for _, covered := range c.coveredLines[file] {
				if intersection, ok := added.Intersection(covered); ok {
					coveredDiff[file] = append(coveredDiff[file], intersection)
				}
			}
			for _, uncovered := range c.uncoveredLines[file] {
				if intersection, ok := added.Intersection(uncovered); ok {
					uncoveredDiff[file] = append(uncoveredDiff[file], intersection)
				}
			}
		}
	}
	return
}

// Covered returns the percentage of covered lines in the diff.
func (c *DiffCoverage) Covered() float64 {
	coveredSpans, uncoveredSpans := c.coveredAndUncovered()
	var coveredTotal, uncoveredTotal float64
	for _, spans := range coveredSpans {
		for _, s := range spans {
			coveredTotal += float64(s.Len())
		}
	}
	for _, spans := range uncoveredSpans {
		for _, s := range spans {
			uncoveredTotal += float64(s.Len())
		}
	}
	return coveredTotal / (coveredTotal + uncoveredTotal)
}

func (c *DiffCoverage) Files() []File {
	var coveredFiles []File
	coveredSpans, uncoveredSpans := c.coveredAndUncovered()
	for file := range coveredSpans {
		covered := coveredSpans[file]
		uncovered := uncoveredSpans[file]

		coveredFile := File{Name: file}
		for _, s := range covered {
			for i := s.Start; i < s.End; i++ {
				coveredFile.Lines = append(coveredFile.Lines, Line{
					Covered:    true,
					LineNumber: uint(i),
				})
			}
			coveredFile.Covered += uint(s.Len())
		}
		for _, s := range uncovered {
			for i := s.Start; i < s.End; i++ {
				coveredFile.Lines = append(coveredFile.Lines, Line{
					Covered:    false,
					LineNumber: uint(i),
				})
			}
			coveredFile.Uncovered += uint(s.Len())
		}
		sort.Slice(coveredFile.Lines, func(a, b int) bool {
			return coveredFile.Lines[a].LineNumber < coveredFile.Lines[b].LineNumber
		})
		coveredFiles = append(coveredFiles, coveredFile)
	}
	return coveredFiles
}
