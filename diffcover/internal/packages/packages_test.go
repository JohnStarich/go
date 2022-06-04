package packages

import (
	"errors"
	"testing"

	"github.com/johnstarich/go/diffcover/internal/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilePath(t *testing.T) {
	for _, tc := range []struct {
		description      string
		files            map[string]string
		workingDirectory string
		filePattern      string
		expectPath       string
	}{
		{
			description: "root directory",
			files: map[string]string{
				"main.go": `
package main
`,
			},
			workingDirectory: ".",
			filePattern:      "./main.go",
			expectPath:       "main.go",
		},
		{
			description: "subdirectory",
			files: map[string]string{
				"module/main.go": `
package main
`,
			},
			workingDirectory: ".",
			filePattern:      "./module/main.go",
			expectPath:       "module/main.go",
		},
		{
			description: "root directory module path",
			files: map[string]string{
				"go.mod": `
module github.com/myorg/mymodule
`,
				"main.go": `
package main
`,
			},
			workingDirectory: ".",
			filePattern:      "github.com/myorg/mymodule/main.go",
			expectPath:       "main.go",
		},
		{
			description: "root directory module path with working directory",
			files: map[string]string{
				"go.mod": `
module github.com/myorg/mymodule
`,
				"subdir/main.go": `
package main
`,
			},
			workingDirectory: "subdir",
			filePattern:      "github.com/myorg/mymodule/subdir/main.go",
			expectPath:       "main.go",
		},
		{
			description: "subdirectory module path",
			files: map[string]string{
				"mymodule/go.mod": `
module github.com/myorg/mymodule
`,
				"mymodule/subdir/main.go": `
package main
`,
			},
			workingDirectory: "mymodule",
			filePattern:      "github.com/myorg/mymodule/subdir/main.go",
			expectPath:       "subdir/main.go",
		},
		{
			description: "subdirectory module path with working directory",
			files: map[string]string{
				"mymodule/go.mod": `
module github.com/myorg/mymodule
`,
				"mymodule/subdir/main.go": `
package main
`,
			},
			workingDirectory: "mymodule/subdir",
			filePattern:      "github.com/myorg/mymodule/subdir/main.go",
			expectPath:       "main.go",
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			fs := testhelpers.FSWithFiles(t, tc.files)
			pkgFile, err := FilePath(fs, tc.workingDirectory, tc.filePattern, Options{})
			assert.NoError(t, err)
			assert.Equal(t, tc.expectPath, pkgFile)
		})
	}
}

func TestPanicIfErr(t *testing.T) {
	t.Parallel()
	someErr := errors.New("some error")
	assert.PanicsWithValue(t, someErr, func() {
		panicIfErr(someErr)
	})
	assert.NotPanics(t, func() {
		panicIfErr(nil)
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

func TestFindModule(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		description      string
		dir              string
		files            map[string]string
		expectModuleName string
		expectModuleDir  string
		expectErr        string
	}{
		{
			description:      "root module",
			dir:              ".",
			files:            map[string]string{"go.mod": "module mymodule"},
			expectModuleName: "mymodule",
			expectModuleDir:  ".",
		},
		{
			description:      "subdirectory module",
			dir:              "foo",
			files:            map[string]string{"foo/go.mod": "module mymodule"},
			expectModuleName: "mymodule",
			expectModuleDir:  "foo",
		},
		{
			description: "parent directory module",
			dir:         "foo",
			files: map[string]string{
				"go.mod":  "module mymodule",
				"foo/bar": "bar",
			},
			expectModuleName: "mymodule",
			expectModuleDir:  ".",
		},
		{
			description: "malformed go.mod",
			dir:         ".",
			files:       map[string]string{"go.mod": "not a module"},
			expectErr:   "go.mod:1: unknown directive: not",
		},
	} {
		tc := tc // enable parallel sub-tests
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			fs := testhelpers.FSWithFiles(t, tc.files)
			moduleName, moduleDir, err := findModule(fs, tc.dir)
			if tc.expectErr != "" {
				assert.EqualError(t, err, tc.expectErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expectModuleName, moduleName)
			assert.Equal(t, tc.expectModuleDir, moduleDir)
		})
	}
}
