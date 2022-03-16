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
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, errors.Wrap(err, "invalid path")
	}
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

	config := defaultConfig()
	if err := config.SaveTo(repoPath(path, "config")); err != nil {
		return nil, err
	}

	return &Repository{
		Worktree: path,
		GitDir:   repoPath(path),
		Config:   config,
	}, nil
}

// Load loads the repository at path.
func Load(path string) (*Repository, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, errors.Wrap(err, "invalid path")
	}
	config, err := ini.Load(repoPath(path, "config"))
	if err != nil {
		return nil, err
	}
	return &Repository{
		Worktree: path,
		GitDir:   repoPath(path),
		Config:   config,
	}, nil
}

func repoPath(path string, segments ...string) string {
	return filepath.Join(append([]string{path, ".git"}, segments...)...)
}

// Find loads the repository at path or any of its parent directories.
func Find(path string) (*Repository, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, errors.Wrap(err, "invalid path")
	}
	gitPath := filepath.Join(path, ".git")
	if s, err := os.Stat(gitPath); err != nil && s.IsDir() {
		config, err := ini.Load(repoPath(path, "config"))
		if err != nil {
			return nil, err
		}
		return &Repository{
			Worktree: path,
			GitDir:   repoPath(path),
			Config:   config,
		}, nil
	}
	parent, err := filepath.Abs(filepath.Join(path, ".."))
	if err != nil {
		return nil, err
	}
	if parent == path {
		return nil, fmt.Errorf("could not find parent git directory")
	}
	return Find(parent)
}

func defaultConfig() *ini.File {
	f := ini.Empty()
	core := f.Section("core")
	core.Key("repositoryformatversion").SetValue("0")
	core.Key("filemode").SetValue("false")
	core.Key("bare").SetValue("false")
	return f
}
