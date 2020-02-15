package git

import (
	"context"
	"errors"
	"fmt"
	github "github.com/google/go-github/v29/github"
	"golang.org/x/oauth2"
)

var (
	ErrInvalidParam   = errors.New("[git][err] parameters invalids")
	ErrApiQuotaExceed = errors.New("[git][err] api-quota exceeds")
	ErrNotFound       = errors.New("[git][err] not found")

	githubRateLimit *github.RateLimitError
)

type Git interface {
	User() (*User, error)
	SetReadme(starred []*Starred)
	ListStarredAll() ([]*Starred, error)
	ListReadme(owners []string, repos []string) ([]*Readme, error)
}

// NewGit returns a github client by a personal access token.
// reference: https://github.com/settings/tokens
func NewGit(token string) (Git, error) {
	if token == "" {
		return nil, fmt.Errorf("[err] NewGit %w", ErrInvalidParam)
	}

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	client := github.NewClient(oauth2.NewClient(context.Background(), ts))

	return &wrapper{Client: client, token: token}, nil
}
