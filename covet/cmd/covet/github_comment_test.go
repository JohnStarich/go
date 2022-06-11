package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-github/v44/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureAppGitHubComment(t *testing.T) {
	t.Parallel()
	someCommentID := int64(1)
	created := false
	updated := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch fmt.Sprintf("%s:%s", r.Method, r.URL.Path) {
		// create new comment
		case "GET:/api/v3/repos/my-bot/org/myrepo/issues/456/comments":
			require.NoError(t, json.NewEncoder(w).Encode([]*github.IssueComment{}))
		case "POST:/api/v3/repos/my-bot/org/myrepo/issues/456/comments":
			created = true
			var comment github.IssueComment
			require.NoError(t, json.NewDecoder(r.Body).Decode(&comment))
			assert.Equal(t, github.IssueComment{
				Body: stringPtr(`<!-- covet -->

some body`),
			}, comment)
			w.WriteHeader(http.StatusNoContent)

		// update existing comment
		case "GET:/api/v3/repos/my-bot/org/myrepo/issues/123/comments":
			require.NoError(t, json.NewEncoder(w).Encode([]*github.IssueComment{
				{
					ID: &someCommentID,
					Body: stringPtr(`<!-- covet -->

some body`),
				},
			}))
		case "PATCH:/api/v3/repos/my-bot/org/myrepo/issues/comments/1":
			updated = true
			var comment github.IssueComment
			require.NoError(t, json.NewDecoder(r.Body).Decode(&comment))
			assert.Equal(t, github.IssueComment{
				Body: stringPtr(`<!-- covet -->

some other body`),
			}, comment)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatal("Unknown request method and path:", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	ctx := context.Background()
	// create new comment
	err := ensureAppGitHubComment(ctx, gitHubCommentOptions{
		GitHubEndpoint: server.URL,
		GitHubToken:    "some-token",
		RepoOwner:      "my-bot",
		Repo:           "org/myrepo",
		IssueNumber:    456,
		Body:           "some body",
	})
	assert.NoError(t, err)
	assert.True(t, created)
	// update existing comment
	err = ensureAppGitHubComment(ctx, gitHubCommentOptions{
		GitHubEndpoint: server.URL,
		GitHubToken:    "some-token",
		RepoOwner:      "my-bot",
		Repo:           "org/myrepo",
		IssueNumber:    123,
		Body:           "some other body",
	})
	assert.NoError(t, err)
	assert.True(t, updated)
}

func TestStringOrEmpty(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "", stringOrEmpty(nil))
	s := "foo"
	assert.Equal(t, s, stringOrEmpty(&s))
}
