package diffcover

import (
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	for _, tc := range []struct {
		description    string
		diff           string
		diffReader     io.Reader
		coverage       string
		coverageReader io.Reader
		expectCovered  float64
		expectErr      string
	}{
		{
			description: "half covered",
			diff: `
diff --git a/diffcover.go b/diffcover.go
index 0000000..1111111 100644
--- a/diffcover.go
+++ b/diffcover.go
@@ -0,0 +1,2 @@
+added 1
+added 2
`,
			coverage: `
mode: atomic
github.com/johnstarich/go/diffcover/diffcover.go:1.1,1.7 1 1
github.com/johnstarich/go/diffcover/diffcover.go:2.1,2.7 1 0
`,
			expectCovered: 0.5,
		},
		{
			description: "no coverage hits",
			diff: `
diff --git a/diffcover.go b/diffcover.go
index 0000000..1111111 100644
--- a/diffcover.go
+++ b/diffcover.go
@@ -0,0 +1,2 @@
+added 1
+added 2
`,
			coverage: `
mode: atomic
github.com/johnstarich/go/diffcover/diffcover.go:1.1,1.7 1 0
github.com/johnstarich/go/diffcover/diffcover.go:2.1,2.7 1 0
`,
			expectCovered: 0,
		},
		{
			description: "bad diff reader",
			diffReader:  iotest.ErrReader(errors.New("some error")),
			coverage: `
mode: atomic
github.com/johnstarich/go/diffcover/diffcover.go:1.1,1.7 1 1
github.com/johnstarich/go/diffcover/diffcover.go:2.1,2.7 1 0
`,
			expectErr: "some error",
		},
		{
			description: "bad coverage reader",
			diff: `
diff --git a/diffcover.go b/diffcover.go
index 0000000..1111111 100644
--- a/diffcover.go
+++ b/diffcover.go
@@ -0,0 +1,2 @@
+added 1
+added 2
`,
			coverageReader: iotest.ErrReader(errors.New("some error")),
			expectErr:      "some error",
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			if tc.diffReader == nil {
				tc.diffReader = strings.NewReader(strings.TrimSpace(tc.diff))
			}
			if tc.coverageReader == nil {
				tc.coverageReader = strings.NewReader(strings.TrimSpace(tc.coverage))
			}
			diffcover, err := Parse(Options{
				Diff:       tc.diffReader,
				GoCoverage: tc.coverageReader,
			})
			if tc.expectErr != "" {
				assert.EqualError(t, err, tc.expectErr)
				return
			}
			assert.NoError(t, err)

			files := diffcover.Files()
			assert.NotEmpty(t, files)

			covered := diffcover.Covered()
			assert.Equal(t, tc.expectCovered, covered)
		})
	}
}
