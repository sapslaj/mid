package rpc

import (
	"crypto/rand"
	"encoding/json"
	"os"
	"path"
	"strings"
)

type AnsibleExecuteArgs struct {
	Name               string
	Args               map[string]any
	Environment        map[string]string
	Check              bool
	DebugKeepTempFiles bool
}

type AnsibleExecuteResult struct {
	Stderr       []byte
	Stdout       []byte
	ExitCode     int
	Success      bool
	Result       map[string]any
	DebugTempDir string
}

func AnsibleExecute(args AnsibleExecuteArgs) (AnsibleExecuteResult, error) {
	result := AnsibleExecuteResult{}

	tmpdir, err := os.MkdirTemp(os.TempDir(), "mid-ansible-"+args.Name+"-"+strings.ToLower(rand.Text()))
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

	// TODO: support supplying Python3 location
	execResult, err := Exec(ExecArgs{
		Command: []string{
			"python3",
			"-m",
			"ansible.modules." + args.Name,
			string(dataEncoded),
		},
		Environment: args.Environment,
		Dir:         path.Join(".mid", "ansible"),
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
