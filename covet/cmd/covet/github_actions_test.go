package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWorkflowCommand(t *testing.T) {
	command := workflowCommand("error", "My message.", map[string]string{
		"file": "someFile.txt",
		"name": "hello",
	})
	assert.Equal(t, `::error file=someFile.txt,name=hello::My message.`, command)
}
