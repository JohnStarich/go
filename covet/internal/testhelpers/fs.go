package testhelpers

import (
	goos "os"
	"path"
	"strings"
	"testing"

	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/mem"
	"github.com/hack-pad/hackpadfs/mount"
	"github.com/hack-pad/hackpadfs/os"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testDirPermission = 0700

// OSFSWithTemp returns an os.FS instance with 1) the current module's directory and 2) a temporary directory mounted inside.
// Returns the FS paths to both mounts.
func OSFSWithTemp(t *testing.T, relPathToModuleDir string) (_ hackpadfs.FS, workingDirectory, tempDirectory string) {
	t.Helper()

	wd, err := goos.Getwd()
	require.NoError(t, err)
	osFS := os.NewFS()
	wdFSPath, err := osFS.FromOSPath(wd)
	require.NoError(t, err)
	wdPath := path.Join(wdFSPath, relPathToModuleDir)
	if fs, err := osFS.Sub(wdPath); assert.NoError(t, err) {
		osFS = fs.(*os.FS)
	}

	memFS, err := mem.NewFS()
	require.NoError(t, err)
	fs, err := mount.NewFS(memFS)
	require.NoError(t, err)

	workingDirectory = "work"
	require.NoError(t, hackpadfs.MkdirAll(fs, workingDirectory, testDirPermission))
	require.NoError(t, fs.AddMount(workingDirectory, osFS))

	tempDirectory = "tmp"
	require.NoError(t, hackpadfs.MkdirAll(fs, tempDirectory, testDirPermission))
	return fs, workingDirectory, tempDirectory
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
