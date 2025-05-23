package types

import "github.com/pulumi/pulumi-go-provider/infer"

type ExecCommand struct {
	Command     []string           `pulumi:"command"`
	Environment *map[string]string `pulumi:"environment,optional"`
	Dir         *string            `pulumi:"dir,optional"`
	Stdin       *string            `pulumi:"stdin,optional"`
}

func (i *ExecCommand) Annotate(a infer.Annotator) {
	a.Describe(
		&i.Command,
		`List of arguments to execute. Under the hood, these are passed to `+
		"`execve`" + `, bypassing any shell`,
	)
	a.Describe(
		&i.Environment,
		`Key-value pairs of environment variables to pass to the process. These are
merged with any system-wide environment variables.`,
	)
	a.Describe(
		&i.Dir,
		`Directory path to chdir to before executing the command. Defaults to the
default working directory for the SSH user and session, usually the user's
home.`,
	)
	a.Describe(
		&i.Stdin,
		`Pass a string to the command's process as standard in.`,
	)
}

type ExecLogging string

const (
	ExecLoggingStdout          ExecLogging = "stdout"
	ExecLoggingStderr          ExecLogging = "stderr"
	ExecLoggingStdoutAndStderr ExecLogging = "stdoutAndStderr"
	ExecLoggingNone            ExecLogging = "none"
)
