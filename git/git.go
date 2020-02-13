package git

import (
	"context"
	"errors"
	"fmt"
	"time"

	github "github.com/google/go-github/v29/github"
	"golang.org/x/oauth2"
)

var (
	ErrInvalidParam   = errors.New("[git][err] parameters invalids")
	ErrApiQuotaExceed = errors.New("[git][err] api-quota exceeds")

	githubRateLimit *github.RateLimitError
)

type Git interface {
	User() *github.User
	ListStarredAll(retries int) ([]*Repository, error)
}

// NewGit returns a github client by a personal access token.
// reference: https://github.com/settings/tokens
func NewGit(token string) (Git, error) {
	if token == "" {
		return nil, fmt.Errorf("[err] NewGit %w", ErrInvalidParam)
	}

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	client := github.NewClient(oauth2.NewClient(context.Background(), ts))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	w := &wrapper{Client: client}

	user, _, err := w.Users.Get(ctx, "")
	switch {
	case err == nil:
		w.user = user
		return w, nil
	case errors.As(err, &githubRateLimit):
		return nil, fmt.Errorf("[err] NewGit %w", ErrApiQuotaExceed)
	default:
		return nil, fmt.Errorf("[err] NewGit %w", err)
	}
}
