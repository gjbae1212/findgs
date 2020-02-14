package git

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	github "github.com/google/go-github/v29/github"
)

const (
	perPage        = 100
	parallelSize   = 20
	requestTimeout = time.Second * 10
)

type Readme struct {
	Owner   string `json:"readme,omitempty"`
	Repo    string `json:"repo,omitempty"`
	Content string `json:"content,omitempty"`
	Err     error  `json:"-"`
}

type Starred struct {
	Owner           string   `json:"owner,omitempty"`
	Repo            string   `json:"repo,omitempty"`
	FullName        string   `json:"full_name,omitempty"`
	Url             string   `json:"url,omitempty"`
	Description     string   `json:"description,omitempty"`
	Topics          []string `json:"topics,omitempty"`
	WatchersCount   int      `json:"watchers_count,omitempty"`
	StargazersCount int      `json:"stargazers_count,omitempty"`
	ForksCount      int      `json:"forks_count,omitempty"`
	StarredAt       JsonTime `json:"starred_at,omitempty"`
	CreatedAt       JsonTime `json:"created_at,omitempty"`
	UpdateAt        JsonTime `json:"updated_at,omitempty"`
	PushedAt        JsonTime `json:"pushed_at,omitempty"`
}

type User struct {
	Owner     string   `json:"owner,omitempty"`
	AvatarURL string   `json:"avatar_url,omitempty"`
	Url       string   `json:"url,omitempty"`
	Bio       string   `json:"bio,omitempty"`
	CreatedAt JsonTime `json:"created_at,omitempty"`
	UpdatedAt JsonTime `json:"updated_at,omitempty"`
}

type JsonTime struct {
	time.Time
}

// MarshalJSON converts struct to json bytes.
func (jt *JsonTime) MarshalJSON() ([]byte, error) {
	s := jt.Format(time.RFC3339)
	return json.Marshal(s)
}

// UnmarshalJSON converts bytes to struct.
func (jt *JsonTime) UnmarshalJSON(bys []byte) error {
	var s string
	if err := json.Unmarshal(bys, &s); err != nil {
		return err
	}

	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return err
	}
	jt.Time = t
	return nil
}

type wrapper struct {
	*github.Client
}

// User returns github user object.
func (w *wrapper) User() (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user, _, err := w.Users.Get(ctx, "")
	switch {
	case err == nil:
		if user == nil {
			return nil, fmt.Errorf("[err] User %w", ErrNotFound)
		}
		return &User{
			Owner:     user.GetLogin(),
			AvatarURL: user.GetAvatarURL(),
			Url:       user.GetHTMLURL(),
			Bio:       user.GetBio(),
			CreatedAt: JsonTime{user.GetCreatedAt().Time},
			UpdatedAt: JsonTime{user.GetUpdatedAt().Time},
		}, nil
	case errors.As(err, &githubRateLimit):
		return nil, fmt.Errorf("[err] NewGit %w", ErrApiQuotaExceed)
	default:
		return nil, fmt.Errorf("[err] NewGit %w", err)
	}
}

// ListStarredAll returns all of starred projects.
func (w *wrapper) ListStarredAll() ([]*Starred, error) {
	var repos []*github.StarredRepository
	page := 1
	for {
		paging, resp, err := w.listStarredPaging(page, perPage)

		// check whether raised error or not.
		switch {
		case errors.As(err, &githubRateLimit):
			return nil, fmt.Errorf("[err] ListStarredAll %w", ErrApiQuotaExceed)
		case err != nil:
			return nil, fmt.Errorf("[err] ListStarredAll %w", err)
		}

		// append repos
		repos = append(repos, paging...)
		if page >= resp.LastPage {
			break
		}
		page += 1
	}

	var starred []*Starred
	for _, star := range repos {
		starred = append(starred, &Starred{
			Owner: star.Repository.Owner.GetLogin(), Repo: star.Repository.GetName(),
			FullName: star.Repository.GetFullName(), Url: star.Repository.GetHTMLURL(),
			Description: star.GetRepository().GetDescription(), Topics: star.GetRepository().Topics,
			WatchersCount: star.GetRepository().GetWatchersCount(), StargazersCount: star.GetRepository().GetStargazersCount(),
			ForksCount: star.GetRepository().GetForksCount(), StarredAt: JsonTime{star.GetStarredAt().Time},
			CreatedAt: JsonTime{star.GetRepository().GetCreatedAt().Time}, UpdateAt: JsonTime{star.GetRepository().GetUpdatedAt().Time},
			PushedAt: JsonTime{star.GetRepository().GetPushedAt().Time},
		})
	}

	return starred, nil
}

// ListReadme returns readme list.
func (w *wrapper) ListReadme(owners []string, repos []string) ([]*Readme, error) {
	if len(owners) == 0 || len(repos) == 0 || len(owners) != len(repos) {
		return nil, fmt.Errorf("[err] GetMultiReadme %w", ErrInvalidParam)
	}

	total := len(owners)
	queueSize := total/parallelSize + parallelSize

	wg := sync.WaitGroup{}
	var multiQueue []chan *Readme

	for i := 0; i < parallelSize; i++ {
		wg.Add(1)
		ch := make(chan *Readme, queueSize)
		multiQueue = append(multiQueue, ch)
		go func(queue chan *Readme) {
			for r := range queue {
				r.Content, r.Err = w.getReadme(r.Owner, r.Repo)
			}
			wg.Done()
		}(ch)
	}

	// distributes items
	var readmeList []*Readme
	for i, _ := range owners {
		hole := i % parallelSize
		readme := &Readme{Owner: owners[i], Repo: repos[i]}
		readmeList = append(readmeList, readme)
		multiQueue[hole] <- readme
	}

	// close channel
	for _, ch := range multiQueue {
		close(ch)
	}
	wg.Wait()

	return readmeList, nil
}

func (w *wrapper) getReadme(owner, repo string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	readme, _, err := w.Repositories.GetReadme(ctx, owner, repo, nil)
	if err != nil {
		return "", fmt.Errorf("[err] GetReadme %w", err)
	}

	content, err := readme.GetContent()
	if err != nil {
		return "", fmt.Errorf("[err] GetReadme %w", err)
	}

	return content, nil
}

func (w *wrapper) listStarredPaging(page, perPage int) ([]*github.StarredRepository, *github.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	opt := &github.ActivityListStarredOptions{}
	opt.Page = page
	opt.PerPage = perPage

	return w.Activity.ListStarred(ctx, "", opt)
}
