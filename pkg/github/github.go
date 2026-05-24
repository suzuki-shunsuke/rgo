package github

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/google/go-github/v88/github"
	"golang.org/x/oauth2"
)

type (
	ListOptions        = github.ListOptions
	Reference          = github.Reference
	Response           = github.Response
	RepositoryTag      = github.RepositoryTag
	RepositoryRelease  = github.RepositoryRelease
	Client             = github.Client
	GitObject          = github.GitObject
	Commit             = github.Commit
	PullRequestComment = github.PullRequestComment
	Repository         = github.Repository
)

func New(ctx context.Context) (*Client, error) {
	client, err := github.NewClient(github.WithHTTPClient(getHTTPClientForGitHub(ctx, getGitHubToken())))
	if err != nil {
		return nil, fmt.Errorf("create a GitHub client: %w", err)
	}
	return client, nil
}

func getGitHubToken() string {
	return os.Getenv("GITHUB_TOKEN")
}

func getHTTPClientForGitHub(ctx context.Context, token string) *http.Client {
	if token == "" {
		return http.DefaultClient
	}
	return oauth2.NewClient(ctx, oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	))
}
