// Package repository implements functionality for git repositories.
package repository

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"gopkg.in/ini.v1"
)

// Repository represents a git repository.
type Repository struct {
	Worktree string
	GitDir   string
	Config   *ini.File
}

const dirperms = 0775

// Init initializes a new got repository.
func Init(path string) (*Repository, error) {
	if s, err := os.Stat(path); err == nil {
		if !s.IsDir() {
			return nil, fmt.Errorf("%s is not a directory", path)
		}
		if fs, err := ioutil.ReadDir(path); err != nil {
			return nil, errors.Wrap(err, "could not read repository path")
		} else if len(fs) > 0 {
			return nil, fmt.Errorf("%s is not empty", path)
		}
	} else {
		if err := os.Mkdir(path, dirperms); err != nil {
			return nil, errors.Wrap(err, "could not create repository")
		}
	}
	// path exists and is empty
	for _, subdir := range [][]string{
		{"branches"},
		{"objects"},
		{"refs", "tags"},
		{"refs", "heads"},
	} {
		if err := os.MkdirAll(repoPath(path, subdir...), dirperms); err != nil {
			return nil, err
		}
	}

	desc, err := os.Create(repoPath(path, "description"))
	if err != nil {
		return nil, err
	}
	defer desc.Close()
	desc.WriteString("Unnamed repository; edit this file 'description' to name the repository.\n")

	head, err := os.Create(repoPath(path, "HEAD"))
	if err != nil {
		return nil, err
	}
	defer head.Close()
	head.WriteString("ref: refs/heads/master\n")

	if err := defaultConfig().SaveTo(repoPath(path, "config")); err != nil {
		return nil, err
	}

	return nil, nil
}

func repoPath(path string, segments ...string) string {
	return filepath.Join(append([]string{path, ".git"}, segments...)...)
}

func defaultConfig() *ini.File {
	f := ini.Empty()
	core := f.Section("core")
	core.Key("repositoryformatversion").SetValue("0")
	core.Key("filemode").SetValue("false")
	core.Key("bare").SetValue("false")
	return f
}
