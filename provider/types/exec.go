package types

type ExecCommand struct {
	Command     []string           `pulumi:"command"`
	Environment *map[string]string `pulumi:"environment,optional"`
	Dir         *string            `pulumi:"dir,optional"`
	Stdin       *string            `pulumi:"stdin,optional"`
}

type ExecLogging string

const (
	ExecLoggingStdout          ExecLogging = "stdout"
	ExecLoggingStderr          ExecLogging = "stderr"
	ExecLoggingStdoutAndStderr ExecLogging = "stdoutAndStderr"
	ExecLoggingNone            ExecLogging = "none"
)
