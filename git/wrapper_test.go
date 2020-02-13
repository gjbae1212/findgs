package git

import (
	"os"
	"testing"

	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
)

func TestWrapper_User(t *testing.T) {
	assert := assert.New(t)

	g, err := NewGit(os.Getenv("GITHUB_TOKEN"))
	assert.NoError(err)

	tests := map[string]struct {
		input Git
	}{
		"success": {input: g},
	}

	for _, t := range tests {
		assert.NotEmpty(t.input.User())
	}
}

func TestWrapper_ListStarredAll(t *testing.T) {
	assert := assert.New(t)

	g, err := NewGit(os.Getenv("GITHUB_TOKEN"))
	assert.NoError(err)

	tests := map[string]struct {
		retries int
		isErr   bool
	}{
		"success": {retries: 1},
	}

	for _, t := range tests {
		result, err := g.ListStarredAll(t.retries)
		assert.Equal(t.isErr, err != nil)
		if err == nil {
			yes := 0
			no := 0
			for _, repo := range result {
				if repo.Readme == "" {
					no++
				} else {
					yes++
				}
			}
			color.Green("Exist Readme: %d Not Readme: %d", yes, no)
		}
	}
}
