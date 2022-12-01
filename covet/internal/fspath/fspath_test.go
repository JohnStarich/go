package fspath

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/mem"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func noSlashes(s string) string {
	return strings.ReplaceAll(s, separator, ">")
}

func TestCommonBase(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		a, b   string
		expect string
	}{
		{
			a:      "a",
			b:      "a",
			expect: "a",
		},
		{
			a:      "a",
			b:      "b",
			expect: ".",
		},
		{
			a:      "a/b",
			b:      "a/c",
			expect: "a",
		},
		{
			a:      "a/b/c",
			b:      "a/b/d",
			expect: "a/b",
		},
	} {
		tc := tc // enable parallel sub-tests
		t.Run(fmt.Sprintf("%s --> %s", noSlashes(tc.a), noSlashes(tc.b)), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expect, CommonBase(tc.a, tc.b))
		})
	}
}

func TestRel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		basePath   string
		targetPath string
		expectPath string
		expectErr  string
	}{
		{
			basePath:   "a",
			targetPath: "a",
			expectPath: ".",
		},
		{
			basePath:   "a",
			targetPath: "b",
			expectPath: "../b",
		},
		{
			basePath:   "a",
			targetPath: "a/b",
			expectPath: "b",
		},
		{
			basePath:   "a/b",
			targetPath: "a",
			expectPath: "..",
		},
		{
			basePath:   "a/b/c",
			targetPath: "a/b/c",
			expectPath: ".",
		},
		{
			basePath:   "a/b",
			targetPath: "a/b/c",
			expectPath: "c",
		},
		{
			basePath:   "a/b/c",
			targetPath: "a/b",
			expectPath: "..",
		},
		{
			basePath:   "a/b/c",
			targetPath: "a/b/d",
			expectPath: "../d",
		},
	} {
		description := fmt.Sprintf("%s --> %s", noSlashes(tc.basePath), noSlashes(tc.targetPath))
		tc := tc // enable parallel sub-tests
		t.Run(description, func(t *testing.T) {
			t.Parallel()
			p, err := Rel(tc.basePath, tc.targetPath)
			if tc.expectErr != "" {
				assert.EqualError(t, err, tc.expectErr)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.expectPath, p)
		})
	}
}

func TestWorkingDirectoryFS(t *testing.T) {
	t.Parallel()
	t.Run("default fs for current OS", func(t *testing.T) {
		t.Parallel()
		fs, err := WorkingDirectoryFS()
		assert.NoError(t, err)
		assert.NotNil(t, fs)
	})

	fs1, err := mem.NewFS()
	require.NoError(t, err)
	fs2, err := mem.NewFS()
	require.NoError(t, err)
	for _, tc := range []struct {
		description         string
		fs                  hackpadfs.FS
		goos                string
		workingDirectory    string
		workingDirectoryErr error
		volumeName          string
		subVolume           hackpadfs.FS
		subVolumeErr        error
		expectFS            hackpadfs.FS
		expectErr           string
	}{
		{
			description:  "non-windows OS",
			goos:         "not-windows",
			fs:           fs1,
			subVolumeErr: errors.New("some error"),
			expectFS:     fs1,
		},
		{
			description:      "windows OS",
			goos:             "windows",
			fs:               fs1,
			workingDirectory: "foo",
			volumeName:       "bar",
			subVolume:        fs2,
			expectFS:         fs2,
			expectErr:        "",
		},
		{
			description:         "windows working dir error",
			goos:                "windows",
			workingDirectoryErr: errors.New("some error"),
			expectErr:           "some error",
		},
		{
			description:  "windows sub volume error",
			goos:         "windows",
			subVolumeErr: errors.New("some error"),
			expectErr:    "some error",
		},
	} {
		tc := tc // enable parallel sub-tests
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			getWorkingDirectory := func() (string, error) {
				return tc.workingDirectory, tc.workingDirectoryErr
			}
			subVolume := func(path string) (hackpadfs.FS, error) {
				assert.Equal(t, tc.volumeName, path)
				return tc.subVolume, tc.subVolumeErr
			}
			volumeName := func(path string) string {
				assert.Equal(t, tc.workingDirectory, path)
				return tc.volumeName
			}
			fs, err := workingDirectoryFS(
				tc.fs,
				tc.goos,
				getWorkingDirectory,
				subVolume,
				volumeName,
			)
			if tc.expectErr != "" {
				assert.EqualError(t, err, tc.expectErr)
				return
			}
			require.NoError(t, err)
			assert.Same(t, tc.expectFS, fs)
		})
	}
}
