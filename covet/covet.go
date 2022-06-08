package covet

import (
	"io"
	"path"
	"sort"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/os"
	"github.com/johnstarich/go/covet/internal/fspath"
	"github.com/johnstarich/go/covet/internal/packages"
	"github.com/johnstarich/go/covet/internal/span"
	"github.com/pkg/errors"
	"golang.org/x/tools/cover"
)

// Covet generates reports for a diff and coverage combination
type Covet struct {
	options Options
	addedLines,
	coveredLines,
	uncoveredLines map[string][]span.Span
}

// Options contains parse options
type Options struct {
	// FS is the file system to read files, Go package information, and more.
	// Defaults to hackpadfs's os.NewFS().
	FS hackpadfs.FS
	// Diff is a reader with patch or diff formatted contents
	Diff io.Reader
	// DiffBaseDir is the FS path to the repo's root directory
	DiffBaseDir string
	// GoCoverage is the FS path to a Go coverage file
	GoCoveragePath string
	// GoCoverageBaseDir is the FS path to the coverage file's module. Defaults to the coverage file's directory.
	GoCoverageBaseDir string
}

// Parse reads and parses both a diff file and Go coverage file, then returns a Covet instance to render reports
func Parse(options Options) (covet *Covet, err error) {
	defer func() { err = errors.Wrap(err, "covet") }()
	if !hackpadfs.ValidPath(options.DiffBaseDir) {
		return nil, errors.Errorf("invalid diff base directory FS path: %s", options.DiffBaseDir)
	}
	if !hackpadfs.ValidPath(options.GoCoveragePath) {
		return nil, errors.Errorf("invalid coverage FS path: %s", options.GoCoveragePath)
	}
	if options.GoCoverageBaseDir == "" {
		options.GoCoverageBaseDir = path.Dir(options.GoCoveragePath)
	}
	if !hackpadfs.ValidPath(options.GoCoverageBaseDir) {
		return nil, errors.Errorf("invalid coverage base directory FS path: %s", options.GoCoverageBaseDir)
	}
	if options.FS == nil {
		options.FS = os.NewFS()
	}
	if options.Diff == nil {
		return nil, errors.New("diff reader must not be nil")
	}

	diffFiles, _, err := gitdiff.Parse(options.Diff)
	if err != nil {
		return nil, err
	}

	coverageFile, err := options.FS.Open(options.GoCoveragePath)
	if err != nil {
		return nil, err
	}
	defer coverageFile.Close()
	coverageFiles, err := cover.ParseProfilesFromReader(coverageFile)
	if err != nil {
		return nil, err
	}

	covet = &Covet{
		options:        options,
		addedLines:     make(map[string][]span.Span),
		coveredLines:   make(map[string][]span.Span),
		uncoveredLines: make(map[string][]span.Span),
	}
	_, err = covet.coverageToDiffRel()
	if err != nil {
		return nil, err
	}

	if err := covet.addDiff(diffFiles); err != nil {
		return nil, err
	}
	if err := covet.addCoverage(options.FS, options.GoCoveragePath, options.GoCoverageBaseDir, coverageFiles); err != nil {
		return nil, err
	}
	return covet, nil
}

func (c *Covet) addDiff(diffFiles []*gitdiff.File) error {
	for _, file := range diffFiles {
		spans := findDiffAddSpans(file.TextFragments)
		c.addedLines[file.NewName] = append(c.addedLines[file.NewName], spans...)
	}
	return nil
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

func (c *Covet) addCoverage(fs hackpadfs.FS, coveragePath, baseDir string, coverageFiles []*cover.Profile) error {
	for _, file := range coverageFiles {
		for _, block := range file.Blocks {
			coverageFile, err := packages.FilePath(fs, baseDir, file.FileName, packages.Options{})
			if err != nil {
				return err
			}
			if block.Count > 0 {
				c.coveredLines[coverageFile] = append(c.coveredLines[coverageFile], span.Span{
					Start: int64(block.StartLine),
					End:   int64(block.EndLine + 1),
				})
			} else {
				c.uncoveredLines[coverageFile] = append(c.uncoveredLines[coverageFile], span.Span{
					Start: int64(block.StartLine),
					End:   int64(block.EndLine + 1),
				})
			}
		}
	}
	return nil
}

func (c *Covet) coverageToDiffRel() (string, error) {
	return fspath.Rel(c.options.DiffBaseDir, c.options.GoCoverageBaseDir)
}

func (c *Covet) coveredAndUncovered() (fileNames map[string]bool, coveredDiff, uncoveredDiff map[string][]span.Span) {
	covToDiffRel, _ := c.coverageToDiffRel() // ignore error since it's checked during setup
	fileNames = make(map[string]bool)
	coveredDiff = make(map[string][]span.Span)
	uncoveredDiff = make(map[string][]span.Span)
	for file := range c.addedLines {
		for _, added := range c.addedLines[file] {
			file, err := fspath.Rel(covToDiffRel, file)
			if err != nil {
				panic(err)
			}
			for _, covered := range c.coveredLines[file] {
				if intersection, ok := added.Intersection(covered); ok {
					fileNames[file] = true
					coveredDiff[file] = append(coveredDiff[file], intersection)
				}
			}
			for _, uncovered := range c.uncoveredLines[file] {
				if intersection, ok := added.Intersection(uncovered); ok {
					fileNames[file] = true
					uncoveredDiff[file] = append(uncoveredDiff[file], intersection)
				}
			}
		}
	}
	return
}

// DiffCovered returns the percentage of covered lines in the diff.
func (c *Covet) DiffCovered() float64 {
	_, coveredSpans, uncoveredSpans := c.coveredAndUncovered()
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

func (c *Covet) DiffCoverageFiles() []File {
	var coveredFiles []File
	fileNames, coveredSpans, uncoveredSpans := c.coveredAndUncovered()
	for file := range fileNames {
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
