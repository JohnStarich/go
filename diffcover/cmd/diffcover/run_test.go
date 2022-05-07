package main

import (
	"bytes"
	"path"
	"strings"
	"testing"

	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/mem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		description string
		args        []string
		stdin       string
		expectOut   string
		expectErr   string
	}{
		{
			description: "missing required args",
			args:        nil,
			expectOut:   "Usage of diffcover:",
			expectErr:   "flag -cover-go is required",
		},
		{
			description: "help",
			args:        []string{"-help"},
			expectOut:   "Usage of diffcover:",
		},
		{
			description: "wrong flag type",
			args:        []string{"-target-diff-coverage", "-1"},
			expectOut:   "Usage of diffcover:",
			expectErr:   `invalid value "-1" for flag -target-diff-coverage: parse error`,
		},
		{
			description: "attempt running command",
			args: []string{
				"-diff-file", "-",
				"-cover-go", "/does/not/exist",
			},
			expectOut: "",
			expectErr: "open /does/not/exist: invalid argument",
		},
	} {
		tc := tc // enable parallel sub-tests
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			fs, err := mem.NewFS()
			require.NoError(t, err)
			var output bytes.Buffer
			err = run(
				tc.args,
				strings.NewReader(tc.stdin),
				&output,
				&output,
				fs,
			)
			assert.Contains(t, output.String(), tc.expectOut)
			if tc.expectErr != "" {
				assert.EqualError(t, err, tc.expectErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestRunArgs(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		description string
		args        Args
		stdin       string
		files       map[string]string
		expectOut   string
		expectErr   string
	}{
		{
			description: "print empty diffcover summary",
			args: Args{
				DiffFile:       "my.patch",
				GoCoverageFile: "cover.out",
			},
			files: map[string]string{
				"my.patch":  ``,
				"cover.out": ``,
			},
			expectOut: `
No coverage information intersects with diff.
`,
		},
		{
			description: "print diffcover summary",
			args: Args{
				DiffFile:       "my.patch",
				GoCoverageFile: "cover.out",
			},
			files: map[string]string{
				"my.patch": `
diff --git a/run.go b/run.go
index 0000000..1111111 100644
--- a/run.go
+++ b/run.go
@@ -0,0 +1,2 @@
+added 1
+added 2
`,
				"cover.out": `
mode: atomic
github.com/johnstarich/go/diffcover/cmd/diffcover/run.go:1.1,1.7 1 1
github.com/johnstarich/go/diffcover/cmd/diffcover/run.go:2.1,2.7 1 0
`,
			},
			expectOut: `
Total diff coverage:  50.0%

Diff coverage is below target. Add tests for these files:
┌───────┬──────────────┬────────┐
│ LINES │ COVERAGE     │ FILE   │
├───────┼──────────────┼────────┤
│  1/2  │  50.0% ██▌   │ run.go │
└───────┴──────────────┴────────┘
`,
		},
	} {
		tc := tc // enable parallel sub-tests
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			fs, err := mem.NewFS()
			require.NoError(t, err)
			for name, contents := range tc.files {
				require.NoError(t, hackpadfs.MkdirAll(fs, path.Dir(name), 0700))
				f, err := hackpadfs.Create(fs, name)
				require.NoError(t, err)
				_, err = hackpadfs.WriteFile(f, []byte(strings.TrimSpace(contents)))
				require.NoError(t, err)
				require.NoError(t, f.Close())
			}
			var output bytes.Buffer
			deps := Deps{
				Stdin:  strings.NewReader(tc.stdin),
				Stdout: &output,
				FS:     fs,
			}

			err = runArgs(tc.args, deps)
			assert.Equal(t, strings.TrimSpace(tc.expectOut), strings.TrimSpace(output.String()))
			if tc.expectErr != "" {
				assert.EqualError(t, err, tc.expectErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestParseIssueURL(t *testing.T) {
	for _, tc := range []struct {
		url          string
		expectOrg    string
		expectRepo   string
		expectNumber int
		expectErr    string
	}{
		{
			url:       "",
			expectErr: "-gh-issue is required",
		},
		{
			url:       "example.com",
			expectErr: "malformed issue URL: expected 4+ path components, e.g. github.com/org/repo/pull/123",
		},
		{
			url:          "github.com/myorg/myrepo/pull/123",
			expectOrg:    "myorg",
			expectRepo:   "myrepo",
			expectNumber: 123,
		},
		{
			url:          "github.com/myorg/myrepo/pull/123/extra",
			expectOrg:    "myorg",
			expectRepo:   "myrepo",
			expectNumber: 123,
		},
		{
			url:       "github.com/myorg/myrepo/pull/not-a-number",
			expectErr: `strconv.ParseInt: parsing "not-a-number": invalid syntax`,
		},
	} {
		t.Run(tc.url, func(t *testing.T) {
			org, repo, number, err := parseIssueURL(tc.url)
			assert.Equal(t, tc.expectOrg, org)
			assert.Equal(t, tc.expectRepo, repo)
			assert.Equal(t, tc.expectNumber, number)
			if tc.expectErr != "" {
				assert.EqualError(t, err, tc.expectErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
