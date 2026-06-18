package pluginregistry

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"sort"
)

// Fingerprint returns a deterministic SHA-256 over a plugin's file bundle. The
// hash is stable regardless of map iteration order: paths are sorted and each
// path and its content length are length-prefixed before hashing so that no two
// distinct bundles can collide by shifting bytes across the path/content
// boundary.
func Fingerprint(files map[string][]byte) string {
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	hasher := sha256.New()
	var lenBuf [8]byte
	writeChunk := func(b []byte) {
		binary.BigEndian.PutUint64(lenBuf[:], uint64(len(b)))
		hasher.Write(lenBuf[:])
		hasher.Write(b)
	}
	for _, path := range paths {
		writeChunk([]byte(path))
		writeChunk(files[path])
	}
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil))
}
