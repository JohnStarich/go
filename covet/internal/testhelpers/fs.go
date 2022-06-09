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
	"github.com/johnstarich/go/covet/internal/fspath"
	"github.com/stretchr/testify/require"
)

// OSFSWithTemp returns an os.FS instance with 1) the current module's directory and 2) a temporary directory mounted inside.
// Returns the FS paths to both mounts.
func OSFSWithTemp(t *testing.T, relPathToModuleDir string) (_ hackpadfs.FS, workingDirectory, tempDirectory string) {
	t.Helper()

	wd, err := goos.Getwd()
	require.NoError(t, err)
	wdPath := path.Join(fspath.ToFSPath(wd), relPathToModuleDir)
	osFS, err := os.NewFS().Sub(wdPath)
	require.NoError(t, err)

	memFS, err := mem.NewFS()
	require.NoError(t, err)
	fs, err := mount.NewFS(memFS)
	require.NoError(t, err)

	workingDirectory = "work"
	require.NoError(t, hackpadfs.MkdirAll(fs, workingDirectory, 0700))
	require.NoError(t, fs.AddMount(workingDirectory, osFS))

	tempDirectory = "tmp"
	require.NoError(t, hackpadfs.MkdirAll(fs, tempDirectory, 0700))
	return fs, workingDirectory, tempDirectory
}

// FSWithFiles returns an FS with the given files contents generated inside it.
// The contents are trimmed and a newline appended for convenient comparisons.
func FSWithFiles(t *testing.T, files map[string]string) hackpadfs.FS {
	t.Helper()
	fs, err := mem.NewFS()
	require.NoError(t, err)
	for name, contents := range files {
		require.NoError(t, fs.MkdirAll(path.Dir(name), 0700))
		f, err := hackpadfs.Create(fs, name)
		require.NoError(t, err)
		_, err = hackpadfs.WriteFile(f, []byte(strings.TrimSpace(contents)+"\n"))
		require.NoError(t, err)
	}
	return fs
}
