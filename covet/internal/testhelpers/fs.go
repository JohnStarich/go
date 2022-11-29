// Package testhelpers contains file system generators for use in tests.
package testhelpers

import (
	goos "os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/mem"
	"github.com/hack-pad/hackpadfs/mount"
	"github.com/hack-pad/hackpadfs/os"
	"github.com/stretchr/testify/require"
)

const testDirPermission = 0700

// FromOSToFS returns the FS and FS path for the given OS path.
// If using Windows, the os.FS used will target the osPath's volume (e.g. C:\) before converting.
func FromOSToFS(t *testing.T, osPath string) (*os.FS, string) {
	fs := os.NewFS()
	if runtime.GOOS == "windows" {
		volFS, err := fs.SubVolume(filepath.VolumeName(osPath))
		require.NoError(t, err)
		fs = volFS.(*os.FS)
	}
	fsPath, err := fs.FromOSPath(osPath)
	require.NoError(t, err)
	return fs, fsPath
}

// OSFSWithTemp returns an os.FS instance with 1) the current module's directory and 2) a temporary directory mounted inside.
// Returns the FS paths to both mounts.
func OSFSWithTemp(t *testing.T) (fs hackpadfs.FS, workingDir, tempDir string) {
	t.Helper()

	memFS, err := mem.NewFS()
	require.NoError(t, err)
	mountFS, err := mount.NewFS(memFS)
	require.NoError(t, err)

	wd, err := goos.Getwd()
	require.NoError(t, err)
	workingDirFS, workingDirSubPath := FromOSToFS(t, wd)
	workingDirSubFS, err := workingDirFS.Sub(workingDirSubPath)
	require.NoError(t, err)

	workingDir = "work"
	require.NoError(t, memFS.Mkdir(workingDir, testDirPermission))
	require.NoError(t, mountFS.AddMount(workingDir, workingDirSubFS))

	tempDir = "tmp"
	require.NoError(t, memFS.Mkdir(tempDir, testDirPermission))
	return mountFS, workingDir, tempDir
}

// FSWithFiles returns an FS with the given files contents generated inside it.
// The contents are trimmed and a newline appended for convenient comparisons.
func FSWithFiles(t *testing.T, files map[string]string) hackpadfs.FS {
	t.Helper()
	fs, err := mem.NewFS()
	require.NoError(t, err)
	for name, contents := range files {
		require.NoError(t, fs.MkdirAll(path.Dir(name), testDirPermission))
		f, err := hackpadfs.Create(fs, name)
		require.NoError(t, err)
		_, err = hackpadfs.WriteFile(f, []byte(strings.TrimSpace(contents)+"\n"))
		require.NoError(t, err)
	}
	return fs
}
