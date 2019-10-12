package regext

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoveComments(t *testing.T) {
	for _, tc := range []struct {
		description string
		expr        string
		expected    string
	}{
		{
			description: "no comments",
			expr:        "  just a normal expr  ",
			expected:    "  just a normal expr  ",
		},
		{
			description: "trailing comment",
			expr:        " something # with a trailing comment",
			expected:    " something ",
		},
		{
			description: "escaped comment",
			expr:        " something \\# with a trailing comment",
			expected:    " something \\# with a trailing comment",
		},
		{
			description: "escaped comment in a comment",
			expr:        " something #\\# with a trailing comment",
			expected:    " something ",
		},
		{
			description: "comment in a comment",
			expr:        " something # with a trailing #comment",
			expected:    " something ",
		},
		{
			description: "multiline expr",
			expr: `some #comment
			and another #comment
			much # wow`,
			expected: `some 
			and another 
			much `,
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			assert.Equal(t, tc.expected, removeComments(tc.expr))
		})
	}
}

func TestRemoveWhitespace(t *testing.T) {
	for _, tc := range []struct {
		description string
		expr        string
		expected    string
	}{
		{
			description: "only spaces",
			expr:        " \t  \n  ",
			expected:    "",
		},
		{
			description: "leading spaces",
			expr:        "   hi",
			expected:    "hi",
		},
		{
			description: "trailing spaces",
			expr:        "hi    ",
			expected:    "hi",
		},
		{
			description: "surrounding spaces",
			expr:        "   hi   ",
			expected:    "hi",
		},
		{
			description: "multiline expr",
			expr: `
				some lines \s here
			and here
			`,
			expected: `somelines\shereandhere`,
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			assert.Equal(t, tc.expected, removeWhitespace(tc.expr))
		})
	}
}

func TestExtendedRegex(t *testing.T) {
	for _, tc := range []struct {
		description string
		expr        string
		expected    string
	}{
		{
			description: "blank expr",
			expr:        "",
			expected:    "",
		},
		{
			description: "nothing to remove",
			expr:        `some(\sexpr)`,
			expected:    `some(\sexpr)`,
		},
		{
			description: "remove comment",
			expr:        `some#comment`,
			expected:    `some`,
		},
		{
			description: "remove whitespace",
			expr: `some
			(\s expr)  `,
			expected: `some(\sexpr)`,
		},
		{
			description: "remove both comments and whitespace",
			expr: `
				some \s #complex
				(\s expr)? # expression
			`,
			expected: `some\s(\sexpr)?`,
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			assert.Equal(t, tc.expected, extendedRegexp(tc.expr))
		})
	}
}

func TestCompile(t *testing.T) {
	expr, err := Compile(`some
		long #expr`)
	require.NoError(t, err)
	assert.Equal(t, regexp.MustCompile(`somelong`), expr)

	_, err = Compile(`broken expr**`)
	assert.Error(t, err)
}

func TestMustCompile(t *testing.T) {
	assert.NotPanics(t, func() {
		expr := MustCompile(`some
			long #expr`)
		assert.Equal(t, regexp.MustCompile(`somelong`), expr)
	})

	assert.Panics(t, func() {
		_ = MustCompile(`broken expr**`)
	})
}
