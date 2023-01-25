package generate

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/therve/go/gopages/internal/flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateDocs(t *testing.T) { //nolint:paralleltest // TODO: Remove chdir, use a io/fs.FS implementation to work around billy's limitations.
	wd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(wd))
	})

	const modulePackage = "github.com/my/thing"
	for _, tc := range []struct { //nolint:paralleltest // TODO: Remove chdir, use a io/fs.FS implementation to work around billy's limitations.
		description            string
		args                   flags.Args
		files                  map[string]string
		expectIndexContains    []string
		expectIndexNotContains []string
		expectDocs             []string
	}{
		{
			description: "basic docs",
			args:        flags.Args{},
			files: map[string]string{
				"go.mod": `module github.com/my/thing`,
				"main.go": `
package main

func main() {
	println("Hello world")
}
`,
				"internal/hello/hello.go": `
package hello

// Hello says hello
func Hello() {
	println("Hello world")
}
`,
				"mylib/lib.go": `
package mylib
`,
				".git/something": `ignored dot dir`,
				".dotfile":       `ignored dot file`,
			},
			expectIndexContains: []string{
				"mylib",
			},
			expectIndexNotContains: []string{
				"internal",
			},
			expectDocs: []string{
				"404.html",
				"index.html",
				"pkg/github.com/index.html",
				"pkg/github.com/my/index.html",
				"pkg/github.com/my/thing/index.html",
				"pkg/github.com/my/thing/internal/hello/index.html",
				"pkg/github.com/my/thing/internal/index.html",
				"pkg/github.com/my/thing/mylib/index.html",
				"pkg/index.html",
				"src/github.com/index.html",
				"src/github.com/my/index.html",
				"src/github.com/my/thing/index.html",
				"src/github.com/my/thing/internal/hello/hello.go.html",
				"src/github.com/my/thing/internal/hello/index.html",
				"src/github.com/my/thing/internal/index.html",
				"src/github.com/my/thing/main.go.html",
				"src/github.com/my/thing/mylib/index.html",
				"src/github.com/my/thing/mylib/lib.go.html",
				"src/index.html",
			},
		},
		{
			// Verifies fix for https://github.com/JohnStarich/go/issues/7
			description: "file paths are URL encoded",
			args:        flags.Args{},
			files: map[string]string{
				"go.mod": `module github.com/my/thing`,
				"main.go": `
package main

func main() {
	println("Hello world")
}
`,
				"%name.PascalCased%/other.go": `package bad_url_decode`,
			},
			expectDocs: []string{
				"404.html",
				"index.html",
				"pkg/github.com/index.html",
				"pkg/github.com/my/index.html",
				"pkg/github.com/my/thing/%name.PascalCased%/index.html",
				"pkg/github.com/my/thing/index.html",
				"pkg/index.html",
				"src/github.com/index.html",
				"src/github.com/my/index.html",
				"src/github.com/my/thing/%name.PascalCased%/index.html",
				"src/github.com/my/thing/%name.PascalCased%/other.go.html",
				"src/github.com/my/thing/index.html",
				"src/github.com/my/thing/main.go.html",
				"src/index.html",
			},
		},
		{
			description: "include internal in index",
			args: flags.Args{
				IndexInternalPackages: true,
			},
			files: map[string]string{
				"go.mod": `module github.com/my/thing`,
				"internal/hello/hello.go": `
package hello

// Hello says hello
func Hello() {
	println("Hello world")
}
`,
			},
			expectIndexContains: []string{
				"internal/hello",
			},
			expectDocs: []string{
				"404.html",
				"index.html",
				"pkg/github.com/index.html",
				"pkg/github.com/my/index.html",
				"pkg/github.com/my/thing/index.html",
				"pkg/github.com/my/thing/internal/hello/index.html",
				"pkg/github.com/my/thing/internal/index.html",
				"pkg/index.html",
				"src/github.com/index.html",
				"src/github.com/my/index.html",
				"src/github.com/my/thing/index.html",
				"src/github.com/my/thing/internal/hello/hello.go.html",
				"src/github.com/my/thing/internal/hello/index.html",
				"src/github.com/my/thing/internal/index.html",
				"src/index.html",
			},
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			// create a new package "thing" and generate docs for it
			thing, err := os.MkdirTemp("", "")
			require.NoError(t, err)
			require.NoError(t, os.Chdir(thing))
			t.Cleanup(func() {
				os.RemoveAll(thing)
			})
			thingFS := osfs.New("")
			outputFS := memfs.New()
			writeFile := func(path, contents string) {
				path = filepath.Join(thing, path)
				err := os.MkdirAll(filepath.Dir(path), 0700)
				require.NoError(t, err)
				err = os.WriteFile(path, []byte(contents), 0600)
				require.NoError(t, err)
			}

			for filePath, contents := range tc.files {
				writeFile(filePath, contents)
			}

			linker, err := tc.args.Linker(modulePackage)
			require.NoError(t, err)
			err = Docs(thing, modulePackage, thingFS, outputFS, tc.args, linker)
			assert.NoError(t, err)

			f, err := outputFS.Open("pkg/github.com/my/thing/index.html")
			require.NoError(t, err)
			indexContents, err := io.ReadAll(f)
			require.NoError(t, err)
			indexContentsStr := string(indexContents)
			for _, s := range tc.expectIndexContains {
				assert.Contains(t, indexContentsStr, s)
			}
			for _, s := range tc.expectIndexNotContains {
				assert.NotContains(t, indexContentsStr, s)
			}

			var foundDocs []string
			require.NoError(t, walkFiles(outputFS, "", func(path string, isDir bool) error {
				if !isDir && !strings.HasPrefix(path, filepath.Join("lib", "godoc")) {
					foundDocs = append(foundDocs, filepath.ToSlash(path))
				}
				return nil
			}))
			sort.Strings(foundDocs)
			assert.Equal(t, tc.expectDocs, foundDocs)

			assert.NoError(t, Docs(thing, modulePackage, thingFS, outputFS, tc.args, linker), "Should not fail to re-run doc generation on same output directory.")
		})
	}
}

