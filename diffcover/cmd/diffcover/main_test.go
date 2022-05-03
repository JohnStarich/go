package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	for _, tc := range []struct {
		description string
		args        []string
		expectErr   string
	}{
		{
			description: "missing required args",
			args:        nil,
			expectErr:   "flag -cover-go is required",
		},
		{
			description: "help",
			args:        []string{"-help"},
		},
		{
			description: "wrong flag type",
			args:        []string{"-target-diff-coverage", "-1"},
			expectErr:   `invalid value "-1" for flag -target-diff-coverage: parse error`,
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			err := run(tc.args)
			if tc.expectErr != "" {
				assert.EqualError(t, err, tc.expectErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestParseIssueURL(t *testing.T) {
	for _, tc := range []struct {
		url          string
		expectOrg    string
		expectRepo   string
		expectNumber int
		expectErr    string
	}{
		{
			url:       "",
			expectErr: "-gh-issue is required",
		},
		{
			url:       "example.com",
			expectErr: "malformed issue URL: expected 4+ path components, e.g. github.com/org/repo/pull/123",
		},
		{
			url:          "github.com/myorg/myrepo/pull/123",
			expectOrg:    "myorg",
			expectRepo:   "myrepo",
			expectNumber: 123,
		},
		{
			url:          "github.com/myorg/myrepo/pull/123/extra",
			expectOrg:    "myorg",
			expectRepo:   "myrepo",
			expectNumber: 123,
		},
		{
			url:       "github.com/myorg/myrepo/pull/not-a-number",
			expectErr: `strconv.ParseInt: parsing "not-a-number": invalid syntax`,
		},
	} {
		t.Run(tc.url, func(t *testing.T) {
			org, repo, number, err := parseIssueURL(tc.url)
			assert.Equal(t, tc.expectOrg, org)
			assert.Equal(t, tc.expectRepo, repo)
			assert.Equal(t, tc.expectNumber, number)
			if tc.expectErr != "" {
				assert.EqualError(t, err, tc.expectErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
