package covet

import (
	"bytes"
	"errors"
	"io"
	goos "os"
	"path"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/os"
	"github.com/johnstarich/go/covet/internal/fspath"
	"github.com/johnstarich/go/covet/internal/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		description       string
		diff              string
		diffReader        io.Reader
		coverage          string
		expectDiffCovered float64
		expectErr         string
	}{
		{
			description: "half covered",
			diff: `
diff --git a/covet.go b/covet.go
index 0000000..1111111 100644
--- a/covet.go
+++ b/covet.go
@@ -0,0 +1,2 @@
+added 1
+added 2
`,
			coverage: `
mode: atomic
github.com/johnstarich/go/covet/covet.go:1.1,1.7 1 1
github.com/johnstarich/go/covet/covet.go:2.1,2.7 1 0
`,
			expectDiffCovered: 0.5,
		},
		{
			description: "no coverage hits",
			diff: `
diff --git a/covet.go b/covet.go
index 0000000..1111111 100644
--- a/covet.go
+++ b/covet.go
@@ -0,0 +1,2 @@
+added 1
+added 2
`,
			coverage: `
mode: atomic
github.com/johnstarich/go/covet/covet.go:1.1,1.7 1 0
github.com/johnstarich/go/covet/covet.go:2.1,2.7 1 0
`,
			expectDiffCovered: 0,
		},
		{
			description: "bad diff reader",
			diffReader:  iotest.ErrReader(errors.New("some error")),
			coverage: `
mode: atomic
github.com/johnstarich/go/covet/covet.go:1.1,1.7 1 1
github.com/johnstarich/go/covet/covet.go:2.1,2.7 1 0
`,
			expectErr: "some error",
		},
		{
			description: "malformed coverage file",
			diff: `
diff --git a/covet.go b/covet.go
index 0000000..1111111 100644
--- a/covet.go
+++ b/covet.go
@@ -0,0 +1,2 @@
+added 1
+added 2
`,
			coverage: `
foo
`,
			expectErr: "bad mode line: foo",
		},
	} {
		tc := tc // enable parallel sub-tests
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			if tc.diffReader == nil {
				tc.diffReader = strings.NewReader(strings.TrimSpace(tc.diff))
			}
			fs, wd, tmpDir := testhelpers.OSFSWithTemp(t, "")

			coverFile := path.Join(tmpDir, "cover.out")
			{
				f, err := hackpadfs.OpenFile(fs, coverFile, hackpadfs.FlagWriteOnly|hackpadfs.FlagCreate, 0600)
				require.NoError(t, err)
				_, err = hackpadfs.WriteFile(f, []byte(strings.TrimSpace(tc.coverage)))
				require.NoError(t, err)
				require.NoError(t, f.Close())
			}

			covet, err := Parse(Options{
				FS:                fs,
				Diff:              tc.diffReader,
				DiffBaseDir:       wd,
				GoCoveragePath:    coverFile,
				GoCoverageBaseDir: wd,
			})
			if tc.expectErr != "" {
				assert.EqualError(t, err, "covet: "+tc.expectErr)
				return
			}
			require.NoError(t, err)

			files := covet.DiffCoverageFiles()
			assert.NotEmpty(t, files)

			covered := covet.DiffCovered()
			assert.Equal(t, tc.expectDiffCovered, covered)
		})
	}
}

func TestParseInvalidOptions(t *testing.T) {
	t.Parallel()
	wd, err := goos.Getwd()
	require.NoError(t, err)
	var (
		workingDirectory = fspath.ToFSPath(wd)
		baseDir          = path.Join(workingDirectory, "testdata")
		coverFile        = path.Join(baseDir, "add2.out")
	)
	for _, tc := range []struct {
		description string
		options     Options
		expectErr   string
	}{
		{
			description: "invalid diff base dir",
			options: Options{
				DiffBaseDir: "/os-path/not/ok",
			},
			expectErr: "invalid diff base directory FS path: /os-path/not/ok",
		},
		{
			description: "invalid go coverage file path",
			options: Options{
				DiffBaseDir:    ".",
				GoCoveragePath: "/os-path/not/ok",
			},
			expectErr: "invalid coverage FS path: /os-path/not/ok",
		},
		{
			description: "fs is optional",
			options: Options{
				FS:             nil,
				Diff:           bytes.NewReader(nil),
				DiffBaseDir:    baseDir,
				GoCoveragePath: coverFile,
			},
		},
		{
			description: "go coverage base dir is optional",
			options: Options{
				FS:                os.NewFS(),
				Diff:              bytes.NewReader(nil),
				DiffBaseDir:       ".",
				GoCoveragePath:    coverFile,
				GoCoverageBaseDir: "",
			},
		},
		{
			description: "invalid go coverage base dir",
			options: Options{
				FS:                os.NewFS(),
				Diff:              bytes.NewReader(nil),
				DiffBaseDir:       ".",
				GoCoveragePath:    coverFile,
				GoCoverageBaseDir: "/os-path/not/ok",
			},
			expectErr: "invalid coverage base directory FS path: /os-path/not/ok",
		},
		{
			description: "diff is required",
			options: Options{
				FS:                os.NewFS(),
				Diff:              nil,
				DiffBaseDir:       ".",
				GoCoveragePath:    coverFile,
				GoCoverageBaseDir: baseDir,
			},
			expectErr: "diff reader must not be nil",
		},
	} {
		tc := tc // enable parallel sub-tests
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			_, err := Parse(tc.options)
			if tc.expectErr != "" {
				assert.EqualError(t, err, "covet: "+tc.expectErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}
