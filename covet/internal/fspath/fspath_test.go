package fspath

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func noSlashes(s string) string {
	return strings.ReplaceAll(s, separator, ">")
}

func TestCommonBase(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		a, b   string
		expect string
	}{
		{
			a:      "a",
			b:      "a",
			expect: "a",
		},
		{
			a:      "a",
			b:      "b",
			expect: ".",
		},
		{
			a:      "a/b",
			b:      "a/c",
			expect: "a",
		},
		{
			a:      "a/b/c",
			b:      "a/b/d",
			expect: "a/b",
		},
	} {
		tc := tc // enable parallel sub-tests
		t.Run(fmt.Sprintf("%s --> %s", noSlashes(tc.a), noSlashes(tc.b)), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expect, CommonBase(tc.a, tc.b))
		})
	}
}

func TestRel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		basePath   string
		targetPath string
		expectPath string
		expectErr  string
	}{
		{
			basePath:   "a",
			targetPath: "a",
			expectPath: ".",
		},
		{
			basePath:   "a",
			targetPath: "b",
			expectPath: "../b",
		},
		{
			basePath:   "a",
			targetPath: "a/b",
			expectPath: "b",
		},
		{
			basePath:   "a/b",
			targetPath: "a",
			expectPath: "..",
		},
		{
			basePath:   "a/b/c",
			targetPath: "a/b/c",
			expectPath: ".",
		},
		{
			basePath:   "a/b",
			targetPath: "a/b/c",
			expectPath: "c",
		},
		{
			basePath:   "a/b/c",
			targetPath: "a/b",
			expectPath: "..",
		},
		{
			basePath:   "a/b/c",
			targetPath: "a/b/d",
			expectPath: "../d",
		},
	} {
		description := fmt.Sprintf("%s --> %s", noSlashes(tc.basePath), noSlashes(tc.targetPath))
		tc := tc // enable parallel sub-tests
		t.Run(description, func(t *testing.T) {
			t.Parallel()
			p, err := Rel(tc.basePath, tc.targetPath)
			if tc.expectErr != "" {
				assert.EqualError(t, err, tc.expectErr)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.expectPath, p)
		})
	}
}
