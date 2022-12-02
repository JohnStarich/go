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
	"github.com/johnstarich/go/gopages/internal/flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateDocs(t *testing.T) { //nolint:paralleltest // TODO: Remove chdir, use a io/fs.FS implementation to work around billy's limitations.
	// create a new package "thing" and generate docs for it
	thing, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	defer os.RemoveAll(thing)
	wd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Chdir(wd))
	}()
	require.NoError(t, os.Chdir(thing))

	thingFS := osfs.New("")
	outputFS := memfs.New()
	writeFile := func(path, contents string) {
		path = filepath.Join(thing, path)
		err := os.MkdirAll(filepath.Dir(path), 0700)
		require.NoError(t, err)
		err = os.WriteFile(path, []byte(contents), 0600)
		require.NoError(t, err)
	}

	writeFile("go.mod", `module github.com/my/thing`)
	writeFile("main.go", `
package main

func main() {
	println("Hello world")
}
`)
	writeFile("internal/hello/hello.go", `
package lib

// Hello says hello
func Hello() {
	println("Hello world")
}
`)
	writeFile("mylib/lib.go", `
package mylib
`)
	writeFile(".git/something", `ignored dot dir`)
	writeFile(".dotfile", `ignored dot file`)
	writeFile("%name.PascalCased%/other.go", `package bad_url_decode`)

	args := flags.Args{}
	const modulePackage = "github.com/my/thing"
	linker, err := args.Linker(modulePackage)
	require.NoError(t, err)
	err = Docs(thing, modulePackage, thingFS, outputFS, args, linker)
	assert.NoError(t, err)

	expectedDocs := []string{
		"404.html",
		"index.html",
		"pkg/github.com/index.html",
		"pkg/github.com/my/index.html",
		"pkg/github.com/my/thing/%name.PascalCased%/index.html", // Verifies fix for https://github.com/JohnStarich/go/issues/7
		"pkg/github.com/my/thing/index.html",
		"pkg/github.com/my/thing/internal/hello/index.html",
		"pkg/github.com/my/thing/internal/index.html",
		"pkg/github.com/my/thing/mylib/index.html",
		"pkg/index.html",
		"src/github.com/index.html",
		"src/github.com/my/index.html",
		"src/github.com/my/thing/%name.PascalCased%/index.html",
		"src/github.com/my/thing/%name.PascalCased%/other.go.html",
		"src/github.com/my/thing/index.html",
		"src/github.com/my/thing/internal/hello/hello.go.html",
		"src/github.com/my/thing/internal/hello/index.html",
		"src/github.com/my/thing/internal/index.html",
		"src/github.com/my/thing/main.go.html",
		"src/github.com/my/thing/mylib/index.html",
		"src/github.com/my/thing/mylib/lib.go.html",
		"src/index.html",
	}
	var foundDocs []string
	require.NoError(t, walkFiles(outputFS, "", func(path string, isDir bool) error {
		if !isDir && !strings.HasPrefix(path, filepath.Join("lib", "godoc")) {
			foundDocs = append(foundDocs, filepath.ToSlash(path))
		}
		return nil
	}))
	sort.Strings(foundDocs)
	assert.Equal(t, expectedDocs, foundDocs)

	f, err := outputFS.Open("pkg/github.com/my/thing/index.html")
	require.NoError(t, err)
	indexContents, err := io.ReadAll(f)
	require.NoError(t, err)
	assert.Contains(t, string(indexContents), "mylib")

	// verify you can run again without removing the output dir
	assert.NoError(t, Docs(thing, modulePackage, thingFS, outputFS, args, linker))
}

func TestGenerateDocsAvoidOverwritingExistingOutput(t *testing.T) {
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
		outputFS := memfs.New()
		const outputDir = "foo"
		err := generateDocs(t, outputFS, outputDir)
		assert.EqualError(t, err, outputPathOKError)
	})

	t.Run("empty output dir is OK", func(t *testing.T) {
		outputFS := memfs.New()
		const outputDir = "foo"
		require.NoError(t, outputFS.MkdirAll(outputDir, 0700))
		err := generateDocs(t, outputFS, outputDir)
		assert.EqualError(t, err, outputPathOKError)
	})

	t.Run("unexpected files in output dir should fail", func(t *testing.T) {
		outputFS := memfs.New()
		const outputDir = "foo"
		require.NoError(t, outputFS.MkdirAll(outputFS.Join(outputDir, "bar"), 0700))
		err := generateDocs(t, outputFS, outputDir)
		assert.EqualError(t, err, `pipe: refusing to clean output directory "foo" - directory does not resemble a gopages result; remove the directory to continue`)
	})

	t.Run("expected files in output dir should succeed", func(t *testing.T) {
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
