package main

import (
	"bytes"
	"io/ioutil"
	"testing"

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
