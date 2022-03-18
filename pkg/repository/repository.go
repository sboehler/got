// Package repository implements functionality for git repositories.
package repository

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/natefinch/atomic"
	"github.com/pkg/errors"

	"gopkg.in/ini.v1"
)

// Repository represents a git repository.
type Repository struct {
	Worktree string
	GitDir   string
	Config   *ini.File
}

// GitPath returns the path to a file in the repository.
func (r *Repository) GitPath(ss ...string) string {
	return repoPath(r.Worktree, ss...)
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

	err = atomic.WriteFile(repoPath(path, "description"), strings.NewReader("Unnamed repository; edit this file 'description' to name the repository.\n"))
	if err != nil {
		return nil, errors.Wrapf(err, "error writing %s", repoPath(path, "description"))
	}

	err = atomic.WriteFile(repoPath(path, "HEAD"), strings.NewReader("ref: refs/heads/master\n"))
	if err != nil {
		return nil, errors.Wrapf(err, "error writing %s", repoPath(path, "HEAD"))
	}

	config := defaultConfig()
	var cb bytes.Buffer
	config.WriteTo(&cb)
	err = atomic.WriteFile(repoPath(path, "config"), &cb)
	if err != nil {
		return nil, errors.Wrapf(err, "error writing %s", repoPath(path, "config"))
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
	if s, err := os.Stat(gitPath); err == nil && s.IsDir() {
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
