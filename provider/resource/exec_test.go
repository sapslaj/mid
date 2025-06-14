package resource

import (
	"testing"

	"github.com/sapslaj/mid/agent/ansible"
	"github.com/sapslaj/mid/pkg/ptr"
	"github.com/sapslaj/mid/provider/types"
	"github.com/stretchr/testify/assert"
)

func TestExec_argsToTaskParameters(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input               ExecArgs
		lifecycle           string
		expectedParameters  ansible.CommandParameters
		expectedEnvironment map[string]string
	}{
		"simple create": {
			input: ExecArgs{
				Create: types.ExecCommand{
					Command: []string{"touch", "/grass"},
				},
			},
			lifecycle: "create",
			expectedParameters: ansible.CommandParameters{
				Argv:               ptr.Of([]string{"touch", "/grass"}),
				ExpandArgumentVars: ptr.Of(false),
				StripEmptyEnds:     ptr.Of(false),
			},
			expectedEnvironment: map[string]string{},
		},

		"create and update are same": {
			input: ExecArgs{
				Create: types.ExecCommand{
					Command: []string{"touch", "/grass"},
				},
			},
			lifecycle: "update",
			expectedParameters: ansible.CommandParameters{
				Argv:               ptr.Of([]string{"touch", "/grass"}),
				ExpandArgumentVars: ptr.Of(false),
				StripEmptyEnds:     ptr.Of(false),
			},
			expectedEnvironment: map[string]string{},
		},

		"create and update are different": {
			input: ExecArgs{
				Create: types.ExecCommand{
					Command: []string{"touch", "/grass"},
				},
				Update: &types.ExecCommand{
					Command: []string{"touch", "/cat"},
				},
			},
			lifecycle: "update",
			expectedParameters: ansible.CommandParameters{
				Argv:               ptr.Of([]string{"touch", "/cat"}),
				ExpandArgumentVars: ptr.Of(false),
				StripEmptyEnds:     ptr.Of(false),
			},
			expectedEnvironment: map[string]string{},
		},

		"create and delete": {
			input: ExecArgs{
				Create: types.ExecCommand{
					Command: []string{"touch", "/grass"},
				},
				Delete: &types.ExecCommand{
					Command: []string{"rm", "-f", "/grass"},
				},
			},
			lifecycle: "delete",
			expectedParameters: ansible.CommandParameters{
				Argv:               ptr.Of([]string{"rm", "-f", "/grass"}),
				ExpandArgumentVars: ptr.Of(false),
				StripEmptyEnds:     ptr.Of(false),
			},
			expectedEnvironment: map[string]string{},
		},

		"shared environment": {
			input: ExecArgs{
				Create: types.ExecCommand{
					Command: []string{"touch", "/grass"},
				},
				Environment: ptr.Of(map[string]string{
					"FOO": "BAR",
				}),
			},
			lifecycle: "create",
			expectedParameters: ansible.CommandParameters{
				Argv:               ptr.Of([]string{"touch", "/grass"}),
				ExpandArgumentVars: ptr.Of(false),
				StripEmptyEnds:     ptr.Of(false),
			},
			expectedEnvironment: map[string]string{
				"FOO": "BAR",
			},
		},

		"lifecycle specific environment": {
			input: ExecArgs{
				Create: types.ExecCommand{
					Command: []string{"touch", "/grass"},
					Environment: ptr.Of(map[string]string{
						"FOO": "BAR",
					}),
				},
			},
			lifecycle: "create",
			expectedParameters: ansible.CommandParameters{
				Argv:               ptr.Of([]string{"touch", "/grass"}),
				ExpandArgumentVars: ptr.Of(false),
				StripEmptyEnds:     ptr.Of(false),
			},
			expectedEnvironment: map[string]string{
				"FOO": "BAR",
			},
		},

		"mixed environment": {
			input: ExecArgs{
				Create: types.ExecCommand{
					Command: []string{"touch", "/grass"},
					Environment: ptr.Of(map[string]string{
						"B": "2",
						"C": "3",
					}),
				},
				Environment: ptr.Of(map[string]string{
					"A": "1",
					"B": "1",
				}),
			},
			lifecycle: "create",
			expectedParameters: ansible.CommandParameters{
				Argv:               ptr.Of([]string{"touch", "/grass"}),
				ExpandArgumentVars: ptr.Of(false),
				StripEmptyEnds:     ptr.Of(false),
			},
			expectedEnvironment: map[string]string{
				"A": "1",
				"B": "2",
				"C": "3",
			},
		},

		"shared dir": {
			input: ExecArgs{
				Create: types.ExecCommand{
					Command: []string{"touch", "grass"},
				},
				Dir: ptr.Of("/tmp"),
			},
			lifecycle: "create",
			expectedParameters: ansible.CommandParameters{
				Argv:               ptr.Of([]string{"touch", "grass"}),
				Chdir:              ptr.Of("/tmp"),
				ExpandArgumentVars: ptr.Of(false),
				StripEmptyEnds:     ptr.Of(false),
			},
			expectedEnvironment: map[string]string{},
		},

		"lifecycle specific dir": {
			input: ExecArgs{
				Create: types.ExecCommand{
					Command: []string{"touch", "grass"},
					Dir:     ptr.Of("/tmp/create"),
				},
			},
			lifecycle: "create",
			expectedParameters: ansible.CommandParameters{
				Argv:               ptr.Of([]string{"touch", "grass"}),
				Chdir:              ptr.Of("/tmp/create"),
				ExpandArgumentVars: ptr.Of(false),
				StripEmptyEnds:     ptr.Of(false),
			},
			expectedEnvironment: map[string]string{},
		},

		"mixed dir": {
			input: ExecArgs{
				Create: types.ExecCommand{
					Command: []string{"touch", "grass"},
					Dir:     ptr.Of("/tmp/create"),
				},
				Dir: ptr.Of("/tmp"),
			},
			lifecycle: "create",
			expectedParameters: ansible.CommandParameters{
				Argv:               ptr.Of([]string{"touch", "grass"}),
				Chdir:              ptr.Of("/tmp/create"),
				ExpandArgumentVars: ptr.Of(false),
				StripEmptyEnds:     ptr.Of(false),
			},
			expectedEnvironment: map[string]string{},
		},

		"expandArgumentVars": {
			input: ExecArgs{
				Create: types.ExecCommand{
					Command: []string{"mkdir", "-p", "$HOME"},
				},
				ExpandArgumentVars: ptr.Of(true),
			},
			lifecycle: "create",
			expectedParameters: ansible.CommandParameters{
				Argv:               ptr.Of([]string{"mkdir", "-p", "$HOME"}),
				ExpandArgumentVars: ptr.Of(true),
				StripEmptyEnds:     ptr.Of(false),
			},
			expectedEnvironment: map[string]string{},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r := Exec{}

			gotParameters, gotEnvironment, err := r.argsToTaskParameters(tc.input, tc.lifecycle)

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedParameters, gotParameters)
			assert.Equal(t, tc.expectedEnvironment, gotEnvironment)
		})
	}
}

