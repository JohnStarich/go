package diffcover

import (
	"go/build"
	"io"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/mem"
	"github.com/hack-pad/hackpadfs/mount"
	"github.com/johnstarich/go/diffcover/internal/fspath"
	"github.com/johnstarich/go/diffcover/internal/span"
	"github.com/pkg/errors"
	"golang.org/x/mod/modfile"
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
	// FS is the file system to read files, Go package information, and more.
	// If you're not sure which FS to use, pass hackpadfs's os.NewFS().
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
	memFS, err := mem.NewFS()
	if err != nil {
		return nil, err
	}
	const (
		dirPerm = 0700
		workDir = "work"
		tempDir = "tmp"
	)
	fs, err := mount.NewFS(memFS)
	if err != nil {
		return nil, err
	}
	if err := memFS.Mkdir(workDir, dirPerm); err != nil {
		return nil, err
	}
	if err := fs.AddMount(workDir, options.FS); err != nil {
		return nil, err
	}
	if err := memFS.Mkdir(tempDir, dirPerm); err != nil {
		return nil, err
	}

	if !hackpadfs.ValidPath(options.DiffBaseDir) {
		return nil, errors.Errorf("invalid diff base dir FS path: %s", options.DiffBaseDir)
	}
	options.DiffBaseDir = path.Join(workDir, options.DiffBaseDir)
	if !hackpadfs.ValidPath(options.GoCoveragePath) {
		return nil, errors.Errorf("invalid coverage FS path: %s", options.GoCoveragePath)
	}
	options.GoCoveragePath = path.Join(workDir, options.GoCoveragePath)

	diffFiles, _, err := gitdiff.Parse(options.Diff)
	if err != nil {
		return nil, err
	}

	coverageFile, err := fs.Open(options.GoCoveragePath)
	if err != nil {
		return nil, err
	}
	defer coverageFile.Close()
	coverageFiles, err := cover.ParseProfilesFromReader(coverageFile)
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
	if err := diffcov.addCoverage(fs, options.GoCoveragePath, options.DiffBaseDir, tempDir, coverageFiles); err != nil {
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

func (c *DiffCoverage) addCoverage(fs hackpadfs.FS, coveragePath, baseDir, tmpDir string, coverageFiles []*cover.Profile) error {
	for _, file := range coverageFiles {
		for _, block := range file.Blocks {
			pkgFile, trimDir, err := getBuildPackagePath(fs, baseDir, tmpDir, file.FileName)
			if err != nil {
				return err
			}
			coverageFile := pkgFile
			coverageFile = strings.TrimPrefix(coverageFile, trimDir+"/")
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

func getModule(fs hackpadfs.FS, dir string) (moduleName, moduleDir string, err error) {
	for ; dir != "."; dir = path.Dir(dir) {
		file := path.Join(dir, "go.mod")
		_, err = hackpadfs.Stat(fs, file)
		if err != nil && !errors.Is(err, hackpadfs.ErrNotExist) {
			return "", "", err
		}
		if err == nil {
			contents, err := hackpadfs.ReadFile(fs, file)
			if err != nil {
				return "", "", err
			}
			modFile, err := modfile.Parse(file, contents, nil)
			if err != nil {
				return "", "", err
			}
			return modFile.Module.Mod.Path, dir, nil
		}
	}
	return "", "", nil
}

func newFSBuildContext(fs hackpadfs.FS, workingDirectory, tmpDir string) (_ *build.Context, trimDir string, err error) {
	defer func() { err = errors.WithStack(err) }()

	ctx := build.Default
	ctx.GOROOT = fspath.ToFSPath(ctx.GOROOT)
	ctx.GOPATH = fspath.ToFSPathList(ctx.GOPATH)
	trimDir = workingDirectory

	moduleName, moduleDir, err := getModule(fs, workingDirectory)
	if err != nil {
		return nil, "", err
	}
	if moduleName != "" {
		mountFS, err := mount.NewFS(fs)
		if err != nil {
			return nil, "", err
		}
		memFS, err := mem.NewFS()
		if err != nil {
			return nil, "", err
		}
		err = mountFS.AddMount(tmpDir, memFS)
		if err != nil {
			return nil, "", err
		}
		moduleGoPath := path.Join("src", moduleName)
		err = memFS.MkdirAll(moduleGoPath, 0700)
		if err != nil {
			return nil, "", err
		}
		moduleFS, err := hackpadfs.Sub(fs, moduleDir)
		if err != nil {
			return nil, "", err
		}
		err = mountFS.AddMount(path.Join(tmpDir, moduleGoPath), moduleFS) // "symlink" original fs inside the new GOPATH-like directory
		if err != nil {
			return nil, "", err
		}
		fs = mountFS // reassign build context fs to "symlinked" version
		ctx.GOPATH += string(filepath.ListSeparator) + tmpDir
		workDirSubPath := strings.TrimPrefix(workingDirectory, moduleDir+"/")
		trimDir = path.Join(tmpDir, moduleGoPath, workDirSubPath)
	}

	ctx.JoinPath = path.Join
	ctx.SplitPathList = filepath.SplitList
	ctx.IsAbsPath = func(path string) bool {
		return !build.IsLocalImport(path)
	}
	ctx.IsDir = func(path string) bool {
		info, err := hackpadfs.Stat(fs, path)
		return err == nil && info.IsDir()
	}
	ctx.HasSubdir = func(root, dir string) (rel string, ok bool) {
		// TODO add EvalSymlinks support to hackpadfs
		const sep = "/"
		root = path.Clean(root)
		if !strings.HasSuffix(root, sep) {
			root += sep
		}
		dir = path.Clean(dir)
		if !strings.HasPrefix(dir, root) {
			return "", false
		}
		return dir[len(root):], true
	}
	ctx.ReadDir = func(dir string) ([]hackpadfs.FileInfo, error) {
		dirEntries, err := hackpadfs.ReadDir(fs, dir)
		if err != nil {
			return nil, err
		}
		var infos []hackpadfs.FileInfo
		for _, dirEntry := range dirEntries {
			info, err := dirEntry.Info()
			if err != nil {
				return nil, err
			}
			infos = append(infos, info)
		}
		return infos, nil
	}
	ctx.OpenFile = func(path string) (io.ReadCloser, error) {
		return fs.Open(path)
	}
	return &ctx, trimDir, nil
}

func getBuildPackagePath(fs hackpadfs.FS, workingDirectory, tmpDir, coverageEntry string) (pkgFile, trimDir string, err error) {
	ctx, trimDir, err := newFSBuildContext(fs, workingDirectory, tmpDir)
	if err != nil {
		return "", "", err
	}
	packageName, coverageFile := path.Split(coverageEntry)
	pkg, err := ctx.Import(packageName, workingDirectory, build.FindOnly)
	if err != nil {
		return "", "", err
	}
	return path.Join(pkg.Dir, coverageFile), trimDir, nil
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
