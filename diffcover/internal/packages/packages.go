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
	"github.com/johnstarich/go/diffcover/internal/fspath"
	"github.com/pkg/errors"
	"golang.org/x/mod/modfile"
)

func FilePath(fs hackpadfs.FS, workingDirectory, coverageEntry string) (pkgFile string, err error) {
	memFS, err := mem.NewFS()
	if err != nil {
		return "", err
	}
	const (
		dirPerm = 0700
		workDir = "work"
		tempDir = "tmp"
	)
	mountFS, err := mount.NewFS(memFS)
	if err != nil {
		return "", err
	}
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

	ctx, trimDir, err := newFSBuildContext(mountFS, workingDirectory, tempDir)
	if err != nil {
		return "", err
	}
	packageName, coverageFile := path.Split(coverageEntry)
	pkg, err := ctx.Import(packageName, workingDirectory, build.FindOnly)
	if err != nil {
		return "", err
	}
	coverageFile = path.Join(pkg.Dir, coverageFile)
	coverageFile = strings.TrimPrefix(coverageFile, trimDir+"/")
	return coverageFile, nil
}

func newFSBuildContext(mountFS *mount.FS, workingDirectory, tmpDir string) (_ *build.Context, trimDir string, err error) {
	defer func() { err = errors.WithStack(err) }()

	ctx := build.Default
	ctx.GOROOT = fspath.ToFSPath(ctx.GOROOT)
	ctx.GOPATH = fspath.ToFSPathList(ctx.GOPATH)
	trimDir = workingDirectory

	moduleName, moduleDir, err := getModule(mountFS, workingDirectory)
	if err != nil {
		return nil, "", err
	}
	if moduleName != "" {
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
		moduleFS, err := hackpadfs.Sub(mountFS, moduleDir)
		if err != nil {
			return nil, "", err
		}
		err = mountFS.AddMount(path.Join(tmpDir, moduleGoPath), moduleFS) // "symlink" original fs inside the new GOPATH-like directory
		if err != nil {
			return nil, "", err
		}
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
		info, err := hackpadfs.Stat(mountFS, path)
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
		dirEntries, err := hackpadfs.ReadDir(mountFS, dir)
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
		return mountFS.Open(path)
	}
	return &ctx, trimDir, nil
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
