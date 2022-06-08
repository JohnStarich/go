package main

import (
	"context"
	"strings"

	"github.com/google/go-github/v44/github"
	"golang.org/x/oauth2"
)

type gitHubCommentOptions struct {
	GitHubEndpoint string
	GitHubToken    string
	RepoOwner      string
	Repo           string
	IssueNumber    int
	Body           string
}

func ensureAppGitHubComment(ctx context.Context, options gitHubCommentOptions) error {
	const appMarker = "<!-- covet -->\n\n"
	authClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: options.GitHubToken}))

	client, err := github.NewEnterpriseClient(options.GitHubEndpoint, "", authClient)
	if err != nil {
		return err
	}
	comments, _, err := client.Issues.ListComments(ctx, options.RepoOwner, options.Repo, options.IssueNumber, &github.IssueListCommentsOptions{
		Sort: stringPtr("created"),
	})
	if err != nil {
		return err
	}
	var comment *github.IssueComment
	for _, c := range comments {
		if strings.HasPrefix(stringOrEmpty(c.Body), appMarker) {
			comment = c
		}
	}
	newComment := &github.IssueComment{
		Body: stringPtr(appMarker + options.Body),
	}
	if comment == nil {
		_, _, err = client.Issues.CreateComment(ctx, options.RepoOwner, options.Repo, options.IssueNumber, newComment)
	} else {
		_, _, err = client.Issues.EditComment(ctx, options.RepoOwner, options.Repo, *comment.ID, newComment)
	}
	return err
}

func stringPtr(s string) *string {
	return &s
}

func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