func TestGenerateDocsAvoidOverwritingExistingOutput(t *testing.T) {
	t.Parallel()
	generateDocs := func(t *testing.T, outputFS billy.Filesystem, outputDir string) error {
		moduleFS := memfs.New()
		args := flags.Args{OutputPath: outputDir}

		const modulePackage = "github.com/my/thing"
		linker, err := args.Linker(modulePackage)
		require.NoError(t, err)

		return Docs(".", modulePackage, moduleFS, outputFS, args, linker)
	}
	const outputPathOKError = "pipe: Are there any Go files present? Failed to initialize corpus: godoc: corpus fstree is nil"

	t.Run("output dir does not exist", func(t *testing.T) {
		t.Parallel()
		outputFS := memfs.New()
		const outputDir = "foo"
		err := generateDocs(t, outputFS, outputDir)
		assert.EqualError(t, err, outputPathOKError)
	})

	t.Run("empty output dir is OK", func(t *testing.T) {
		t.Parallel()
		outputFS := memfs.New()
		const outputDir = "foo"
		require.NoError(t, outputFS.MkdirAll(outputDir, 0700))
		err := generateDocs(t, outputFS, outputDir)
		assert.EqualError(t, err, outputPathOKError)
	})

	t.Run("unexpected files in output dir should fail", func(t *testing.T) {
		t.Parallel()
		outputFS := memfs.New()
		const outputDir = "foo"
		require.NoError(t, outputFS.MkdirAll(outputFS.Join(outputDir, "bar"), 0700))
		err := generateDocs(t, outputFS, outputDir)
		assert.EqualError(t, err, `pipe: refusing to clean output directory "foo" - directory does not resemble a gopages result; remove the directory to continue`)
	})

	t.Run("expected files in output dir should succeed", func(t *testing.T) {
		t.Parallel()
		outputFS := memfs.New()
		const outputDir = "foo"
		require.NoError(t, outputFS.MkdirAll(outputDir, 0700))
		require.NoError(t, outputFS.MkdirAll(outputFS.Join(outputDir, "bar"), 0700)) // include other contents, which are ignored if expected files present
		f, err := outputFS.Create(outputFS.Join(outputDir, "index.html"))
		require.NoError(t, err)
		require.NoError(t, f.Close())
		for _, dirName := range []string{
			"lib",
			"pkg",
			"src",
		} {
			require.NoError(t, outputFS.MkdirAll(outputFS.Join(outputDir, dirName), 0700))
		}

		err = generateDocs(t, outputFS, outputDir)
		assert.EqualError(t, err, outputPathOKError)
	})
}
