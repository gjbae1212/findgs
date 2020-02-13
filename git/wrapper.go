package git

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	backoff "github.com/cenkalti/backoff/v4"
	github "github.com/google/go-github/v29/github"
)

const (
	perPage        = 100
	requestTimeout = time.Second * 10
	maxSearchCount = 4800 // github api quota is 5000 per hourly. so it makes a limit request count in logically.
)

type wrapper struct {
	*github.Client
	user *github.User
}

type Repository struct {
	Owner           string
	Repo            string
	FullName        string
	Url             string
	Description     string
	Topics          []string
	WatchersCount   int
	StargazersCount int
	ForksCount      int
	Readme          string
}

// User returns github user object.
func (w *wrapper) User() *github.User {
	return w.user
}

// ListStarredAll returns all of starred github projects.
func (w *wrapper) ListStarredAll(retries int) ([]*Repository, error) {
	// initialize backoff
	backoff := backoff.NewExponentialBackOff()
	var starred []*github.StarredRepository

	page := 1
	repeat := 0
	for {
		repos, resp, err := w.listStarredPaging(page, perPage)

		// check whether raised error or not.
		switch {
		case errors.As(err, &githubRateLimit):
			return nil, fmt.Errorf("[err] ListStarredAll %w", ErrApiQuotaExceed)
		case err != nil:
			if repeat < retries {
				repeat++
				time.Sleep(backoff.NextBackOff())
				continue
			}
			return nil, fmt.Errorf("[err] ListStarredAll %w", err)
		}

		// append repos
		starred = append(starred, repos...)
		if page >= resp.LastPage {
			break
		}
		page += 1
	}

	// reinitialize setting
	repeat = 0
	backoff.Reset()

	var result []*Repository
	for _, star := range starred {
		result = append(result, &Repository{
			Owner: star.Repository.Owner.GetLogin(), Repo: star.Repository.GetName(),
			FullName: star.Repository.GetFullName(), Url: star.Repository.GetHTMLURL(),
			Description: star.GetRepository().GetDescription(), Topics: star.GetRepository().Topics,
			WatchersCount: star.GetRepository().GetWatchersCount(), StargazersCount: star.GetRepository().GetStargazersCount(),
			ForksCount: star.GetRepository().GetForksCount(),
		})
	}

	// drop repositories.
	if len(result) > maxSearchCount {
		result = result[:maxSearchCount]
	}

	// set readme
	// make parallel requests.
	wg := sync.WaitGroup{}
	var multi []chan *Repository
	for i := 0; i < 10; i++ {
		wg.Add(1)
		ch := make(chan *Repository, 500)
		multi = append(multi, ch)
		go func(queue chan *Repository) {
			for data := range queue {
				w.setReadme(data)
			}
			wg.Done()
		}(ch)
	}

	// spray items
	for i, r := range result {
		ix := i % 10
		multi[ix] <- r
	}

	// close channel
	for _, ch := range multi {
		close(ch)
	}
	wg.Wait()
	return result, nil
}

func (w *wrapper) listStarredPaging(page, perPage int) ([]*github.StarredRepository, *github.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	opt := &github.ActivityListStarredOptions{}
	opt.Page = page
	opt.PerPage = perPage

	return w.Activity.ListStarred(ctx, "", opt)
}

func (w *wrapper) setReadme(r *Repository) error {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	readme, _, err := w.Repositories.GetReadme(ctx, r.Owner, r.Repo, nil)
	if err != nil {
		return fmt.Errorf("[err] setReadme %w", err)
	}

	content, err := readme.GetContent()
	if err != nil {
		return fmt.Errorf("[err] setReadme %w", err)
	}
	r.Readme = content

	return nil
}
