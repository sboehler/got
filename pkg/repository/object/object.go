// Package object implements Git objects.
package object

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
	"strconv"

	"github.com/natefinch/atomic"
	"github.com/pkg/errors"
	"github.com/sboehler/got/pkg/repository"
)

// Object represents an object.
type Object interface {
	Serialize() []byte
	Deserialize([]byte) error
	Type() string
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
	switch s := string(ot); s {
	case "blob":
		return NewBlob(br)
	// case "commit":
	// 	return nil, nil
	// case "tree":
	// 	return nil, nil
	// case "tag":
	// 	return nil, nil
	default:
		return nil, fmt.Errorf("invalid object tag: %q", s)
	}
}

// Write writes an object to the repository.
func Write(repo *repository.Repository, o Object) error {
	var typeID string
	switch t := o.(type) {
	case *Blob:
		typeID = "blob"
	default:
		return fmt.Errorf("unknown object type: %T", t)
	}
	data := o.Serialize()
	header := createHeader(typeID, data)

	hasher := sha1.New()
	hasher.Write(header)
	hasher.Write(data)
	hash := hex.EncodeToString(hasher.Sum(nil))

	f := repo.GitPath("objects", hash[:2], hash[2:])
	r := io.MultiReader(bytes.NewReader(header), bytes.NewReader(data))
	err := atomic.WriteFile(f, r)
	return errors.Wrapf(err, "error writing object %s", hash)
}

// createPayload returns the bytes to be written and the sha1 hash in
// hexadecimal format.
func createHeader(typeID string, data []byte) []byte {
	var b bytes.Buffer
	b.WriteString(typeID)
	b.WriteByte(0x20)
	b.WriteString(strconv.FormatInt(int64(len(data)), 10))
	b.WriteByte(0x00)
	return b.Bytes()
}

// Blob represents a blob.
type Blob struct {
	data []byte
}

func NewBlob(r io.Reader) (*Blob, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return &Blob{data}, nil
}

var _ Object = (*Blob)(nil)

// Deserialize implements Object.
func (b *Blob) Deserialize(bs []byte) error {
	b.data = bs
	return nil
}

// Serialize implements Object.
func (b *Blob) Serialize() []byte {
	return b.data
}

func (Blob) Type() string {
	return "blob"
}

// Find resolves the given object reference.
func Find(r *repository.Repository, name string, ot string, follow bool) string {
	return name
}
