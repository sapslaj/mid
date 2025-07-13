package rpc

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
)

type ExecArgs struct {
	Command            []string
	Dir                string
	Environment        map[string]string
	Stdin              []byte
	ExpandArgumentVars bool
}

type ExecResult struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
	Pid      int
}

func Exec(args ExecArgs) (ExecResult, error) {
	if len(args.Command) == 0 {
		return ExecResult{}, errors.New("no command specified")
	}

	if args.ExpandArgumentVars {
		mapping := func(key string) string {
			value, ok := args.Environment[key]
			if ok {
				return value
			}
			return os.Getenv(key)
		}

		for i := range args.Command {
			args.Command[i] = os.Expand(args.Command[i], mapping)
		}
	}

	stdin := bytes.NewReader(args.Stdin)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command(args.Command[0], args.Command[1:]...)
	cmd.Dir = args.Dir
	cmd.Stdin = stdin
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	cmd.Env = os.Environ()
	for key, value := range args.Environment {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	err := cmd.Run()
	if err != nil {
		_, isExitError := err.(*exec.ExitError)
		if !isExitError {
			return ExecResult{
				Stdout:   stdout.Bytes(),
				Stderr:   stderr.Bytes(),
				ExitCode: -1,
			}, err
		}
	}
	return ExecResult{
		Stdout:   stdout.Bytes(),
		Stderr:   stderr.Bytes(),
		ExitCode: cmd.ProcessState.ExitCode(),
		Pid:      cmd.ProcessState.Pid(),
	}, nil
}