func TestExec_updateStateFromOutput(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		output  ansible.CommandReturn
		logging types.ExecLogging
		stderr  string
		stdout  string
	}{
		"none": {
			output: ansible.CommandReturn{
				Stdout: ptr.Of("this is stdout"),
				Stderr: ptr.Of("this is stderr"),
			},
			logging: types.ExecLoggingNone,
			stdout:  "",
			stderr:  "",
		},
		"stderr": {
			output: ansible.CommandReturn{
				Stdout: ptr.Of("this is stdout"),
				Stderr: ptr.Of("this is stderr"),
			},
			logging: types.ExecLoggingStderr,
			stdout:  "",
			stderr:  "this is stderr",
		},
		"stdout": {
			output: ansible.CommandReturn{
				Stdout: ptr.Of("this is stdout"),
				Stderr: ptr.Of("this is stderr"),
			},
			logging: types.ExecLoggingStdout,
			stdout:  "this is stdout",
			stderr:  "",
		},
		"stdoutAndStderr": {
			output: ansible.CommandReturn{
				Stdout: ptr.Of("this is stdout"),
				Stderr: ptr.Of("this is stderr"),
			},
			logging: types.ExecLoggingStdoutAndStderr,
			stdout:  "this is stdout",
			stderr:  "this is stderr",
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r := Exec{}

			state := r.updateStateFromOutput(ExecArgs{Logging: &tc.logging}, ExecState{}, tc.output)

			assert.Equal(t, tc.stderr, state.Stderr)
			assert.Equal(t, tc.stdout, state.Stdout)
		})
	}
}
