package flags

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testFile(t *testing.T, contents string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "file")
	err := os.WriteFile(p, []byte(contents), 0600)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestFilePathContents(t *testing.T) {
	t.Parallel()
	t.Run("no flag", func(t *testing.T) {
		t.Parallel()
		set := flag.NewFlagSet("", flag.ContinueOnError)
		var f FilePathContents
		set.Var(&f, "myflag", "")
		err := set.Parse(nil)
		assert.NoError(t, err)
		assert.Equal(t, "", string(f.Contents()))
	})

	t.Run("valid file", func(t *testing.T) {
		t.Parallel()
		set := flag.NewFlagSet("", flag.ContinueOnError)
		const someContents = "some contents"
		tempFile := testFile(t, someContents)

		var f FilePathContents
		set.Var(&f, "myflag", "")
		err := set.Parse([]string{"-myflag", tempFile})
		assert.NoError(t, err)
		assert.Equal(t, someContents, string(f.Contents()))
		assert.Equal(t, someContents, f.String())
	})

	t.Run("invalid file", func(t *testing.T) {
		t.Parallel()
		set := flag.NewFlagSet("", flag.ContinueOnError)
		var f FilePathContents
		set.Var(&f, "myflag", "")
		err := set.Parse([]string{"-myflag", "/does/not/exist"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "open /does/not/exist: no such file or directory")
		assert.Equal(t, "", f.String())
	})
}
