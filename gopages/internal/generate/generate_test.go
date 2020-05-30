package generate

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/johnstarich/go/gopages/internal/flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateDocs(t *testing.T) {
	// create a new package "thing" and generate docs for it
	thing, err := ioutil.TempDir("", "")
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
	writeFile("internal/hello/hello.go", `
package lib

// Hello says hello
func Hello() {
	println("Hello world")
}
`)

	args := flags.Args{}
	err = Docs(thing, "thing", thingFS, outputFS, args)
	assert.NoError(t, err)

	expectedDocs := []string{
		"404.html",
		"index.html",
		"pkg/thing/index.html",
		"pkg/thing/internal/hello/index.html",
		"src/thing/internal/hello/hello.go.html",
		"src/thing/main.go.html",
	}
	sort.Strings(expectedDocs)
	var foundDocs []string
	require.NoError(t, walkFiles(outputFS, "", func(path string) error {
		if !strings.HasPrefix(path, filepath.Join("lib", "godoc")) {
			foundDocs = append(foundDocs, path)
		}
		return nil
	}))
	sort.Strings(foundDocs)
	assert.Equal(t, expectedDocs, foundDocs)

	f, err := outputFS.Open("pkg/thing/index.html")
	require.NoError(t, err)
	indexContents, err := ioutil.ReadAll(f)
	require.NoError(t, err)
	assert.Contains(t, string(indexContents), "internal/hello")
}
