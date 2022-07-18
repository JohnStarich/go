package covet

import (
	"bytes"
	"path"
	"strings"
	"testing"

	"github.com/hack-pad/hackpadfs"
	"github.com/johnstarich/go/covet/internal/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReportSummaryMarkdown(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		description    string
		diff           string
		coverage       string
		expectMarkdown string
		expectTerminal string
		expectErr      string
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
			expectMarkdown: `
Diff coverage is below target. Add tests for these files:
|  | Lines | Coverage | File |
| --- |:---:| --- | --- |
| 🟠 | ~~1/2~~ | ~~ 50.0% ██▌  ~~ | covet.go |
`,
			expectTerminal: `
Diff coverage is below target. Add tests for these files:
┌───────┬──────────────┬──────────┐
│ LINES │ COVERAGE     │ FILE     │
├───────┼──────────────┼──────────┤
│  1/2  │  50.0% ██▌   │ covet.go │
└───────┴──────────────┴──────────┘
`,
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
			expectMarkdown: `
Diff coverage is below target. Add tests for these files:
|  | Lines | Coverage | File |
| --- |:---:| --- | --- |
| 🔴 | ~~0/2~~ | ~~  0.0% ▏    ~~ | covet.go |
`,
			expectTerminal: `
Diff coverage is below target. Add tests for these files:
┌───────┬──────────────┬──────────┐
│ LINES │ COVERAGE     │ FILE     │
├───────┼──────────────┼──────────┤
│  0/2  │   0.0% ▏     │ covet.go │
└───────┴──────────────┴──────────┘
`,
		},
	} {
		tc := tc // enable parallel sub-tests
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			diffReader := strings.NewReader(strings.TrimSpace(tc.diff))
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
				Diff:              diffReader,
				DiffBaseDir:       wd,
				GoCoveragePath:    coverFile,
				GoCoverageBaseDir: wd,
			})
			require.NoError(t, err)
			var buf bytes.Buffer
			assert.NoError(t, covet.ReportSummaryMarkdown(&buf, ReportSummaryOptions{
				Target: 90,
			}))
			expectMarkdown := strings.ReplaceAll(tc.expectMarkdown, "~~", "``")
			assert.Equal(t, strings.TrimSpace(expectMarkdown), strings.TrimSpace(buf.String()))
			buf.Reset()
			assert.NoError(t, covet.ReportSummaryColorTerminal(&buf, ReportSummaryOptions{
				Target: 90,
			}))
			assert.Equal(t, strings.TrimSpace(tc.expectTerminal), strings.TrimSpace(buf.String()))
		})
	}
}

func TestFindReportableUncoveredFiles(t *testing.T) {
	t.Parallel()
	t.Run("sort and filter just enough files", func(t *testing.T) {
		t.Parallel()
		files := []File{
			{Name: "foo", Covered: 2, Uncovered: 0},
			{Name: "bar", Covered: 1, Uncovered: 2},
			{Name: "baz", Covered: 1, Uncovered: 2},
			{Name: "biff", Covered: 0, Uncovered: 2},
		}
		reportable := findReportableUncoveredFiles(files, 0.75, 0.4)
		assert.Equal(t, []File{
			{Name: "bar", Covered: 1, Uncovered: 2},
			{Name: "baz", Covered: 1, Uncovered: 2},
			{Name: "biff", Covered: 0, Uncovered: 2},
		}, reportable)
	})

	t.Run("include more small files if the biggest chunks are not close enough to target", func(t *testing.T) {
		t.Parallel()
		files := []File{
			{Name: "foo", Covered: 0, Uncovered: 1},
			{Name: "bar", Covered: 0, Uncovered: 1},
			{Name: "baz", Covered: 0, Uncovered: 1},
			{Name: "biff", Covered: 0, Uncovered: 7},
		}
		reportable := findReportableUncoveredFiles(files, 0.8, 0)
		assert.Equal(t, []File{
			{Name: "biff", Covered: 0, Uncovered: 7},
			{Name: "bar", Covered: 0, Uncovered: 1},
			{Name: "baz", Covered: 0, Uncovered: 1},
			{Name: "foo", Covered: 0, Uncovered: 1},
		}, reportable)
	})
}
