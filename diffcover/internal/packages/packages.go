package packages

import (
	"go/build"
	"io"
	"path"
	"path/filepath"
	"strings"

	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/mem"
	"github.com/hack-pad/hackpadfs/mount"
	"github.com/pkg/errors"
	"golang.org/x/mod/modfile"
)

type Options struct {
	// GoRoot sets a custom GOROOT on the current build context.
	// Defaults to no root. Paths are relative to the 'fs'.
	GoRoot string
	// GoPath appends custom GOPATH elements on the current build context.
	// Defaults to including the current Go Module in a pseudo-GOPATH directory. Paths are relative to the 'fs'.
	GoPath string
}

// FilePath finds a package file's relative path inside the 'workingDirectory' of 'fs'.
//
// Pass a file package pattern like "github.com/org/mymodule/myfile.go" or "./mymodule/myfile.go".
// If you would like to include more sources for the current Go build context, customize them with 'options'.
func FilePath(fs hackpadfs.FS, workingDirectory, filePattern string, options Options) (pkgFile string, err error) {
	memFS, err := mem.NewFS()
	panicIfErr(err)
	mountFS, err := mount.NewFS(memFS)
	panicIfErr(err)

	const (
		dirPerm = 0700
		workDir = "work"
		tempDir = "tmp"
	)
	var sErr error
	setErr(memFS.Mkdir(workDir, dirPerm), &sErr)
	setErr(mountFS.AddMount(workDir, fs), &sErr)
	workingDirectory = path.Join(workDir, workingDirectory)
	setErr(memFS.Mkdir(tempDir, dirPerm), &sErr)
	options.GoPath = makePathList([]string{tempDir}, filepath.SplitList(options.GoPath))

	moduleName, moduleDir, err := findModule(mountFS, workingDirectory)
	setErr(err, &sErr)
	trimDir := workingDirectory
	if moduleName != "" {
		memFS, err := mem.NewFS()
		panicIfErr(err)
		setErr(mountFS.AddMount(tempDir, memFS), &sErr)
		moduleGoPath := path.Join("src", moduleName)
		setErr(memFS.MkdirAll(moduleGoPath, 0700), &sErr)
		moduleFS, err := hackpadfs.Sub(mountFS, moduleDir)
		if err != nil {
			return "", err
		}
		setErr(mountFS.AddMount(path.Join(tempDir, moduleGoPath), moduleFS), &sErr) // "symlink" original fs inside the new GOPATH-like directory
		workDirSubPath := strings.TrimPrefix(workingDirectory, moduleDir)
		workDirSubPath = strings.TrimPrefix(workDirSubPath, "/")
		trimDir = path.Join(tempDir, moduleGoPath, workDirSubPath)
	}

	packageName, coverageFile := path.Split(filePattern)
	ctx := newFSBuildContext(mountFS, workingDirectory, options)
	pkg, err := ctx.Import(packageName, workingDirectory, build.FindOnly)
	setErr(err, &sErr)
	coverageFile = path.Join(pkg.Dir, coverageFile)
	coverageFile = strings.TrimPrefix(coverageFile, trimDir+"/")
	return coverageFile, sErr
}

func panicIfErr(err error) {
	if err != nil {
		panic(err)
	}
}

func setErr(err error, setErr *error) {
	if err != nil && *setErr == nil {
		*setErr = err
	}
}

func makePathList(elems ...[]string) string {
	var allElems []string
	for _, e := range elems {
		allElems = append(allElems, e...)
	}
	return strings.Join(allElems, string(filepath.ListSeparator))
}

func newFSBuildContext(fs hackpadfs.FS, workingDirectory string, options Options) build.Context {
	ctx := build.Default
	ctx.GOROOT = options.GoRoot
	ctx.GOPATH = options.GoPath
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
	return ctx
}

func findModule(fs hackpadfs.FS, dir string) (moduleName, moduleDir string, err error) {
	for ; dir != "."; dir = path.Dir(dir) {
		var ok bool
		moduleName, moduleDir, ok, err = getModule(fs, dir)
		if ok || err != nil {
			return
		}
	}
	moduleName, moduleDir, _, err = getModule(fs, dir)
	return
}

func getModule(fs hackpadfs.FS, dir string) (moduleName, moduleDir string, ok bool, err error) {
	file := path.Join(dir, "go.mod")
	_, err = hackpadfs.Stat(fs, file)
	if err != nil {
		if errors.Is(err, hackpadfs.ErrNotExist) {
			err = nil
		}
		return "", "", false, err
	}

	contents, err := hackpadfs.ReadFile(fs, file)
	if err != nil {
		return "", "", false, err
	}
	modFile, err := modfile.Parse(file, contents, nil)
	if err != nil {
		return "", "", false, err
	}
	return modFile.Module.Mod.Path, dir, true, nil
}
