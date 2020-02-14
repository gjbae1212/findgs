package search

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/boltdb/bolt"
	"github.com/gjbae1212/findgs/git"
	"github.com/mitchellh/go-homedir"
)

const (
	configFolderName = ".findgs"
)

var (
	ErrInvalidParam = errors.New("[err] Invalid param")
)

type Searcher interface{}

type searcher struct {
	gitClient git.Git
}

// NewSearcher returns an object implemented Searcher.
func NewSearcher(token string) (Searcher, error) {
	if token == "" {
		return nil, fmt.Errorf("[err] NewSearcher %w", ErrInvalidParam)
	}

	// create config folder(.findfs)
	home, err := homedir.Dir()
	if err != nil {
		return nil, fmt.Errorf("[err] NewSearcher %w", err)
	}
	configDir := filepath.Join(home, configFolderName)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		if suberr := os.MkdirAll(configDir, os.ModePerm); suberr != nil {
			return nil, fmt.Errorf("[err] NewSearcher %w", suberr)
		}
	}

	// TODO: open boltdb (initialize or if fail delete and initialize)

	return nil, nil
}
