package dirhash

import (
	"io"
	"io/fs"
	"iter"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilehash(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		path   string
		fp     func(*testing.T) io.ReadCloser
		expect string
	}{
		"expected output": {
			path: "testfile",
			fp: func(t *testing.T) io.ReadCloser {
				reader, err := os.Open("./testdata/testfile")
				require.NoError(t, err)
				return reader
			},
			expect: "599a65bde43d8ae16ee0bd67888e6b713a30c8953c19613353a60cf97f1d4c84",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := Filehash(tc.path, tc.fp(t))

			require.NoError(t, err)
			require.Equal(t, tc.expect, got)
		})
	}
}

func TestDirdash(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		tree   func(*testing.T) iter.Seq2[string, io.ReadCloser]
		expect string
	}{
		"expected output": {
			tree: func(t *testing.T) iter.Seq2[string, io.ReadCloser] {
				return func(yield func(string, io.ReadCloser) bool) {
					err := filepath.WalkDir("./testdata/testdir", func(path string, dirent fs.DirEntry, err error) error {
						if err != nil {
							require.NoError(t, err)
							return err
						}
						if dirent.IsDir() {
							return nil
						}
						fp, err := os.Open(path)
						if err != nil {
							require.NoError(t, err)
							return err
						}
						if !yield(path, fp) {
							return fs.SkipAll
						}
						return nil
					})
					require.NoError(t, err)
				}
			},
			expect: "91b870a32b3cb042f80552b2937f8d6019fc8541393f2bcc6be3ba1220837de5",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := Dirhash(tc.tree(t))
			require.NoError(t, err)
			require.Equal(t, tc.expect, got)
		})
	}
}
