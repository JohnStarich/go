package diffcover

import (
	"errors"
	"io"
	"os"
	"path"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/hack-pad/hackpadfs"
	"github.com/johnstarich/go/diffcover/internal/fspath"
	"github.com/johnstarich/go/diffcover/internal/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	for _, tc := range []struct {
		description   string
		diff          string
		diffReader    io.Reader
		coverage      string
		expectCovered float64
		expectErr     string
	}{
		{
			description: "half covered",
			diff: `
diff --git a/diffcover.go b/diffcover.go
index 0000000..1111111 100644
--- a/diffcover.go
+++ b/diffcover.go
@@ -0,0 +1,2 @@
+added 1
+added 2
`,
			coverage: `
mode: atomic
github.com/johnstarich/go/diffcover/diffcover.go:1.1,1.7 1 1
github.com/johnstarich/go/diffcover/diffcover.go:2.1,2.7 1 0
`,
			expectCovered: 0.5,
		},
		{
			description: "no coverage hits",
			diff: `
diff --git a/diffcover.go b/diffcover.go
index 0000000..1111111 100644
--- a/diffcover.go
+++ b/diffcover.go
@@ -0,0 +1,2 @@
+added 1
+added 2
`,
			coverage: `
mode: atomic
github.com/johnstarich/go/diffcover/diffcover.go:1.1,1.7 1 0
github.com/johnstarich/go/diffcover/diffcover.go:2.1,2.7 1 0
`,
			expectCovered: 0,
		},
		{
			description: "bad diff reader",
			diffReader:  iotest.ErrReader(errors.New("some error")),
			coverage: `
mode: atomic
github.com/johnstarich/go/diffcover/diffcover.go:1.1,1.7 1 1
github.com/johnstarich/go/diffcover/diffcover.go:2.1,2.7 1 0
`,
			expectErr: "some error",
		},
		{
			description: "malformed coverage file",
			diff: `
diff --git a/diffcover.go b/diffcover.go
index 0000000..1111111 100644
--- a/diffcover.go
+++ b/diffcover.go
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
		t.Run(tc.description, func(t *testing.T) {
			if tc.diffReader == nil {
				tc.diffReader = strings.NewReader(strings.TrimSpace(tc.diff))
			}
			fs, wd, tmpDir := testhelpers.OSFSWithTemp(t)

			coverFile := path.Join(tmpDir, "cover.out")
			{
				f, err := hackpadfs.OpenFile(fs, coverFile, hackpadfs.FlagWriteOnly|hackpadfs.FlagCreate, 0600)
				require.NoError(t, err)
				_, err = hackpadfs.WriteFile(f, []byte(strings.TrimSpace(tc.coverage)))
				require.NoError(t, err)
				require.NoError(t, f.Close())
			}

			diffcover, err := Parse(Options{
				FS:             fs,
				TempDir:        path.Clean(fspath.ToFSPath(os.TempDir())),
				Diff:           tc.diffReader,
				DiffBaseDir:    wd,
				GoCoveragePath: coverFile,
			})
			if tc.expectErr != "" {
				assert.EqualError(t, err, tc.expectErr)
				return
			}
			require.NoError(t, err)

			files := diffcover.Files()
			assert.NotEmpty(t, files)

			covered := diffcover.Covered()
			assert.Equal(t, tc.expectCovered, covered)
		})
	}
}
