package search

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/blevesearch/bleve"
	"github.com/boltdb/bolt"
	"github.com/fatih/color"
	"github.com/gjbae1212/findgs/git"
	"github.com/mitchellh/go-homedir"
)

const (
	dbFileName          = "cache.db"
	userBucketName      = "user"
	starredBucketSuffix = "starred"
)

var (
	ErrInvalidParam = errors.New("[err] Invalid param")

	configPath string
	configOnce sync.Once
	configErr  error
)

type Searcher interface {
	CreateIndex() error
	Search(text string, size int) ([]*Result, error)
}

type searcher struct {
	gitToken string
	dbPath   string
	git      git.Git
	db       *bolt.DB
	index    bleve.Index
}

type Result struct {
	*git.Starred
	Score float64
}

func ConfigPath() (string, error) {
	configOnce.Do(func() {
		// getting home directory.
		home, err := homedir.Dir()
		if err != nil {
			configErr = fmt.Errorf("[err] ConfigPath %w", err)
			return
		}

		// make config directory
		path := filepath.Join(home, ".findgs")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if suberr := os.MkdirAll(path, os.ModePerm); suberr != nil {
				configErr = fmt.Errorf("[err] NewSearcher %w", suberr)
				return
			}
		}
		configPath = path
	})
	return configPath, configErr
}

// NewSearcher returns an object implemented Searcher.
func NewSearcher(token string) (Searcher, error) {
	if token == "" {
		return nil, fmt.Errorf("[err] NewSearcher %w", ErrInvalidParam)
	}

	cfgPath, err := ConfigPath()
	if err != nil {
		return nil, fmt.Errorf("[err] NewSearcher %w", err)
	}
	dbPath := filepath.Join(cfgPath, dbFileName)

	// make git client
	git, err := git.NewGit(token)
	if err != nil {
		return nil, fmt.Errorf("[err] NewSearcher %w", err)
	}

	// make bolt db
	db, err := bolt.Open(dbPath, os.ModePerm, &bolt.Options{Timeout: 2 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("[err] NewSearcher fail db %w", err)
	}

	// make index
	index, err := bleve.NewMemOnly(bleve.NewIndexMapping())
	if err != nil {
		return nil, fmt.Errorf("[err] NewSearcher fail db %w", err)
	}

	return &searcher{git: git, db: db, index: index, gitToken: token, dbPath: dbPath}, nil
}

// Search executes full text search.
func (s *searcher) Search(text string, size int) ([]*Result, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return []*Result{}, nil
	}

	// search text from index.
	query := bleve.NewMatchQuery(text)
	search := bleve.NewSearchRequestOptions(query, size, 0, false)
	search.SortBy([]string{"-_score", "_id"})
	searchResult, err := s.index.Search(search)
	if err != nil {
		return nil, fmt.Errorf("[err] Search %w", err)
	}

	// get a detailed starred information
	var list []*Result
	s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(starredBucketName(s.gitToken)))
		if bucket == nil {
			return nil
		}
		for _, r := range searchResult.Hits {
			data := bucket.Get([]byte(r.ID))
			var starred *git.Starred
			if err := json.Unmarshal(data, &starred); err == nil {
				list = append(list, &Result{Starred: starred, Score: r.Score})
			}
		}
		return nil
	})

	return list, nil
}

// CreateIndex makes bleve.Index
func (s *searcher) CreateIndex() error {
	// get user
	user, reload, err := s.getUser()
	if err != nil {
		return fmt.Errorf("[err] createIndex %w", err)
	}

	// check to whether exist starred items or not.
	var isNewIndex bool
	if err := s.db.Update(func(tx *bolt.Tx) error {
		var err error
		bucket := tx.Bucket([]byte(starredBucketName(s.gitToken)))
		if bucket == nil {
			bucket, err = tx.CreateBucket([]byte(starredBucketName(s.gitToken)))
			if err != nil {
				return err
			}
			isNewIndex = true
		} else {
			isNewIndex = false
		}
		return nil
	}); err != nil {
		os.RemoveAll(s.dbPath)
		color.Yellow("[err] collapse db file, so delete db file")
		return fmt.Errorf("[err] createIndex %w", err)
	}

	// read old database.
	var oldStarredList []*git.Starred
	oldStarredMap := map[string]*git.Starred{}
	if !isNewIndex {
		// read old starred from db
		s.db.View(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(starredBucketName(s.gitToken)))
			bucket.ForEach(func(k, v []byte) error {
				var starred *git.Starred
				if err := json.Unmarshal(v, &starred); err != nil {
					color.Yellow("[err] parsing %s", string(k))
				} else {
					oldStarredList = append(oldStarredList, starred)
					oldStarredMap[starred.FullName] = starred
				}
				return nil
			})
			return nil
		})

		// write old starred to index
		for _, starred := range oldStarredList {
			if err := s.index.Index(starred.FullName, starred); err != nil {
				color.Yellow("[err] indexing %s", starred.FullName)
			}
		}
	}

	// are you all ready?
	if !reload && !isNewIndex {
		count, _ := s.index.DocCount()
		color.Green("[success][using cache] %d items", count)
		return nil
	}

	// reload new starred list.
	newStarredList, err := s.git.ListStarredAll()
	if err != nil {
		color.Yellow("[err] don't getting starred list %s", err.Error())
		if !isNewIndex {
			count, _ := s.index.DocCount()
			color.Yellow("[fail][using cache] %d items", count)
			return nil
		}
		return fmt.Errorf("[err] CreateIndex %w", err)
	}
	newStarredMap := map[string]*git.Starred{}
	for _, starred := range newStarredList {
		newStarredMap[starred.FullName] = starred
	}

	// update and insert
	if isNewIndex {
		color.Blue("[refresh] all repositories")
		s.git.SetReadme(newStarredList)
		s.writeDBAndIndex(newStarredList)
	} else {

		// insert or update starred
		var insertList []*git.Starred
		var updateList []*git.Starred
		for _, newStarred := range newStarredList {
			if oldStarred, ok := oldStarredMap[newStarred.FullName]; !ok {
				insertList = append(insertList, newStarred)
				color.Blue("[insert] %s repository pushed_at %s",
					newStarred.FullName, newStarred.PushedAt.Format(time.RFC3339))
			} else {
				if oldStarred.PushedAt.Unix() != newStarred.PushedAt.Unix() {
					updateList = append(updateList, newStarred)
					color.Blue("[update] %s repository pushed_at %s",
						newStarred.FullName, newStarred.PushedAt.Format(time.RFC3339))
				}
			}
		}

		// insert
		s.git.SetReadme(insertList)
		s.writeDBAndIndex(insertList)

		// update
		s.git.SetReadme(updateList)
		s.writeDBAndIndex(updateList)

		// delete starred
		var deleteList []*git.Starred
		for _, oldStarred := range oldStarredList {
			if _, ok := newStarredMap[oldStarred.FullName]; !ok {
				deleteList = append(deleteList, oldStarred)
				color.Blue("[delete] %s repository pushed_at %s",
					oldStarred.FullName, oldStarred.PushedAt.Format(time.RFC3339))
			}
		}
		// delete
		s.deleteDBAndIndex(deleteList)
	}

	// rewrite a user to db
	userData, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("[err] createIndex %w", err)
	}
	s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(userBucketName))
		bucket.Put([]byte(s.gitToken), userData)
		return nil
	})

	count, _ := s.index.DocCount()
	color.Green("[success][new reload] %d items", count)
	return nil
}

