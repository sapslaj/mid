// Code generated by ./hack/generate-ansible-types.py DO NOT EDIT
package ansible

import (
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/cast"
)

// Retrieve details about Python applications installed in isolated virtualenvs
// using pipx.
const PipxInfoName = "pipx_info"

// Parameters for the `pipx_info` Ansible module.
type PipxInfoParameters struct {
	// Name of an application installed with `pipx`.
	Name *string `json:"name,omitempty"`

	// Include dependent packages in the output.
	// default: false
	IncludeDeps *bool `json:"include_deps,omitempty"`

	// Include injected packages in the output.
	// default: false
	IncludeInjected *bool `json:"include_injected,omitempty"`

	// Returns the raw output of `pipx list --json`.
	// The raw output is not affected by `include_deps` or `include_injected`.
	// default: false
	IncludeRaw *bool `json:"include_raw,omitempty"`

	// The module will pass the `--global` argument to `pipx`, to execute actions
	// in global scope.
	// The `--global` is only available in `pipx>=1.6.0`, so make sure to have a
	// compatible version when using this option. Moreover, a nasty bug with
	// `--global` was fixed in `pipx==1.7.0`, so it is strongly recommended you
	// used that version or newer.
	// default: false
	Global *bool `json:"global,omitempty"`

	// Path to the `pipx` installed in the system.
	// If not specified, the module will use `python -m pipx` to run the tool,
	// using the same Python interpreter as ansible itself.
	Executable *string `json:"executable,omitempty"`
}

// Wrap the `PipxInfoParameters into an `rpc.RPCCall`.
func (p PipxInfoParameters) ToRPCCall() (rpc.RPCCall[rpc.AnsibleExecuteArgs], error) {
	args, err := cast.AnyToJSONT[map[string]any](p)
	if err != nil {
		return rpc.RPCCall[rpc.AnsibleExecuteArgs]{}, err
	}
	return rpc.RPCCall[rpc.AnsibleExecuteArgs]{
		RPCFunction: rpc.RPCAnsibleExecute,
		Args: rpc.AnsibleExecuteArgs{
			Name: PipxInfoName,
			Args: args,
		},
	}, nil
}

// Return values for the `pipx_info` Ansible module.
type PipxInfoReturn struct {
	AnsibleCommonReturns

	// The list of installed applications.
	Application *map[string]any `json:"application,omitempty"`

	// The raw output of the `pipx list` command, when `include_raw=true`. Used for
	// debugging.
	RawOutput *map[string]any `json:"raw_output,omitempty"`

	// Command executed to obtain the list of installed applications.
	Cmd *[]string `json:"cmd,omitempty"`

	// Version of pipx.
	Version *string `json:"version,omitempty"`
}

// Unwrap the `rpc.RPCResult` into an `PipxInfoReturn`
func PipxInfoReturnFromRPCResult(r rpc.RPCResult[rpc.AnsibleExecuteResult]) (PipxInfoReturn, error) {
	return cast.AnyToJSONT[PipxInfoReturn](r.Result.Result)
}
