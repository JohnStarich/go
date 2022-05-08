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
	"github.com/johnstarich/go/diffcover/internal/fspath"
	"github.com/stretchr/testify/require"
)

func OSFSWithTemp(t *testing.T) (_ hackpadfs.FS, workingDirectory, tempDirectory string) {
	t.Helper()

	fs, err := mount.NewFS(os.NewFS())
	require.NoError(t, err)

	memFS, err := mem.NewFS()
	require.NoError(t, err)
	tmpDir := fspath.ToFSPath(t.TempDir())
	require.NoError(t, fs.AddMount(tmpDir, memFS))

	wd, err := goos.Getwd()
	require.NoError(t, err)
	wd = fspath.ToFSPath(wd)

	return fs, wd, tmpDir
}

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
