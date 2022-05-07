package diffcover

import (
	"fmt"
	"io"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/hack-pad/hackpadfs"
	"github.com/johnstarich/go/diffcover/internal/fspath"
	"github.com/johnstarich/go/diffcover/internal/span"
	"github.com/pkg/errors"
	"golang.org/x/tools/cover"
	"golang.org/x/tools/go/packages"
)

// DiffCoverage generates reports for a diff and coverage combination
type DiffCoverage struct {
	addedLines,
	coveredLines,
	uncoveredLines map[string][]span.Span
}

// Options contains parse options
type Options struct {
	// FS is the file system to use when reading the coverage file and generating path names. Only os.NewFS() from hackpadfs is compatible for now, see below note.
	//
	// NOTE: Coverage parsing still uses the native file system to load package file names. In a future release, hopefully this can use the FS instead.
	FS hackpadfs.FS
	// Diff is a reader with patch or diff formatted contents
	Diff io.Reader
	// DiffBaseDir is the FS path to the repo's root directory
	DiffBaseDir string
	// GoCoverage is the FS path to a Go coverage file
	GoCoveragePath string
}

// Parse reads and parses both a diff file and Go coverage file, then returns a DiffCoverage instance to render reports
func Parse(options Options) (_ *DiffCoverage, err error) {
	defer func() { err = errors.WithStack(err) }()
	if !hackpadfs.ValidPath(options.DiffBaseDir) {
		return nil, errors.Errorf("invalid diff base dir FS path: %s", options.DiffBaseDir)
	}
	if !hackpadfs.ValidPath(options.GoCoveragePath) {
		return nil, errors.Errorf("invalid coverage FS path: %s", options.GoCoveragePath)
	}

	diffFiles, _, err := gitdiff.Parse(options.Diff)
	if err != nil {
		return nil, err
	}

	coverageFile, err := options.FS.Open(options.GoCoveragePath)
	if err != nil {
		return nil, err
	}
	coverageFiles, err := cover.ParseProfilesFromReader(coverageFile)
	coverageFile.Close()
	if err != nil {
		return nil, err
	}

	diffcov := &DiffCoverage{
		addedLines:     make(map[string][]span.Span),
		coveredLines:   make(map[string][]span.Span),
		uncoveredLines: make(map[string][]span.Span),
	}
	if err := diffcov.addDiff(diffFiles); err != nil {
		return nil, err
	}
	if err := diffcov.addCoverage(options.GoCoveragePath, options.DiffBaseDir, coverageFiles); err != nil {
		return nil, err
	}
	return diffcov, nil
}

func (c *DiffCoverage) addDiff(diffFiles []*gitdiff.File) error {
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

func (c *DiffCoverage) addCoverage(coveragePath, baseDir string, coverageFiles []*cover.Profile) error {
	coverageDir := path.Dir(coveragePath)
	toCoveragePath := func(coverageEntry string) string {
		switch {
		case filepath.IsAbs(coverageEntry):
			return filepath.Dir(coverageEntry)
		case strings.HasPrefix(coverageEntry, "./"):
			filePath := path.Join(coverageDir, coverageEntry)
			fileAbsPath := fspath.FromFSPath(filePath)
			dir := filepath.Dir(fileAbsPath)
			return dir
		default: // not a file path
			return path.Dir(coverageEntry)
		}
	}
	pkgCache := make(map[string]*packages.Package)
	getPackage := func(fileName string) (*packages.Package, error) {
		packagePath := toCoveragePath(fileName)
		if pkg, ok := pkgCache[packagePath]; ok {
			return pkg, nil
		}
		var dir string
		if filepath.IsAbs(packagePath) {
			dir = packagePath
		}
		pkgs, err := packages.Load(&packages.Config{
			Mode: packages.NeedFiles,
			Dir:  dir,
		}, packagePath)
		if err != nil {
			return nil, fmt.Errorf("package not found: %w", err)
		}
		if len(pkgs) == 0 {
			return nil, fmt.Errorf("no package found for pattern: %s", fileName)
		}
		if len(pkgs[0].Errors) > 0 {
			return nil, errors.Wrapf(pkgs[0].Errors[0], "loading package %s failed", fileName)
		}
		pkgCache[dir] = pkgs[0]
		return pkgs[0], nil
	}

	for _, file := range coverageFiles {
		for _, block := range file.Blocks {
			pkg, err := getPackage(file.FileName)
			if err != nil {
				return err
			}
			var pkgFile string
			for _, fullPath := range pkg.GoFiles {
				if filepath.Base(fullPath) == path.Base(file.FileName) {
					pkgFile = fullPath
					break
				}
			}
			if pkgFile == "" {
				return fmt.Errorf("package %s does not contain file %s", path.Dir(file.FileName), path.Base(file.FileName))
			}
			coverageFile, err := filepath.Rel(fspath.FromFSPath(baseDir), pkgFile)
			if err != nil {
				return err
			}
			coverageFile = filepath.ToSlash(coverageFile)
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

func (c *DiffCoverage) coveredAndUncovered() (fileNames map[string]bool, coveredDiff, uncoveredDiff map[string][]span.Span) {
	fileNames = make(map[string]bool)
	coveredDiff = make(map[string][]span.Span)
	uncoveredDiff = make(map[string][]span.Span)
	for file := range c.addedLines {
		for _, added := range c.addedLines[file] {
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

// Covered returns the percentage of covered lines in the diff.
func (c *DiffCoverage) Covered() float64 {
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

func (c *DiffCoverage) Files() []File {
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
