// Package object implements Git objects.
package object

import (
	"bufio"
	"compress/zlib"
	"fmt"
	"os"
	"strconv"

	"github.com/pkg/errors"
	"github.com/sboehler/got/pkg/repository"
)

// Object represents an object.
type Object interface {
	Serialize() []byte
	Deserialize([]byte) error
}

// Load loads an object from the repository.
func Load(r *repository.Repository, sha string) (Object, error) {
	f, err := os.Open(r.GitPath("objects", sha[:2], sha[2:]))
	if err != nil {
		return nil, errors.Wrapf(err, "error loading object %s", sha)
	}
	zr, err := zlib.NewReader(f)
	if err != nil {
		return nil, err
	}
	br := bufio.NewReader(zr)
	ot, err := br.ReadBytes(0x20)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't read object type")
	}
	ot = ot[:len(ot)-1]
	size, err := br.ReadBytes(0x00)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't read object size")
	}
	_, err = strconv.ParseInt(string(size[:len(size)-1]), 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "invalid size")
	}
	switch s := string(ot[:len(ot)-1]); s {
	// case "commit":
	// 	return nil, nil
	// case "tree":
	// 	return nil, nil
	// case "blob":
	// 	return nil, nil
	// case "tag":
	// 	return nil, nil
	default:
		return nil, fmt.Errorf("invalid object tag: %q", s)
	}
}
