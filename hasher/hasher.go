// Package hasher provides a unified interface for various hashing algorithms.
// It supports standard cryptographic hashes (MD5, SHA1, SHA256) and
// high-performance non-cryptographic hashes (XXH3, HighwayHash, Wyhash, Blake3)
// specifically optimized for file integrity verification.
package hasher

import (
	"crypto/md5"  // #nosec G501 -- MD5 is supported as a legacy hash option for file integrity verification, not for security-critical contexts.
	"crypto/sha1" // #nosec G505 -- SHA1 is supported as a legacy hash option for file integrity verification, not for security-critical contexts.
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"

	"github.com/minio/highwayhash"
	"github.com/orisano/wyhash"
	"github.com/zeebo/blake3"
	"github.com/zeebo/xxh3"
)

// Hash types constants define the supported hashing algorithms.
const (
	// HashMD5 is the legacy MD5 algorithm (128-bit).
	HashMD5 = "MD5"
	// HashSHA1 is the legacy SHA1 algorithm (160-bit).
	HashSHA1 = "SHA1"
	// HashSHA256 is the standard SHA-256 algorithm (256-bit).
	HashSHA256 = "SHA256"
	// HashXXH3 is the 128-bit version of XXH3, optimized for high performance.
	HashXXH3 = "XXH3-128"
	// HashHighway is HighwayHash-128, a robust and fast PRF.
	HashHighway = "HIGHWAYHASH"
	// HashWyhash is the 64-bit Wyhash algorithm, known for its extreme speed.
	HashWyhash = "WYHASH"
	// HashBlake3 is the Blake3 cryptographic hash, designed for extreme speed and security.
	HashBlake3 = "BLAKE3"
)

// Func is a function type that takes a reader and returns a hash string or an error.
type Func func(io.Reader) (string, error)

// GetHasher returns the appropriate hash function based on the requested hash type.
// It returns a Func that can process an io.Reader and an error if the type is unsupported.
func GetHasher(hashType string) (Func, error) {
	switch hashType {
	case HashMD5:
		// #nosec G401 -- MD5 is supported as a legacy hash option for file integrity verification, not for security-critical contexts.
		return newHashStreamFunc(md5.New), nil
	case HashSHA1:
		// #nosec G401 -- SHA1 is supported as a legacy hash option for file integrity verification, not for security-critical contexts.
		return newHashStreamFunc(sha1.New), nil
	case HashSHA256:
		return newHashStreamFunc(sha256.New), nil
	case HashXXH3:
		// Uses zeebo/xxh3 implementation for 128-bit hashes.
		return newHashStreamFunc(func() hash.Hash { return xxh3.New128() }), nil
	case HashHighway:
		// Uses minio/highwayhash with a standardized fixed key.
		return hashHighwayStream, nil
	case HashWyhash:
		// Uses orisano/wyhash with a standardized fixed seed.
		return hashWyhashStream, nil
	case HashBlake3:
		// Uses zeebo/blake3 for high-performance cryptographic hashing.
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

// hashHighwayStream computes HighwayHash using a fixed all-zeros 32-byte key.
func hashHighwayStream(r io.Reader) (string, error) {
	key := make([]byte, 32) // Fixed all-zeros key
	h, err := highwayhash.New128(key)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// hashWyhashStream computes Wyhash using a fixed seed of 0.
func hashWyhashStream(r io.Reader) (string, error) {
	h := wyhash.New(0)
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
