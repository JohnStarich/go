package plist

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToJSON(t *testing.T) {
	for _, tc := range []struct {
		fileName string
		expected string
	}{
		{
			fileName: "time-machine-backup-compare.plist",
			expected: `{
				"Changes": [
					{
						"AddedItem": {
							"Path": "/some/file/path",
							"Size": 153
						}
					},
					{
						"RemovedItem": {
							"Path": "/some/other/file/path",
							"Size": 854071
						}
					}
				],
				"Totals": {
					"AddedSize": 500382415,
					"ChangedSize": 0,
					"RemovedSize": 492709651
				}
			}`,
		},
		{
			fileName: "all-datatypes.plist",
			expected: `[
				true,
				false,
				123,
				123e10,
				"some string",
				"YmFzZTY0IGRhdGE=",
				"2020-01-01T01:00:00Z"
			]`,
		},
	} {
		t.Run(tc.fileName, func(t *testing.T) {
			f, err := os.Open(filepath.Join("testdata", tc.fileName))
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, f.Close())
			})

			buf, err := ToJSON(f)
			if assert.NoError(t, err) {
				assert.JSONEq(t, tc.expected, string(buf))
			}
		})
	}
}

func TestToGo(t *testing.T) {
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
