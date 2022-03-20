// Package repository implements functionality for git repositories.
package repository

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/natefinch/atomic"
	"github.com/pkg/errors"
	"github.com/sboehler/got/pkg/object"

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

// Object represents an object.
type Object interface {
	Serialize() []byte
	Deserialize([]byte) error
}

// LoadObject loads an object from the repository.
func (r *Repository) LoadObject(sha string, objectType string) (Object, error) {
	f, err := os.Open(r.GitPath("objects", sha[:2], sha[2:]))
	if err != nil {
		return nil, errors.Wrapf(err, "error loading object %s", sha)
	}
	defer f.Close()
	zr, err := zlib.NewReader(f)
	if err != nil {
		return nil, err
	}
	of, err := ReadObjectFile(bufio.NewReader(zr))
	if of.ObjectType != objectType {
		return nil, fmt.Errorf("wrong object type %s, want %s", of.ObjectType, objectType)
	}
	switch of.ObjectType {
	case "blob":
		return object.NewBlob(of.Data), nil
	default:
		return nil, fmt.Errorf("unsupported object type %s", of.ObjectType)
	}
}

// WriteObject writes the given object to the repository.
func (r *Repository) WriteObject(of *ObjectFile) (string, error) {
	var (
		buf bytes.Buffer
		w   = zlib.NewWriter(&buf)
	)
	if _, err := of.Write(w); err != nil {
		return "", err
	}
	w.Close()
	hash := Hash(of)
	f := r.GitPath("objects", hash[:2], hash[2:])
	err := atomic.WriteFile(f, &buf)
	return hash, errors.Wrapf(err, "error writing object %s", hash)
}

// Hash hashes the object.
func Hash(of *ObjectFile) string {
	hasher := sha1.New()
	of.Write(hasher)
	return hex.EncodeToString(hasher.Sum(nil))
}

// Find resolves the given object reference.
func (r *Repository) Find(name string, ot string, follow bool) string {
	return name
}

// ObjectFile defines the wire format for storing objects in the repository.
type ObjectFile struct {
	ObjectType string
	Data       []byte
}

var validObjectType = map[string]struct{}{
	"blob": {},
}

// ReadObjectFile reads an object file from a reader.
func ReadObjectFile(r *bufio.Reader) (*ObjectFile, error) {
	bs, err := r.ReadBytes(0x20)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't read object type")
	}
	ot := string(bs[:len(bs)-1])
	if _, ok := validObjectType[ot]; !ok {
		return nil, fmt.Errorf("invalid object type %s", ot)
	}
	bs, err = r.ReadBytes(0x00)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't read object size")
	}
	size, err := strconv.ParseInt(string(bs[:len(bs)-1]), 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "invalid size")
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't read data")
	}
	if int64(len(data)) != size {
		return nil, fmt.Errorf("len(data) == %d, want %d", len(data), size)
	}
	return &ObjectFile{
		ObjectType: ot,
		Data:       data,
	}, nil
}

func (of *ObjectFile) Write(w io.Writer) (int64, error) {
	var (
		total int64
		n     int
		err   error
	)
	n, err = io.WriteString(w, of.ObjectType)
	total += (int64)(n)
	if err != nil {
		return total, err
	}
	n, err = w.Write([]byte{0x20})
	total += (int64)(n)
	if err != nil {
		return total, err
	}
	n, err = io.WriteString(w, strconv.FormatInt(int64(len(of.Data)), 10))
	total += (int64)(n)
	if err != nil {
		return total, err
	}
	n, err = w.Write([]byte{0x00})
	total += (int64)(n)
	if err != nil {
		return total, err
	}
	n64, err := io.Copy(w, bytes.NewReader(of.Data))
	total += n64
	if err != nil {
		return total, err
	}
	return total, nil
}
