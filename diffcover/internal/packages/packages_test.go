package packages

import (
	"testing"

	"github.com/johnstarich/go/diffcover/internal/testhelpers"
	"github.com/stretchr/testify/assert"
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
