package search

import (
	"encoding/json"
	"log"
	"os"
	"testing"
	"time"

	"github.com/blevesearch/bleve"
	"github.com/boltdb/bolt"
	"github.com/gjbae1212/findgs/git"
	"github.com/stretchr/testify/assert"
)

func TestNewSearcher(t *testing.T) {
	assert := assert.New(t)

	tests := map[string]struct {
		token string
		isErr bool
	}{
		"fail":    {token: "", isErr: true},
		"success": {token: "fake-token"},
	}

	for _, t := range tests {
		s, err := NewSearcher(t.token)
		assert.Equal(t.isErr, err != nil)
		if err == nil {
			s.(*searcher).db.Close()
		}
	}
}

func TestCreateIndex(t *testing.T) {
	assert := assert.New(t)

	token := os.Getenv("GITHUB_TOKEN")
	s, err := NewSearcher(token)
	assert.NoError(err)
	defer s.(*searcher).db.Close()

	tests := map[string]struct {
		isErr bool
	}{
		"success": {},
	}

	for _, t := range tests {
		err := s.CreateIndex()
		assert.Equal(t.isErr, err != nil)
		query := bleve.NewMatchQuery("docker")
		search := bleve.NewSearchRequestOptions(query, 10000, 0, true)
		search.SortBy([]string{"-_score", "_id"})
		searchResult, err := s.(*searcher).index.Search(search)
		for _, s := range searchResult.Hits {
			log.Println(s.ID, s.Score)
		}
	}
}

func TestGetUser(t *testing.T) {
	assert := assert.New(t)

	token := os.Getenv("GITHUB_TOKEN")
	s, err := NewSearcher(token)
	assert.NoError(err)
	defer s.(*searcher).db.Close()
	user, err := s.(*searcher).git.User()
	assert.NoError(err)

	tests := map[string]struct {
		reload bool
	}{
		"reload":     {reload: true},
		"not reload": {},
	}

	for k, t := range tests {
		switch k {
		case "reload":
			user.CachedAt = git.JsonTime{time.Now().Add(-2 * time.Hour)}
			userData, err := json.Marshal(user)
			assert.NoError(err)
			s.(*searcher).db.Update(func(tx *bolt.Tx) error {
				bucket := tx.Bucket([]byte(userBucketName))
				bucket.Put([]byte(s.(*searcher).gitToken), userData)
				return nil
			})
			result, reload, err := s.(*searcher).getUser()
			assert.NotEmpty(result)
			assert.NoError(err)
			assert.Equal(reload, t.reload)
		case "not reload":
			user.CachedAt = git.JsonTime{time.Now().Add(1 * time.Hour)}
			userData, err := json.Marshal(user)
			assert.NoError(err)
			s.(*searcher).db.Update(func(tx *bolt.Tx) error {
				bucket := tx.Bucket([]byte(userBucketName))
				bucket.Put([]byte(s.(*searcher).gitToken), userData)
				return nil
			})
			result, reload, err := s.(*searcher).getUser()
			assert.NotEmpty(result)
			assert.NoError(err)
			assert.Equal(reload, t.reload)
		}
	}

}

func TestCRUD_DBAndIndex(t *testing.T) {
	assert := assert.New(t)

	token := os.Getenv("GITHUB_TOKEN")
	s, err := NewSearcher(token)
	assert.NoError(err)
	defer s.(*searcher).db.Close()

	// write and delete
	tests := map[string]struct {
		starredList []*git.Starred
		isErr       bool
	}{
		"write and delete": {starredList: []*git.Starred{
			&git.Starred{Owner: "allan", Repo: "hello", FullName: "allan/hello"},
		}},
	}

	for _, t := range tests {
		err := s.(*searcher).writeDBAndIndex(t.starredList)
		assert.Equal(t.isErr, err != nil)
		s.(*searcher).db.View(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(starredBucketName(s.(*searcher).gitToken)))
			data := bucket.Get([]byte(t.starredList[0].FullName))
			assert.NotEmpty(data)
			return nil
		})

		err = s.(*searcher).deleteDBAndIndex(t.starredList)
		assert.Equal(t.isErr, err != nil)
		s.(*searcher).db.View(func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(starredBucketName(s.(*searcher).gitToken)))
			data := bucket.Get([]byte(t.starredList[0].FullName))
			assert.Empty(data)
			return nil
		})
	}

}

func TestMain(m *testing.M) {
	if os.Getenv("GITHUB_TOKEN") != "" {
		os.Exit(m.Run())
	}
}
