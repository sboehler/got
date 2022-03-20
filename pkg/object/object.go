// Package object implements Git objects.
package object

// Blob represents a blob.
type Blob struct {
	data []byte
}

// NewBlob creates a new blob.
func NewBlob(bs []byte) *Blob {
	return &Blob{bs}
}

// Deserialize implements Object.
func (b *Blob) Deserialize(bs []byte) error {
	b.data = bs
	return nil
}

// Serialize implements Object.
func (b *Blob) Serialize() []byte {
	return b.data
}
