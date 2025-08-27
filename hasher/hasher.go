package hasher

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"

	"github.com/cespare/xxhash/v2"
	"github.com/zeebo/blake3"
)

// Hash types constants.
const (
	HashMD5     = "MD5"
	HashSHA1    = "SHA1"
	HashSHA256  = "SHA256"
	HashXXHash  = "XXHASH64"
	HashBlake3  = "BLAKE3"
)

// Func is a function type that takes a reader and returns a hash string or an error.
type Func func(io.Reader) (string, error)

// GetHasher returns the appropriate hash function based on the hash type string.
func GetHasher(hashType string) (Func, error) {
	switch hashType {
	case HashMD5:
		return newHashStreamFunc(md5.New), nil
	case HashSHA1:
		return newHashStreamFunc(sha1.New), nil
	case HashSHA256:
		return newHashStreamFunc(sha256.New), nil
	case HashXXHash:
		return hashXXHashStream, nil
	case HashBlake3:
		return newHashStreamFunc(func() hash.Hash { return blake3.New() }), nil
	default:
		return nil, fmt.Errorf("unsupported hash type: %s", hashType)
	}
}

// newHashStreamFunc creates a Func from a function that returns a new hash.Hash.
func newHashStreamFunc(newHasher func() hash.Hash) Func {
	return func(r io.Reader) (string, error) {
		h := newHasher()
		if _, err := io.Copy(h, r); err != nil {
			return "", err
		}
		return hex.EncodeToString(h.Sum(nil)), nil
	}
}

// hashXXHashStream creates a new xxhash.Digest and computes the hash.
func hashXXHashStream(r io.Reader) (string, error) {
	h := xxhash.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum64()), nil
}
