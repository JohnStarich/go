package main

import (
	"io/ioutil"
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
	err = generateDocs(thing, "thing", args, thingFS, outputFS)
	assert.NoError(t, err)

	expectedDocs := []string{
		"404.html",
		"index.html",
		"pkg/index.html",
		"pkg/internal/hello/index.html",
	}
	sort.Strings(expectedDocs)
	var foundDocs []string
	walkFS(t, outputFS, "", func(path string) {
		if !strings.HasPrefix(path, filepath.Join("lib", "godoc")) {
			foundDocs = append(foundDocs, path)
		}
	})
	sort.Strings(foundDocs)
	assert.Equal(t, expectedDocs, foundDocs)

	f, err := outputFS.Open("pkg/index.html")
	require.NoError(t, err)
	indexContents, err := ioutil.ReadAll(f)
	require.NoError(t, err)
	assert.Contains(t, string(indexContents), "internal/hello")
}

func walkFS(t *testing.T, fs billy.Filesystem, path string, visit func(path string)) {
	t.Helper()
	info, err := fs.Stat(path)
	if err != nil {
		t.Fatalf("Error looking up file: %s", err.Error())
	}

	if !info.IsDir() {
		visit(path)
		return
	}

	dir, err := fs.ReadDir(path)
	if err != nil {
		t.Fatalf("Error reading directory %q: %s", path, err.Error())
	}
	for _, info = range dir {
		walkFS(t, fs, filepath.Join(path, info.Name()), visit)
	}
}
