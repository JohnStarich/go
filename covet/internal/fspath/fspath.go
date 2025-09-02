// Package fspath contains FS path manipulation tools, much like the standard library's "path/filepath" package.
package fspath

import (
	goOS "os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/os"
	"github.com/pkg/errors"
)

const separator = "/"

// CommonBase returns the common base path between a and b.
// Returns "." if there are no common path elements.
func CommonBase(a, b string) string {
	a = path.Clean(a)
	b = path.Clean(b)
	i := 0
	for ; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			break
		}
	}
	return path.Clean(a[:i])
}

// Rel returns the relative FS path from basePath to targetPath.
// Similar to filepath.Rel without including OS-dependent behavior.
func Rel(basePath, targetPath string) (string, error) {
	basePath = path.Clean(basePath)
	targetPath = path.Clean(targetPath)

	common := CommonBase(basePath, targetPath)
	if common == "" {
		return "", errors.New("could not make relative path between basePath and targetPath")
	}
	base := basePath
	base = strings.TrimPrefix(base, common)
	base = strings.TrimPrefix(base, separator)
	target := targetPath
	target = strings.TrimPrefix(target, common)
	target = strings.TrimPrefix(target, separator)

	switch {
	case base == "" && target == "":
		return ".", nil
	case base == "":
		return target, nil
	default:
		p := strings.Repeat("../", strings.Count(base, separator)+1)
		return path.Join(p, target), nil
	}
}

// WorkingDirectoryFS returns the os.FS for the current working directory.
// If using Windows, the os.FS used will target the osPath's volume (e.g. C:\).
func WorkingDirectoryFS() (*os.FS, error) {
	fs := os.NewFS()
	volFS, err := workingDirectoryFS(fs, runtime.GOOS, goOS.Getwd, fs.SubVolume, filepath.VolumeName)
	if err == nil {
		fs = volFS.(*os.FS)
	}
	return fs, err
}

//nolint:ireturn // Returns an interface intentionally
func workingDirectoryFS(
	fs hackpadfs.FS,
	goos string,
	getWorkingDirectory func() (string, error),
	subVolume func(string) (hackpadfs.FS, error),
	volumeName func(string) string,
) (hackpadfs.FS, error) {
	if goos != "windows" {
		return fs, nil
	}
	wd, err := getWorkingDirectory()
	if err != nil {
		return nil, err
	}
	return subVolume(volumeName(wd))
}
