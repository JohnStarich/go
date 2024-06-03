package generate

import (
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/johnstarich/go/gopages/internal/flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var reGoVersion = regexp.MustCompile(`^go(\d+)\.(\d+)(\.(\d+))?.*$`)

func skipIfNotGo19OrLater(t *testing.T) {
	matches := reGoVersion.FindStringSubmatch(runtime.Version())
	if len(matches) > 2 {
		major, _ := strconv.Atoi(matches[1]) // skip err check, major will be 0 on error
		minor, _ := strconv.Atoi(matches[2]) // skip err check, minor will be 0 on error
		if major > 1 || major == 1 && minor >= 19 {
			return
		}
	}
	t.Skipf("%s not supported prior to go 1.19 (current version: %s)", t.Name(), runtime.Version())
}

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
		trySkip                func(*testing.T)
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
		{
			description: "doclinks rendering",
			args:        flags.Args{},
			trySkip:     skipIfNotGo19OrLater,
			files: map[string]string{
				"go.mod": `module github.com/my/thing`,
				"hello/hello.go": `
package hello
type Hello struct {}
func (*Hello) Method(){}
func Function() {}
`,
				"foo/foo.go": `
package foo
type Foo struct {}
func (*Foo) Bar(){}
func Baz() {}
`,
				"thing.go": `
// Package example is a dummy package showing how comments doc is rendered.
//
// Here are some random doc links:
//   - Std library: [os.File] [*os.File] [encoding/xml.Encoder]
//   - Std imported: [json.Decoder] [json.Encoder]
//   - Packages: [os] [json] [encoding/xml]
//   - This package: [JSONFunc] 
//   - Module Package: [github.com/my/thing/hello]
//   - Module Package Type: [github.com/my/thing/hello.Hello]
//   - Module Package Func: [github.com/my/thing/hello.Function]
//   - Module Package method: [github.com/my/thing/hello.Hello.Method]
//   - Imported Module Package: [foo]
//   - Imported Module Package Type: [foo.Foo]
//   - Imported Module Package Method: [foo.Foo.Bar]
//   - Imported Module Func: [foo.Baz]
package thing

import (
	"encoding/json"
	"github.com/my/thing/foo"
)

// JSONFunc does random stuff with [json.Decoder] and [json.Encoder]
func JSONFunc() {
	json.Marshal("foobar") // Just to import json
	foo.Baz() // Just to import foo
}`,
			},
			expectIndexContains: []string{
				"thing",
				`Std library: <a href="https://pkg.go.dev/os#File">os.File</a> <a href="https://pkg.go.dev/os#File">*os.File</a> <a href="https://pkg.go.dev/encoding/xml#Encoder">encoding/xml.Encoder</a>`,
				`Std imported: <a href="https://pkg.go.dev/encoding/json#Decoder">json.Decoder</a> <a href="https://pkg.go.dev/encoding/json#Encoder">json.Encoder</a>`,
				`Packages: <a href="https://pkg.go.dev/os">os</a> <a href="https://pkg.go.dev/encoding/json">json</a> <a href="https://pkg.go.dev/encoding/xml">encoding/xml</a>`,
				`This package: <a href="#JSONFunc">JSONFunc</a>`,

				// Function docstring
				`JSONFunc does random stuff with <a href="https://pkg.go.dev/encoding/json#Decoder">json.Decoder</a> and <a href="https://pkg.go.dev/encoding/json#Encoder">json.Encoder</a>`,

				`Module Package: <a href="/pkg/github.com/my/thing/hello">github.com/my/thing/hello</a>`,
				`Module Package Type: <a href="/pkg/github.com/my/thing/hello#Hello">github.com/my/thing/hello.Hello</a>`,
				`Module Package Func: <a href="/pkg/github.com/my/thing/hello#Function">github.com/my/thing/hello.Function</a>`,
				`Module Package method: <a href="/pkg/github.com/my/thing/hello#Hello.Method">github.com/my/thing/hello.Hello.Method</a>`,
				`Imported Module Package: <a href="/pkg/github.com/my/thing/foo">foo</a>`,
				`Imported Module Package Type: <a href="/pkg/github.com/my/thing/foo#Foo">foo.Foo</a>`,
				`Imported Module Package Method: <a href="/pkg/github.com/my/thing/foo#Foo.Bar">foo.Foo.Bar</a>`,
				`Imported Module Func: <a href="/pkg/github.com/my/thing/foo#Baz">foo.Baz</a>`,
			},
			expectDocs: []string{
				"404.html",
				"index.html",
				"pkg/github.com/index.html",
				"pkg/github.com/my/index.html",
				"pkg/github.com/my/thing/foo/index.html",
				"pkg/github.com/my/thing/hello/index.html",
				"pkg/github.com/my/thing/index.html",
				"pkg/index.html",
				"src/github.com/index.html",
				"src/github.com/my/index.html",
				"src/github.com/my/thing/foo/foo.go.html",
				"src/github.com/my/thing/foo/index.html",
				"src/github.com/my/thing/hello/hello.go.html",
				"src/github.com/my/thing/hello/index.html",
				"src/github.com/my/thing/index.html",
				"src/github.com/my/thing/thing.go.html",
				"src/index.html",
			},
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			if tc.trySkip != nil {
				tc.trySkip(t)
			}
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
				err := os.MkdirAll(filepath.Dir(path), 0o700)
				require.NoError(t, err)
				err = os.WriteFile(path, []byte(contents), 0o600)
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
		require.NoError(t, outputFS.MkdirAll(outputDir, 0o700))
		err := generateDocs(t, outputFS, outputDir)
		assert.EqualError(t, err, outputPathOKError)
	})

	t.Run("unexpected files in output dir should fail", func(t *testing.T) {
		t.Parallel()
		outputFS := memfs.New()
		const outputDir = "foo"
		require.NoError(t, outputFS.MkdirAll(outputFS.Join(outputDir, "bar"), 0o700))
		err := generateDocs(t, outputFS, outputDir)
		assert.EqualError(t, err, `pipe: refusing to clean output directory "foo" - directory does not resemble a gopages result; remove the directory to continue`)
	})

	t.Run("expected files in output dir should succeed", func(t *testing.T) {
		t.Parallel()
		outputFS := memfs.New()
		const outputDir = "foo"
		require.NoError(t, outputFS.MkdirAll(outputDir, 0o700))
		require.NoError(t, outputFS.MkdirAll(outputFS.Join(outputDir, "bar"), 0o700)) // include other contents, which are ignored if expected files present
		f, err := outputFS.Create(outputFS.Join(outputDir, "index.html"))
		require.NoError(t, err)
		require.NoError(t, f.Close())
		for _, dirName := range []string{
			"lib",
			"pkg",
			"src",
		} {
			require.NoError(t, outputFS.MkdirAll(outputFS.Join(outputDir, dirName), 0o700))
		}

		err = generateDocs(t, outputFS, outputDir)
		assert.EqualError(t, err, outputPathOKError)
	})
}
