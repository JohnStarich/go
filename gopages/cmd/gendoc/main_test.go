package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/johnstarich/go/gopages/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocUpToDate(t *testing.T) {
	templateBytes, err := ioutil.ReadFile("doc.go")
	require.NoError(t, err)
	var newDoc bytes.Buffer
	err = genDoc(templateBytes, &newDoc)
	require.NoError(t, err)

	currentDoc, err := ioutil.ReadFile("../../doc.go")
	require.NoError(t, err)
	if !assert.Equal(t, newDoc.String(), string(currentDoc)) {
		t.Log("Usage docs are out of date: Run `go generate ./...` to regenerate them.")
	}
}

func TestRun(t *testing.T) {
	t.Run("no args", func(t *testing.T) {
		assert.EqualError(t, run("", ""), "Provide doc template and output file paths")
	})

	t.Run("missing template file", func(t *testing.T) {
		assert.Error(t, run("/does/not/exist", "unused"))
	})

	t.Run("can't open output file", func(t *testing.T) {
		assert.Error(t, run("./doc.go", "/cannot/open/file"))
	})

	t.Run("invalid template", func(t *testing.T) {
		tmpl, err := ioutil.TempFile("", "")
		require.NoError(t, err)
		defer os.Remove(tmpl.Name())
		output, err := ioutil.TempFile("", "")
		require.NoError(t, err)
		defer os.Remove(output.Name())

		_, err = tmpl.WriteString("{{ InvalidSyntax }}")
		require.NoError(t, err)
		assert.Error(t, run(tmpl.Name(), output.Name()))
	})

	t.Run("happy path", func(t *testing.T) {
		file, err := ioutil.TempFile("", "")
		require.NoError(t, err)
		defer os.Remove(file.Name())

		err = run("./doc.go", file.Name())
		assert.NoError(t, err)
	})
}

func TestMain(t *testing.T) {
	cmd.SetupTestExiter(t)
	assert.PanicsWithError(t, "Attempted to exit with exit code 1", func() {
		main()
	})
}

func TestSmallestNonNegative(t *testing.T) {
	for _, tc := range []struct {
		input  []int
		expect int
	}{
		{
			input:  []int{2, 1, 0},
			expect: 0,
		},
		{
			input:  []int{-1, 2, 3},
			expect: 2,
		},
		{
			input:  []int{1, -2, 3},
			expect: 1,
		},
		{
			input:  []int{1, 2, -3},
			expect: 1,
		},
		{
			input:  []int{-1},
			expect: 0,
		},
	} {
		t.Run(fmt.Sprintln(tc.input), func(t *testing.T) {
			assert.Equal(t, tc.expect, smallestNonNegative(tc.input...))
		})
	}
}
