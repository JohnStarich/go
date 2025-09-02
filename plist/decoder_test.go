package plist

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToJSON(t *testing.T) {
	t.Parallel()
	/*
		testdata layout:
		- *.plist - Opened as a reader for ToJSON.
		- *.json - The expected output. If not exists, check ToJSON's error.
		- *.error - The expected error.
	*/
	testFiles, err := filepath.Glob(filepath.Join("testdata", "*.plist"))
	require.NoError(t, err)
	require.NotEmpty(t, testFiles, "Ensure glob matches something.")
	for _, testFilePath := range testFiles {
		baseName := filepath.Base(testFilePath)
		t.Run(baseName, func(t *testing.T) {
			t.Parallel()
			rootName := strings.TrimSuffix(baseName, filepath.Ext(baseName))
			expectedFilePath := filepath.Join("testdata", rootName+".json")
			expectedErrFilePath := filepath.Join("testdata", rootName+".error")

			testFile, err := os.Open(testFilePath)
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, testFile.Close())
			})
			expectErr := ""
			expected, err := os.ReadFile(expectedFilePath)
			if errors.Is(err, os.ErrNotExist) {
				var errBuf []byte
				errBuf, err = os.ReadFile(expectedErrFilePath)
				expectErr = strings.TrimSpace(string(errBuf))
			}
			assert.NoError(t, err)

			buf, err := ToJSON(testFile)
			if expectErr != "" {
				assert.Empty(t, string(buf))
				assert.EqualError(t, err, expectErr)
				return
			}
			if assert.NoError(t, err) {
				assert.JSONEq(t, string(expected), string(buf))
			}
		})
	}
}

func TestToGo(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		fileName string
		expected interface{}
	}{
		{
			fileName: "time-machine-backup-compare.plist",
			expected: map[string]interface{}{
				"Changes": []interface{}{
					map[string]interface{}{
						"AddedItem": map[string]interface{}{
							"Path": "/some/file/path",
							"Size": int64(153),
						},
					},
					map[string]interface{}{
						"RemovedItem": map[string]interface{}{
							"Path": "/some/other/file/path",
							"Size": int64(854071),
						},
					},
				},
				"Totals": map[string]interface{}{
					"AddedSize":   int64(500382415),
					"ChangedSize": int64(0),
					"RemovedSize": int64(492709651),
				},
			},
		},
		{
			fileName: "all-datatypes.plist",
			expected: []interface{}{
				true,
				false,
				int64(123),
				123e10,
				"some string",
				[]byte("base64 data"),
				time.Date(2020, time.January, 1, 1, 0, 0, 0, time.UTC),
			},
		},
	} {
		t.Run(tc.fileName, func(t *testing.T) {
			t.Parallel()
			f, err := os.Open(filepath.Join("testdata", tc.fileName))
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, f.Close())
			})

			value, err := newDecoder(f).toGo()
			if assert.NoError(t, err) {
				assert.Equal(t, tc.expected, value)
			}
		})
	}
}
