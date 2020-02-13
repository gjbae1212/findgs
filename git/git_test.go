package git

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	assert := assert.New(t)

	tests := map[string]struct {
		token string
		isErr bool
	}{
		"success": {token: os.Getenv("GITHUB_TOKEN")},
		"fail":    {token: "", isErr: true},
	}

	for _, t := range tests {
		_, err := NewGit(t.token)
		assert.Equal(t.isErr, err != nil)
	}
}

func TestMain(m *testing.M) {
	if os.Getenv("GITHUB_TOKEN") != "" {
		os.Exit(m.Run())
	}
}
