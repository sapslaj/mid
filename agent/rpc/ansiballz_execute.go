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
	Check              bool
	DebugKeepTempFiles bool
}

type AnsiballZExecuteResult struct {
	Stderr       []byte
	Stdout       []byte
	ExitCode     int
	Success      bool
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

	args.Args["_ansible_check_mode"] = args.Check
	args.Args["_ansible_no_log"] = false
	args.Args["_ansible_debug"] = false
	args.Args["_ansible_diff"] = true
	args.Args["_ansible_verbosity"] = 0
	args.Args["_ansible_version"] = "2.18.5"
	args.Args["_ansible_module_name"] = args.Name
	args.Args["_ansible_syslog_facility"] = "LOG_USER"
	args.Args["_ansible_selinux_special_fs"] = []string{"fuse", "nfs", "vboxsf", "ramfs", "9p", "vfat"}
	args.Args["_ansible_string_conversion_action"] = "warn"
	args.Args["_ansible_socket"] = nil
	args.Args["_ansible_shell_executable"] = "/bin/sh"
	args.Args["_ansible_keep_remote_files"] = args.DebugKeepTempFiles
	args.Args["_ansible_tmpdir"] = tmpdir
	args.Args["_ansible_remote_tmp"] = tmpdir
	args.Args["_ansible_ignore_unknown_opts"] = false
	args.Args["_ansible_target_log_info"] = nil

	data := map[string]any{
		"ANSIBLE_MODULE_ARGS": args.Args,
	}

	dataEncoded, err := json.Marshal(data)
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
			string(dataEncoded),
		},
		Dir: tmpdir,
	})
	result.Stderr = execResult.Stderr
	result.Stdout = execResult.Stdout
	result.ExitCode = execResult.ExitCode
	if err != nil {
		return result, err
	}

	if result.ExitCode == 0 {
		result.Success = true
	}

	err = json.Unmarshal(result.Stdout, &result.Result)
	if err != nil {
		return result, err
	}

	return result, nil
}
