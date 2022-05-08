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
	if err != nil {
		return "", err
	}
	mountFS, err := mount.NewFS(memFS)
	if err != nil {
		return "", err
	}

	const (
		dirPerm = 0700
		workDir = "work"
		tempDir = "tmp"
	)
	if err := memFS.Mkdir(workDir, dirPerm); err != nil {
		return "", err
	}
	if err := mountFS.AddMount(workDir, fs); err != nil {
		return "", err
	}
	workingDirectory = path.Join(workDir, workingDirectory)
	if err := memFS.Mkdir(tempDir, dirPerm); err != nil {
		return "", err
	}
	options.GoPath = makePathList([]string{tempDir}, filepath.SplitList(options.GoPath))

	moduleName, moduleDir, err := getModule(mountFS, workingDirectory)
	if err != nil {
		return "", err
	}
	trimDir := workingDirectory
	if moduleName != "" {
		memFS, err := mem.NewFS()
		if err != nil {
			return "", err
		}
		err = mountFS.AddMount(tempDir, memFS)
		if err != nil {
			return "", err
		}
		moduleGoPath := path.Join("src", moduleName)
		err = memFS.MkdirAll(moduleGoPath, 0700)
		if err != nil {
			return "", err
		}
		moduleFS, err := hackpadfs.Sub(mountFS, moduleDir)
		if err != nil {
			return "", err
		}
		err = mountFS.AddMount(path.Join(tempDir, moduleGoPath), moduleFS) // "symlink" original fs inside the new GOPATH-like directory
		if err != nil {
			return "", err
		}
		workDirSubPath := strings.TrimPrefix(workingDirectory, moduleDir+"/")
		trimDir = path.Join(tempDir, moduleGoPath, workDirSubPath)
	}

	packageName, coverageFile := path.Split(filePattern)
	ctx := newFSBuildContext(mountFS, workingDirectory, options)
	pkg, err := ctx.Import(packageName, workingDirectory, build.FindOnly)
	if err != nil {
		return "", err
	}
	coverageFile = path.Join(pkg.Dir, coverageFile)
	coverageFile = strings.TrimPrefix(coverageFile, trimDir+"/")
	return coverageFile, nil
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
