package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWordWrapLines(t *testing.T) {
	const wrapColumn = 20
	for _, tc := range []struct {
		description string
		input       string
		expect      string
	}{
		{
			description: "empty string",
		},
		{
			description: "one long line",
			input:       "This line is pretty darn long.",
			expect:      "This line is pretty\ndarn long.",
		},
		{
			description: "two long lines",
			input: `
This line is pretty darn long.
This one might be too, lots of text.
			`,
			expect: `
This line is pretty
darn long.
This one might be
too, lots of text.
			`,
		},
		{
			description: "wrap indented lines",
			input: `
This line is just adequate.
	This line is pretty darn long.
		This one might be too, lots of text.
			`,
			expect: `
This line is just
adequate.
	This line is pretty
	darn long.
		This one might be
		too, lots of text.
			`,
		},
		{
			description: "don't wrap long words",
			input: `
Super-hyphenated-yet-no-less-amazing.
			`,
			expect: `
Super-hyphenated-yet-no-less-amazing.
			`,
		},
		{
			description: "best effort string quote wrapping - leader is long enough",
			input: `
This little "phrase will" not be broken.
			`,
			expect: `
This little 
"phrase will" not
be broken.
			`,
		},
		{
			description: "best effort string quote wrapping - leader too short alone",
			input: `
This "phrase with multiple words" must not be broken.
			`,
			expect: `
This "phrase with multiple words"
must not be broken.
			`,
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			assert.Equal(t, tc.expect, wordWrapLines(wrapColumn, tc.input))
		})
	}
}
