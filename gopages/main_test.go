package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	gitHTTP "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/johnstarich/go/gopages/cmd"
	"github.com/johnstarich/go/gopages/internal/flags"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(t *testing.T) {
	t.Parallel()
	cmd.SetupTestExiter(t)
	assert.Panics(t, main)
}

func TestMainArgs(t *testing.T) {
	t.Parallel()
	cmd.SetupTestExiter(t)
	tmp, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	defer os.RemoveAll(tmp)

	for _, tc := range []struct {
		description string
		runnerErr   error
		wdErr       error
		args        []string
		expectErr   string
	}{
		{
			description: "bad flag usage",
			args:        []string{"-not-a-flag"},
			expectErr:   "Attempted to exit with exit code 2",
		},
		{
			description: "request usage",
			args:        []string{"-help"},
		},
		{
			description: "getwd error",
			wdErr:       errors.New("some error"),
			expectErr:   "Failed to get current directory: some error",
		},
		{
			description: "runner failed",
			runnerErr:   errors.New("some error"),
			expectErr:   "Attempted to exit with exit code 1",
		},
	} {
		tc := tc // enable parallel sub-tests
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			runner := func(string, flags.Args) error {
				return tc.runnerErr
			}
			getWD := func() (string, error) {
				return tmp, tc.wdErr
			}

			runTest := func() {
				mainArgs(runner, getWD, tc.args...)
			}
			if tc.expectErr != "" {
				assert.PanicsWithError(t, tc.expectErr, runTest)
				return
			}
			assert.NotPanics(t, runTest)
		})
	}
}

func TestRun(t *testing.T) { //nolint:paralleltest // TODO: Remove chdir, use a io/fs.FS implementation to work around billy's limitations.
	//nolint:paralleltest // TODO: Remove chdir, use a io/fs.FS implementation to work around billy's limitations.
	for _, tc := range []testRunTestCase{
		{
			description: "happy path, no flags",
		},
		{
			description: "happy path, gh-pages",
			args:        []string{"-gh-pages"},
			skip:        os.Getenv("CI") == "true" && runtime.GOOS == "windows", // Windows in CI can't handle temp files with working directory ones because they're on different drive letters.
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			testRun(t, tc)
		})
	}
}

type testRunTestCase struct {
	description string
	args        []string
	expectErr   string
	skip        bool
}

func testRun(t *testing.T, tc testRunTestCase) {
	if tc.skip {
		t.Skip("Skipped by test case param")
	}

	// create dummy repo to enable cloning
	ghPagesDir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	defer os.RemoveAll(ghPagesDir)
	ghPagesRepo, err := git.PlainInit(ghPagesDir, false)
	require.NoError(t, err)
	workTree, err := ghPagesRepo.Worktree()
	require.NoError(t, err)
	_, err = workTree.Commit("Initial commit", &git.CommitOptions{
		Author: commitAuthor(),
	})
	require.NoError(t, err)
	require.NoError(t, workTree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(ghPagesBranch),
		Create: true,
	}))

	modulePath, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	defer os.RemoveAll(modulePath)
	wd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(modulePath))
	defer func() {
		require.NoError(t, os.Chdir(wd))
	}()

	// prepare origin remote pointing to dummy repo
	_, err = git.PlainClone(modulePath, false, &git.CloneOptions{
		URL: ghPagesDir,
	})
	require.NoError(t, err)

	writeFile := func(path, contents string) {
		path = filepath.Join(modulePath, path)
		err := os.MkdirAll(filepath.Dir(path), 0700)
		require.NoError(t, err)
		err = os.WriteFile(path, []byte(contents), 0600)
		require.NoError(t, err)
	}

	writeFile("go.mod", `module thing`)
	writeFile("main.go", `
package main

func main() {
	println("Hello world")
}
`)
	writeFile("lib/lib.go", `
package lib

// Hello says hi
func Hello() {
	println("Hello world")
}
`)

	args, _, err := flags.Parse(tc.args...)
	require.NoError(t, err)

	err = run(modulePath, args)
	if tc.expectErr != "" {
		assert.EqualError(t, err, tc.expectErr)
		return
	}
	require.NoError(t, err)

	var foundLib bool
	var fileNames []string
	if contains(tc.args, "-gh-pages") {
		// fetch the new head commit and walk the files in the diff
		head, err := ghPagesRepo.Head()
		require.NoError(t, err)
		headCommit, err := ghPagesRepo.CommitObject(head.Hash())
		require.NoError(t, err)
		files, err := headCommit.Files()
		require.NoError(t, err)
		err = files.ForEach(func(f *object.File) error {
			name := filepath.ToSlash(f.Name)
			name = strings.TrimPrefix(name, "dist/")
			if strings.HasPrefix(name, "lib") {
				foundLib = true
			} else {
				fileNames = append(fileNames, name)
			}
			return nil
		})
		require.NoError(t, err)
	} else {
		err := filepath.Walk(modulePath, func(path string, info os.FileInfo, err error) error {
			prefix := filepath.Join(modulePath, "dist")
			prefix, absErr := filepath.Abs(prefix)
			if absErr != nil {
				return absErr
			}
			prefix += string(filepath.Separator)
			name := strings.TrimPrefix(path, prefix)
			if err == nil &&
				!info.IsDir() &&
				!filepath.IsAbs(name) {
				if strings.HasPrefix(name, "lib") {
					foundLib = true
				} else {
					fileNames = append(fileNames, filepath.ToSlash(name))
				}
			}
			return nil
		})
		require.NoError(t, err)
	}
	require.NoError(t, err)
	assert.True(t, foundLib)
	assert.Equal(t, []string{
		"404.html",
		"index.html",
		"pkg/index.html",
		"pkg/thing/index.html",
		"pkg/thing/lib/index.html",
		"src/index.html",
		"src/thing/index.html",
		"src/thing/lib/index.html",
		"src/thing/lib/lib.go.html",
		"src/thing/main.go.html",
	}, fileNames)
}

func contains(strs []string, s string) bool {
	for _, str := range strs {
		if str == s {
			return true
		}
	}
	return false
}

func TestAuth(t *testing.T) {
	assert.Nil(t, getAuth(flags.Args{}))
	assert.Equal(t,
		&gitHTTP.BasicAuth{Username: "user", Password: "token"},
		getAuth(flags.Args{
			GitHubPagesToken: "token",
			GitHubPagesUser:  "user",
		}))
}
