package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/johnstarich/go/gopages/cmd"
	"github.com/johnstarich/go/gopages/internal/flags"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(t *testing.T) {
	cmd.SetupTestExiter(t)
	assert.Panics(t, main)
}

func TestMainArgs(t *testing.T) {
	cmd.SetupTestExiter(t)
	tmp, err := ioutil.TempDir("", "")
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
		t.Run(tc.description, func(t *testing.T) {
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

func TestRun(t *testing.T) {
	for _, tc := range []struct {
		description string
		args        []string
		expectErr   string
	}{
		{
			description: "happy path, no flags",
		},
		{
			description: "happy path, gh-pages",
			args:        []string{"-gh-pages"},
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			// create dummy repo to enable cloning
			ghPagesDir, err := ioutil.TempDir("", "")
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

			modulePath, err := ioutil.TempDir("", "")
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
				err = ioutil.WriteFile(path, []byte(contents), 0600)
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
					name := strings.TrimPrefix(f.Name, "dist"+string(filepath.Separator))
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
					prefix := strings.Join([]string{
						modulePath,
						"dist",
						"",
					}, string(filepath.Separator))
					name := strings.TrimPrefix(path, prefix)
					if err == nil &&
						!info.IsDir() &&
						!strings.HasPrefix(name, string(filepath.Separator)) {
						if strings.HasPrefix(name, "lib") {
							foundLib = true
						} else {
							fileNames = append(fileNames, name)
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
				"pkg/thing/index.html",
				"pkg/thing/lib/index.html",
			}, fileNames)
		})
	}
}

func contains(strs []string, s string) bool {
	for _, str := range strs {
		if str == s {
			return true
		}
	}
	return false
}
