package git

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
)

func TestModel_MarshalAndUnMarsahlJSON(t *testing.T) {
	assert := assert.New(t)

	// User
	user := &User{
		Owner:     "owner",
		AvatarURL: "avata",
		Url:       "url",
		Bio:       "Bio",
		CreatedAt: JsonTime{time.Now()},
		UpdatedAt: JsonTime{time.Now()},
	}
	userData, err := json.Marshal(user)
	assert.NoError(err)
	log.Println(string(userData))

	var ruser *User
	err = json.Unmarshal(userData, &ruser)
	assert.NoError(err)
	assert.Equal(user.Owner, ruser.Owner)
	assert.Equal(user.AvatarURL, ruser.AvatarURL)
	assert.Equal(user.Url, ruser.Url)
	assert.Equal(user.Bio, ruser.Bio)
	assert.Equal(user.CreatedAt.Unix(), ruser.CreatedAt.Unix())
	assert.Equal(user.UpdatedAt.Unix(), ruser.UpdatedAt.Unix())

	// Starred
	starred := &Starred{
		Owner:           "owner",
		Repo:            "repo",
		FullName:        "full_name",
		Url:             "url",
		Description:     "description",
		Topics:          []string{"allan", "hello"},
		WatchersCount:   10,
		StargazersCount: 20,
		ForksCount:      30,
		StarredAt:       JsonTime{time.Now()},
		CreatedAt:       JsonTime{time.Now()},
		UpdateAt:        JsonTime{time.Now()},
		PushedAt:        JsonTime{time.Now()},
	}
	starredData, err := json.Marshal(starred)
	assert.NoError(err)
	log.Println(string(starredData))

	var rstarred *Starred
	err = json.Unmarshal(starredData, &rstarred)
	assert.NoError(err)
	assert.Equal(starred.Owner, rstarred.Owner)
	assert.Equal(starred.Repo, rstarred.Repo)
	assert.Equal(starred.FullName, rstarred.FullName)
	assert.Equal(starred.Url, rstarred.Url)
	assert.Equal(starred.Description, rstarred.Description)
	assert.Equal(starred.Topics, rstarred.Topics)
	assert.Equal(starred.WatchersCount, rstarred.WatchersCount)
	assert.Equal(starred.StargazersCount, rstarred.StargazersCount)
	assert.Equal(starred.ForksCount, rstarred.ForksCount)
	assert.Equal(starred.StarredAt.Unix(), rstarred.StarredAt.Unix())
	assert.Equal(starred.CreatedAt.Unix(), rstarred.CreatedAt.Unix())
	assert.Equal(starred.UpdateAt.Unix(), rstarred.UpdateAt.Unix())
	assert.Equal(starred.PushedAt.Unix(), rstarred.PushedAt.Unix())

	// Readme
	readme := &Readme{
		Owner:   "owner",
		Repo:    "repo",
		Content: "allan-allala",
		Err:     fmt.Errorf("eerrrr"),
	}

	readmeData, err := json.Marshal(readme)
	assert.NoError(err)
	log.Println(string(readmeData))

	var rreadme *Readme
	err = json.Unmarshal(readmeData, &rreadme)
	assert.NoError(err)
	assert.Equal(readme.Owner, rreadme.Owner)
	assert.Equal(readme.Repo, rreadme.Repo)
	assert.Equal(readme.Content, rreadme.Content)
	assert.Empty(rreadme.Err)
}

func TestJsonTime_MarshalJSON(t *testing.T) {
	assert := assert.New(t)

	tests := map[string]struct {
		input JsonTime
		isErr bool
	}{
		"empty":   {input: JsonTime{}},
		"success": {input: JsonTime{time.Now()}},
	}

	for _, t := range tests {
		data, err := json.Marshal(t.input)
		assert.Equal(t.isErr, err != nil)
		if err == nil {
			log.Println(string(data))
		}
	}

}

func TestJsonTime_UnmarshalJSON(t *testing.T) {
	assert := assert.New(t)
	empty, err := json.Marshal(JsonTime{})
	assert.NoError(err)
	success, err := json.Marshal(JsonTime{time.Now()})
	assert.NoError(err)

	tests := map[string]struct {
		input []byte
		isErr bool
	}{
		"empty":   {input: empty},
		"success": {input: success},
	}

	for _, t := range tests {
		var data JsonTime
		err := json.Unmarshal(t.input, &data)
		assert.Equal(t.isErr, err != nil)
		if err == nil {
			log.Println(data.Unix())
		}
	}
}

func TestWrapper_User(t *testing.T) {
	assert := assert.New(t)

	sgit, err := NewGit(os.Getenv("GITHUB_TOKEN"))
	assert.NoError(err)
	fgit, err := NewGit("invalid token")
	assert.NoError(err)

	tests := map[string]struct {
		input Git
		isErr bool
	}{
		"success":       {input: sgit, isErr: false},
		"invalid-token": {input: fgit, isErr: true},
	}

	for _, t := range tests {
		_, err := t.input.User()
		assert.Equal(t.isErr, err != nil)
	}
}

func TestWrapper_ListStarredAll(t *testing.T) {
	assert := assert.New(t)

	g, err := NewGit(os.Getenv("GITHUB_TOKEN"))
	assert.NoError(err)

	tests := map[string]struct {
		isErr bool
	}{
		"success": {},
	}

	for _, t := range tests {
		result, err := g.ListStarredAll()
		assert.Equal(t.isErr, err != nil)
		if err == nil {
			total := map[string]string{}
			for _, star := range result {
				total[star.FullName] = ""
			}
			color.Green("Total Starred %d %d", len(result), len(total))
		}
	}
}

func TestWrapper_ListReadme(t *testing.T) {
	assert := assert.New(t)

	g, err := NewGit(os.Getenv("GITHUB_TOKEN"))
	assert.NoError(err)

	starred, err := g.ListStarredAll()
	assert.NoError(err)

	var owners []string
	var repos []string
	for _, star := range starred {
		owners = append(owners, star.Owner)
		repos = append(repos, star.Repo)
	}

	tests := map[string]struct {
		owners []string
		repos  []string
		size   int
		isErr  bool
	}{
		"fail":    {isErr: true},
		"success": {owners: owners, repos: repos, size: len(owners)},
	}

	for _, t := range tests {
		readmeList, err := g.ListReadme(t.owners, t.repos)
		assert.Equal(t.isErr, err != nil)
		if err == nil {
			assert.Len(readmeList, t.size)
			errorCount := 0
			successCount := 0
			for _, readme := range readmeList {
				if readme.Err != nil {
					errorCount++
				} else {
					successCount++
				}
			}
			color.Blue("Readme success %d, Readme error %d", successCount, errorCount)
		}
	}
}

func TestWrapper_SetReadme(t *testing.T) {
	assert := assert.New(t)

	g, err := NewGit(os.Getenv("GITHUB_TOKEN"))
	assert.NoError(err)

	starred, err := g.ListStarredAll()
	assert.NoError(err)

	tests := map[string]struct {
		starred []*Starred
		size    int
		isErr   bool
	}{
		"fail":    {isErr: true},
		"success": {starred: starred, size: len(starred)},
	}

	for _, t := range tests {
		g.SetReadme(t.starred)
		fail := 0
		success := 0
		for _, star := range t.starred {
			if star.Error != nil {
				fail++
			} else if star.Readme != "" {
				success++
			}
		}
		assert.Equal(fail+success, t.size)
		color.Blue("Readme success %d, Readme error %d", success, fail)
	}
}
