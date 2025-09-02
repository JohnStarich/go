package main

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func rootFilePath() string {
	root := "/"
	if runtime.GOOS == goosWindows {
		root = `C:\`
	}
	return root
}

func TestParsePackagePattern(t *testing.T) {
	t.Parallel()
	root := rootFilePath()
	homeDir := filepath.Join(root, "homes", "me")
	for _, tc := range []struct {
		pattern string
		expect  Package
		skip    bool
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
			skip: runtime.GOOS == goosWindows, // Tilde '~' expansion is not supported on Windows yet.
		},
		{
			// Canonicalize home directory to '~' for better cross-machine bin support.
			// Home directories can change from user to user, but script should remain portable.
			pattern: homeDir + "/foo/bar",
			expect: Package{
				Name: "bar",
				Path: "~/foo/bar",
			},
			skip: runtime.GOOS == goosWindows, // Tilde '~' expansion is not supported on Windows yet.
		},
	} {
		t.Run(tc.pattern, func(t *testing.T) {
			t.Parallel()
			if tc.skip {
				t.Skip("Skipped by test case param")
			}
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
	root := rootFilePath()
	homeDir := filepath.Join(root, "homes", "me")
	for _, tc := range []struct {
		description    string
		pkg            Package
		expectFilePath string
		expectOk       bool
		skip           bool
	}{
		{
			description: "remote package",
			pkg:         Package{Path: thisPackage},
			expectOk:    false,
		},
		{
			description:    "local package",
			pkg:            Package{Path: filepath.Join(root, "foo")},
			expectFilePath: filepath.Join(root, "foo"),
			expectOk:       true,
		},
		{
			description:    "local should expand home directory",
			pkg:            Package{Path: "~/foo"},
			expectFilePath: homeDir + "/foo",
			expectOk:       true,
			skip:           runtime.GOOS == goosWindows, // Tilde '~' expansion is not supported on Windows yet.
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			if tc.skip {
				t.Skip("Skipped by test case param")
			}
			app := App{
				staticOSHomeDir: homeDir,
			}
			filePath, ok := app.packageFilePath(tc.pkg)
			assert.Equal(t, tc.expectFilePath, filePath)
			assert.Equal(t, tc.expectOk, ok)
		})
	}
}
