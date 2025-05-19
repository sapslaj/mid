package rpc

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type AnsiballZExecuteArgs struct {
	Zip                []byte
	Name               string
	Args               map[string]any
	DebugKeepTempFiles bool
}

type AnsiballZExecuteResult struct {
	Stderr       []byte
	Stdout       []byte
	ExitCode     int
	Result       map[string]any
	DebugTempDir string
}

func AnsiballZExecute(args AnsiballZExecuteArgs) (AnsiballZExecuteResult, error) {
	result := AnsiballZExecuteResult{}

	tmpdir, err := os.MkdirTemp(os.TempDir(), strings.ToLower(rand.Text()))
	result.DebugTempDir = tmpdir
	if err != nil {
		return result, err
	}

	if !args.DebugKeepTempFiles {
		defer os.RemoveAll(tmpdir)
	}

	moduleArgs, err := json.Marshal(args.Args)
	if err != nil {
		return result, err
	}

	reader, err := zip.NewReader(bytes.NewReader(args.Zip), int64(len(args.Zip)))
	if err != nil {
		return result, err
	}

	for _, f := range reader.File {
		err := func() error {
			filePath := filepath.Join(tmpdir, f.Name)
			if !strings.HasPrefix(filePath, filepath.Clean(tmpdir)+string(os.PathSeparator)) {
				return fmt.Errorf("invalid file path: %s", filePath)
			}

			if f.FileInfo().IsDir() {
				err := os.MkdirAll(filePath, os.ModePerm)
				if err != nil {
					return err
				}
				return nil
			}

			err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
			if err != nil {
				return err
			}

			destinationFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer destinationFile.Close()

			zippedFile, err := f.Open()
			if err != nil {
				return err
			}
			defer zippedFile.Close()

			_, err = io.Copy(destinationFile, zippedFile)
			if err != nil {
				return err
			}

			return nil
		}()
		if err != nil {
			return result, err
		}
	}

	// TODO: support supplying Python3 location
	execResult, err := Exec(ExecArgs{
		Command: []string{
			"python3",
			"-m",
			"ansible.modules." + args.Name,
			string(moduleArgs),
		},
		Dir: tmpdir,
	})
	result.Stderr = execResult.Stderr
	result.Stdout = execResult.Stdout
	result.ExitCode = execResult.ExitCode
	if err != nil {
		return result, err
	}

	err = json.Unmarshal(result.Stdout, &result.Result)
	if err != nil {
		return result, err
	}

	return result, nil
}
