package git

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fatih/color"
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
	Readme          string   `json:"readme,omitempty"`
	CachedAt        JsonTime `json:"cached_at,omitempty"`
	Error           error    `json:"-"`
}

type User struct {
	Owner     string   `json:"owner,omitempty"`
	AvatarURL string   `json:"avatar_url,omitempty"`
	Url       string   `json:"url,omitempty"`
	Bio       string   `json:"bio,omitempty"`
	CreatedAt JsonTime `json:"created_at,omitempty"`
	UpdatedAt JsonTime `json:"updated_at,omitempty"`
	Token     string   `json:"token"`
	CachedAt  JsonTime `json:"cached_at"`
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
	token string
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
			CachedAt:  JsonTime{time.Now()},
			Token:     w.token,
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
	initPage := 1
	lastPage := 1

	// first requests
	paging, resp, err := w.listStarredPaging(initPage, perPage)
	if err != nil {
		if errors.As(err, &githubRateLimit) {
			return nil, fmt.Errorf("[err] ListStarredAll %w", ErrApiQuotaExceed)
		}
		return nil, fmt.Errorf("[err] ListStarredAll %w", err)
	}
	// getting last page.
	lastPage = resp.LastPage
	// append repos
	repos = append(repos, paging...)
	if initPage >= resp.LastPage {
	} else {
		lock := &sync.Mutex{}
		wg := sync.WaitGroup{}
		var multiQueue []chan int
		var raisedErrors []error
		for i := 0; i < parallelSize; i++ {
			wg.Add(1)
			ch := make(chan int, lastPage/parallelSize+parallelSize)
			multiQueue = append(multiQueue, ch)
			go func(queue chan int) {
				for r := range queue {
					paging, _, err := w.listStarredPaging(r, perPage)
					if err != nil {
						switch {
						case errors.As(err, &githubRateLimit):
							color.Red("[fail] getting github page %d %s", r, ErrApiQuotaExceed)
						default:
							color.Red("[fail] getting github page %d %s", r, err.Error())
						}
						raisedErrors = append(raisedErrors, err)
						continue
					}
					// race condition.
					lock.Lock()
					repos = append(repos, paging...)
					lock.Unlock()
				}
				wg.Done()
			}(ch)
		}

		// send requests
		for i := initPage + 1; i <= lastPage; i++ {
			hole := i % parallelSize
			multiQueue[hole] <- i
		}

		// close channel
		for _, ch := range multiQueue {
			close(ch)
		}

		wg.Wait()
		if len(raisedErrors) != 0 {
			return nil, fmt.Errorf("[err] ListStarredAll error count %d", len(raisedErrors))
		}
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
			PushedAt: JsonTime{star.GetRepository().GetPushedAt().Time}, CachedAt: JsonTime{time.Now()},
		})
	}

	return starred, nil
}

// SetReadme sets readme to starred.
func (w *wrapper) SetReadme(starred []*Starred) {
	if len(starred) == 0 {
		return
	}

	total := len(starred)
	queueSize := total/parallelSize + parallelSize

	wg := sync.WaitGroup{}
	var multiQueue []chan *Starred

	for i := 0; i < parallelSize; i++ {
		wg.Add(1)
		ch := make(chan *Starred, queueSize)
		multiQueue = append(multiQueue, ch)
		go func(queue chan *Starred) {
			for r := range queue {
				w.setReadmeToStarred(r)
			}
			wg.Done()
		}(ch)
	}

	// distributes items
	for i, star := range starred {
		hole := i % parallelSize
		multiQueue[hole] <- star
	}

	// close channel
	for _, ch := range multiQueue {
		close(ch)
	}
	wg.Wait()
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
		return "", fmt.Errorf("[err] getReadme %w", err)
	}

	content, err := readme.GetContent()
	if err != nil {
		return "", fmt.Errorf("[err] getReadme %w", err)
	}

	return content, nil
}

func (w *wrapper) setReadmeToStarred(s *Starred) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	readme, _, err := w.Repositories.GetReadme(ctx, s.Owner, s.Repo, nil)
	if err != nil {
		s.Error = fmt.Errorf("[err] setReadmeToStarred %w", err)
		return
	}

	content, err := readme.GetContent()
	if err != nil {
		s.Error = fmt.Errorf("[err] setReadmeToStarred %w", err)
		return
	}

	s.Readme = content
	return
}

func (w *wrapper) listStarredPaging(page, perPage int) ([]*github.StarredRepository, *github.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	opt := &github.ActivityListStarredOptions{}
	opt.Page = page
	opt.PerPage = perPage

	return w.Activity.ListStarred(ctx, "", opt)
}
