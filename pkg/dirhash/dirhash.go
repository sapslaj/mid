package dirhash

import (
	"crypto/sha256"
	"fmt"
	"io"
	"iter"
	"slices"
)

func Filehash(path string, fp io.ReadCloser) (string, error) {
	h := sha256.New()
	h.Write([]byte(path))
	_, err := io.Copy(h, fp)
	fp.Close()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func Dirhash(tree iter.Seq2[string, io.ReadCloser]) (string, error) {
	hashes := []string{}
	for path, fp := range tree {
		hash, err := Filehash(path, fp)
		if err != nil {
			return "", err
		}
		hashes = append(hashes, hash)
	}

	slices.Sort(hashes)
	h := sha256.New()
	for _, hash := range hashes {
		_, err := h.Write([]byte(hash))
		if err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
