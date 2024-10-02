package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"
	"text/template"

	"github.com/hack-pad/hackpadfs/mem"
	"github.com/johnstarich/go/covet"
	"github.com/johnstarich/go/covet/internal/span"
	"github.com/johnstarich/go/covet/internal/testhelpers"
	"github.com/pkg/errors"
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
			expectOut:   "Usage of covet:",
			expectErr:   "flag -cover-go is required",
		},
		{
			description: "help",
			args:        []string{"-help"},
			expectOut:   "Usage of covet:",
		},
		{
			description: "wrong flag type",
			args:        []string{"-target-diff-coverage", "-1"},
			expectOut:   "Usage of covet:",
			expectErr:   `invalid value "-1" for flag -target-diff-coverage: parse error`,
		},
		{
			description: "attempt running command",
			args: []string{
				"-diff-file", "-",
				"-cover-go", "/does/not/exist",
			},
			expectOut: "",
			expectErr: "covet: open does/not/exist: file does not exist",
		},
	} {
		tc := tc // enable parallel sub-tests
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			fs, err := mem.NewFS()
			require.NoError(t, err)
			require.NoError(t, fs.Mkdir("tmp", 0o700))
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
			description: "print empty covet summary",
			args: Args{
				DiffFile:       "my.patch",
				GoCoverageFile: "cover.out",
			},
			files: map[string]string{
				"my.patch": ``,
				"cover.out": `
mode: atomic
`,
			},
			expectOut: `
No coverage information intersects with diff.
`,
		},
		{
			description: "print covet summary",
			args: Args{
				DiffFile:       "my.patch",
				GoCoverageFile: "cover.out",
			},
			files: map[string]string{
				"my.patch": `
diff --git a/run.go b/run.go
index 0000000..1111111 100644
--- a/cmd/covet/main.go
+++ b/cmd/covet/main.go
@@ -1,4 +1,6 @@
 package main

 func main() {
+	println(1)
+	println(2)
 }
`,
				"cover.out": `
mode: atomic
github.com/johnstarich/go/covet/cmd/covet/main.go:4.1,4.9 1 1
github.com/johnstarich/go/covet/cmd/covet/main.go:5.1,5.9 1 0
`,
				"go.mod": `
module github.com/johnstarich/go/covet
`,
				"cmd/covet/main.go": `
package main

func main() {
	println(1)
	println(2)
}
`,
			},
			expectOut: `
Total diff coverage:  50.0%

Diff coverage is below target. Add tests for these files:
┌───────┬──────────────┬───────────────────┐
│ LINES │ COVERAGE     │ FILE              │
├───────┼──────────────┼───────────────────┤
│  1/2  │  50.0% ██▌   │ cmd/covet/main.go │
└───────┴──────────────┴───────────────────┘
`,
		},
		{
			description: "print covet summary subdirectory",
			args: Args{
				DiffFile:       "my.patch",
				GoCoverageFile: "mypkg/cover.out",
			},
			files: map[string]string{
				"my.patch": `
diff --git a/run.go b/run.go
index 0000000..1111111 100644
--- a/mypkg/main.go
+++ b/mypkg/main.go
@@ -1,4 +1,6 @@
 package main

 func main() {
+	println(1)
+	println(2)
 }
`,
				"mypkg/cover.out": `
mode: atomic
mymodule/main.go:4.1,4.9 1 1
mymodule/main.go:5.1,5.9 1 0
`,
				"mypkg/go.mod": `
module mymodule
`,
				"mypkg/main.go": `
package main

func main() {
	println(1)
	println(2)
}
`,
			},
			expectOut: `
Total diff coverage:  50.0%

Diff coverage is below target. Add tests for these files:
┌───────┬──────────────┬─────────┐
│ LINES │ COVERAGE     │ FILE    │
├───────┼──────────────┼─────────┤
│  1/2  │  50.0% ██▌   │ main.go │
└───────┴──────────────┴─────────┘
`,
		},
		{
			description: "print covet summary parent directory",
			args: Args{
				DiffFile:       "my.patch",
				GoCoverageFile: "cover.out",
			},
			files: map[string]string{
				"my.patch": `
diff --git a/run.go b/run.go
index 0000000..1111111 100644
--- a/mypkg/main.go
+++ b/mypkg/main.go
@@ -1,4 +1,6 @@
 package main

 func main() {
+	println(1)
+	println(2)
 }
`,
				"cover.out": `
mode: atomic
mymodule/mypkg/main.go:4.1,4.9 1 1
mymodule/mypkg/main.go:5.1,5.9 1 0
`,
				"go.mod": `
module mymodule
`,
				"mypkg/main.go": `
package main

func main() {
	println(1)
	println(2)
}
`,
			},
			expectOut: `
Total diff coverage:  50.0%

Diff coverage is below target. Add tests for these files:
┌───────┬──────────────┬───────────────┐
│ LINES │ COVERAGE     │ FILE          │
├───────┼──────────────┼───────────────┤
│  1/2  │  50.0% ██▌   │ mypkg/main.go │
└───────┴──────────────┴───────────────┘
`,
		},
		{
			description: "print covet",
			args: Args{
				DiffFile:       "my.patch",
				GoCoverageFile: "cover.out",
				ShowCoverage:   true,
			},
			files: map[string]string{
				"my.patch": `
diff --git a/run.go b/run.go
index 0000000..1111111 100644
--- a/cmd/covet/main.go
+++ b/cmd/covet/main.go
@@ -1,4 +1,6 @@
 package main

 func main() {
+	println(1)
+	println(2)
 }
`,
				"cover.out": `
mode: atomic
github.com/johnstarich/go/covet/cmd/covet/main.go:4.1,4.9 1 1
github.com/johnstarich/go/covet/cmd/covet/main.go:5.1,5.9 1 0
`,
				"go.mod": `
module github.com/johnstarich/go/covet
`,
				"cmd/covet/main.go": `
package main

func main() {
	println(1)
	println(2)
}
`,
			},
			expectOut: `
Coverage diff: cmd/covet/main.go
Coverage: 2 to 6
 
 func main() {
+	println(1)
-	println(2)
 }

Total diff coverage:  50.0%

Diff coverage is below target. Add tests for these files:
┌───────┬──────────────┬───────────────────┐
│ LINES │ COVERAGE     │ FILE              │
├───────┼──────────────┼───────────────────┤
│  1/2  │  50.0% ██▌   │ cmd/covet/main.go │
└───────┴──────────────┴───────────────────┘
`,
		},
		{
			description: "post to github comment - bad status does not fail command",
			args: Args{
				DiffFile:       "my.patch",
				GoCoverageFile: "cover.out",
				GitHubEndpoint: "replace-me",
				GitHubToken:    "some-gh-token",
				GitHubIssue:    "github.com/org/repo/pull/123",
			},
			files: map[string]string{
				"my.patch": `
diff --git a/run.go b/run.go
index 0000000..1111111 100644
--- a/cmd/covet/main.go
+++ b/cmd/covet/main.go
@@ -1,4 +1,6 @@
 package main

 func main() {
+	println(1)
+	println(2)
 }
`,
				"cover.out": `
mode: atomic
github.com/johnstarich/go/covet/cmd/covet/main.go:4.1,4.9 1 1
github.com/johnstarich/go/covet/cmd/covet/main.go:5.1,5.9 1 0
`,
				"go.mod": `
module github.com/johnstarich/go/covet
`,
				"cmd/covet/main.go": `
package main

func main() {
	println(1)
	println(2)
}
`,
			},
			expectOut: `
Total diff coverage:  50.0%

Diff coverage is below target. Add tests for these files:
┌───────┬──────────────┬───────────────────┐
│ LINES │ COVERAGE     │ FILE              │
├───────┼──────────────┼───────────────────┤
│  1/2  │  50.0% ██▌   │ cmd/covet/main.go │
└───────┴──────────────┴───────────────────┘

Failed to update GitHub comment, skipping. Error: GET {{.ServerURL}}/api/v3/repos/org/repo/issues/123/comments?sort=created: 500  []
`,
		},
		{
			description: "post to github comment - bad issue URL",
			args: Args{
				DiffFile:       "my.patch",
				GoCoverageFile: "cover.out",
				GitHubToken:    "some-gh-token",
				GitHubIssue:    "foo",
			},
			files: map[string]string{
				"my.patch": `
diff --git a/run.go b/run.go
index 0000000..1111111 100644
--- a/cmd/covet/main.go
+++ b/cmd/covet/main.go
@@ -1,4 +1,6 @@
 package main

 func main() {
+	println(1)
+	println(2)
 }
`,
				"cover.out": `
mode: atomic
github.com/johnstarich/go/covet/cmd/covet/main.go:4.1,4.9 1 1
github.com/johnstarich/go/covet/cmd/covet/main.go:5.1,5.9 1 0
`,
				"go.mod": `
module github.com/johnstarich/go/covet
`,
				"cmd/covet/main.go": `
package main

func main() {
	println(1)
	println(2)
}
`,
			},
			expectOut: `
Total diff coverage:  50.0%

Diff coverage is below target. Add tests for these files:
┌───────┬──────────────┬───────────────────┐
│ LINES │ COVERAGE     │ FILE              │
├───────┼──────────────┼───────────────────┤
│  1/2  │  50.0% ██▌   │ cmd/covet/main.go │
└───────┴──────────────┴───────────────────┘
`,
			expectErr: "malformed issue URL: expected 4+ path components, e.g. github.com/org/repo/pull/123",
		},
	} {
		tc := tc // enable parallel sub-tests
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			fs := testhelpers.FSWithFiles(t, tc.files)
			var output bytes.Buffer
			deps := Deps{
				Stdin:  strings.NewReader(tc.stdin),
				Stdout: &output,
				FS:     fs,
			}
			args := tc.args
			args.DiffBaseDir = "."
			if args.TargetDiffCoverage == 0 {
				args.TargetDiffCoverage = 90
			}

			if args.GitHubEndpoint == "replace-me" {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}))
				args.GitHubEndpoint = server.URL
				t.Cleanup(server.Close)
			}

			var expectOut bytes.Buffer
			require.NoError(t, template.Must(template.New("").Parse(tc.expectOut)).Execute(&expectOut, map[string]interface{}{
				"ServerURL": args.GitHubEndpoint,
			}))

			err := runArgs(args, deps)
			assert.Equal(t, strings.TrimSpace(expectOut.String()), strings.TrimSpace(output.String()))
			if tc.expectErr != "" {
				assert.EqualError(t, err, tc.expectErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestParseIssueURL(t *testing.T) {
	t.Parallel()
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
		tc := tc // enable parallel sub-tests
		t.Run(tc.url, func(t *testing.T) {
			t.Parallel()
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

func TestParseArgs(t *testing.T) {
	t.Parallel()

	t.Run("missing required param", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		_, err := parseArgs([]string{}, &buf)
		assert.EqualError(t, err, "flag -cover-go is required")
	})

	t.Run("invalid flags", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		_, err := parseArgs([]string{
			"-target-diff-coverage", "not-a-number",
		}, &buf)
		assert.EqualError(t, err, `invalid value "not-a-number" for flag -target-diff-coverage: parse error`)
	})

	t.Run("set fs paths", func(t *testing.T) {
		t.Parallel()
		const (
			someCoverPath = "some-cover-path"
			someDiffPath  = "mydiff.patch"
		)
		var buf bytes.Buffer
		args, err := parseArgs([]string{
			"-cover-go", someCoverPath,
			"-diff-file", someDiffPath,
		}, &buf)
		assert.NoError(t, err)

		wd, err := os.Getwd()
		require.NoError(t, err)
		_, workingDir := testhelpers.FromOSToFS(t, wd)

		assert.Equal(t, Args{
			DiffFile:           path.Join(workingDir, someDiffPath),
			DiffBaseDir:        workingDir,
			GoCoverageFile:     path.Join(workingDir, someCoverPath),
			TargetDiffCoverage: 90,
			GitHubEndpoint:     "https://api.github.com",
		}, args)
	})
}

func TestSetErr(t *testing.T) {
	t.Parallel()
	someError := errors.New("some error")
	var err error
	setErr(nil, &err)
	assert.NoError(t, err)

	setErr(someError, &err)
	assert.Same(t, someError, err)
}

func TestFindUncoveredLines(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		description string
		lines       []covet.Line
		expect      []span.Span
	}{
		{
			description: "no lines",
			lines:       []covet.Line{},
			expect:      nil,
		},
		{
			description: "1 covered line",
			lines: []covet.Line{
				{Covered: true, LineNumber: 1},
			},
			expect: nil,
		},
		{
			description: "1 uncovered line",
			lines: []covet.Line{
				{Covered: false, LineNumber: 1},
			},
			expect: []span.Span{
				{Start: 1, End: 2},
			},
		},
		{
			description: "2 small spans",
			lines: []covet.Line{
				{Covered: false, LineNumber: 1},
				{Covered: true, LineNumber: 2},
				{Covered: false, LineNumber: 3},
			},
			expect: []span.Span{
				{Start: 1, End: 2},
				{Start: 3, End: 4},
			},
		},
		{
			description: "2 sparse spans",
			lines: []covet.Line{
				{Covered: false, LineNumber: 1},
				{Covered: false, LineNumber: 3},
			},
			expect: []span.Span{
				{Start: 1, End: 2},
				{Start: 3, End: 4},
			},
		},
		{
			description: "2 spans with 1 larger span",
			lines: []covet.Line{
				{Covered: false, LineNumber: 1},
				{Covered: false, LineNumber: 3},
				{Covered: false, LineNumber: 4},
			},
			expect: []span.Span{
				{Start: 3, End: 5},
				{Start: 1, End: 2},
			},
		},
		{
			description: "multiple spans",
			lines: []covet.Line{
				{Covered: false, LineNumber: 1},

				{Covered: true, LineNumber: 3},
				{Covered: false, LineNumber: 4},

				{Covered: true, LineNumber: 7},
				{Covered: true, LineNumber: 8},

				{Covered: false, LineNumber: 10},
				{Covered: false, LineNumber: 11},
				{Covered: false, LineNumber: 12},
				{Covered: false, LineNumber: 13},
			},
			expect: []span.Span{
				{Start: 10, End: 14},
				{Start: 1, End: 2},
				{Start: 4, End: 5},
			},
		},
	} {
		tc := tc // enable parallel sub-tests
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			spans := findUncoveredLines(covet.File{
				Lines: tc.lines,
			})
			assert.Equal(t, tc.expect, spans)
		})
	}
}
