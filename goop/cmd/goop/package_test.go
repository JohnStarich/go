package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParsePackagePattern(t *testing.T) {
	t.Parallel()
	const homeDir = "/homes/me"
	for _, tc := range []struct {
		pattern string
		expect  Package
	}{
		{
			pattern: "foo",
			expect: Package{
				Name: "foo",
				Path: "foo",
			},
		},
		{
			pattern: "foo/bar",
			expect: Package{
				Name: "bar",
				Path: "foo/bar",
			},
		},
		{
			pattern: "foo/bar@version",
			expect: Package{
				Name:          "bar",
				Path:          "foo/bar",
				ModuleVersion: "version",
			},
		},
		{
			pattern: "~/foo/bar",
			expect: Package{
				Name: "bar",
				Path: "~/foo/bar",
			},
		},
		{
			// Canonicalize home directory to '~' for better cross-machine bin support.
			// Home directories can change from user to user, but script should remain portable.
			pattern: homeDir + "/foo/bar",
			expect: Package{
				Name: "bar",
				Path: "~/foo/bar",
			},
		},
	} {
		tc := tc // enable parallel sub-tests
		t.Run(tc.pattern, func(t *testing.T) {
			t.Parallel()
			app := App{
				staticOSHomeDir: homeDir,
			}
			pkg, err := app.parsePackagePattern(tc.pattern)
			assert.NoError(t, err)
			assert.Equal(t, tc.expect, pkg)
		})
	}
}

func TestPackageFilePath(t *testing.T) {
	t.Parallel()
	const homeDir = "/homes/me"
	for _, tc := range []struct {
		description    string
		pkg            Package
		expectFilePath string
		expectOk       bool
	}{
		{
			description: "remote package",
			pkg:         Package{Path: thisPackage},
			expectOk:    false,
		},
		{
			description:    "local package",
			pkg:            Package{Path: "/foo"},
			expectFilePath: "/foo",
			expectOk:       true,
		},
		{
			description:    "local should expand home directory",
			pkg:            Package{Path: "~/foo"},
			expectFilePath: homeDir + "/foo",
			expectOk:       true,
		},
	} {
		tc := tc // enable parallel sub-tests
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			app := App{
				staticOSHomeDir: homeDir,
			}
			filePath, ok := app.packageFilePath(tc.pkg)
			assert.Equal(t, tc.expectFilePath, filePath)
			assert.Equal(t, tc.expectOk, ok)
		})
	}
}