// getUserInfo returns a user information and reload flag.
func (s *searcher) getUser() (user *git.User, reload bool, err error) {
	// read a user from database.
	var userData []byte
	suberr := s.db.Update(func(tx *bolt.Tx) error {
		var inerr error
		bucket := tx.Bucket([]byte(userBucketName))
		if bucket == nil {
			bucket, inerr = tx.CreateBucket([]byte(userBucketName))
			if inerr != nil {
				return inerr
			}
		}
		userData = bucket.Get([]byte(s.gitToken))
		return nil
	})
	if suberr != nil { // maybe collapse db file.
		os.RemoveAll(s.dbPath)
		color.Yellow("[err] collapse db file, so delete db file")
		err = fmt.Errorf("[err] getUser %w", suberr)
		return
	}

	// if a user doesn't exist.
	if userData == nil || len(userData) == 0 {
		newUser, suberr := s.git.User()
		if suberr != nil {
			err = fmt.Errorf("[err] createIndex %w", suberr)
			return
		}
		user = newUser
		reload = true
		return
	}

	// unmarshal user
	if suberr := json.Unmarshal(userData, &user); suberr != nil {
		color.Yellow("[err] collapse user data, so delete user data")
		color.Red("[err] retry again!")
		s.db.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(userBucketName))
			b.Delete([]byte(s.gitToken))
			return nil
		})
		err = fmt.Errorf("[err] createIndex %w", suberr)
		return
	}

	// check whether reload or not.
	if user.CachedAt.Unix() < time.Now().Add(-1*time.Hour).Unix() {
		reload = true
		newUser, suberr := s.git.User()
		if suberr != nil {
			color.Yellow("[err] a user doesn't reload %s", suberr.Error())
		} else {
			user = newUser
		}
	}
	return
}

func (s *searcher) writeDBAndIndex(starredList []*git.Starred) error {
	if len(starredList) == 0 {
		return nil
	}
	// write db
	s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(starredBucketName(s.gitToken)))
		for _, starred := range starredList {
			if starred.Error != nil {
				color.Yellow("[err][db write] don't found readme data %s", starred.FullName)
				continue
			}
			bys, err := json.Marshal(starred)
			if err != nil {
				color.Yellow("[err][db write] don't parse bytes %s", starred.FullName)
				continue
			}
			if err := bucket.Put([]byte(starred.FullName), bys); err != nil {
				color.Yellow("[err][db write] don't put %s", starred.FullName)
				continue
			}
		}
		return nil
	})
	// write index
	for _, starred := range starredList {
		if starred.Error != nil {
			color.Yellow("[err][index write] don't found readme data %s", starred.FullName)
			continue
		}
		if err := s.index.Index(starred.FullName, starred); err != nil {
			color.Yellow("[err][index write] don't put %s", starred.FullName)
			continue
		}
	}
	return nil
}

func (s *searcher) deleteDBAndIndex(starredList []*git.Starred) error {
	if len(starredList) == 0 {
		return nil
	}
	// delete db
	s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(starredBucketName(s.gitToken)))
		for _, starred := range starredList {
			bucket.Delete([]byte(starred.FullName))
		}
		return nil
	})
	// delete index
	for _, starred := range starredList {
		s.index.Delete(starred.FullName)
	}
	return nil
}

func starredBucketName(token string) string {
	return token + "_" + starredBucketSuffix
}
