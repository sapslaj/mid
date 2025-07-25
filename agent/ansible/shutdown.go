// Code generated by ./hack/generate-ansible-types.py DO NOT EDIT
package ansible

import (
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/cast"
)

// Shut downs a machine.
const ShutdownName = "shutdown"

// Parameters for the `shutdown` Ansible module.
type ShutdownParameters struct {
	// Seconds to wait before shutdown. Passed as a parameter to the shutdown
	// command.
	// On Linux, macOS and OpenBSD, this is converted to minutes and rounded down.
	// If less than 60, it is set to 0.
	// On Solaris and FreeBSD, this represents seconds.
	// default: 0
	Delay *int `json:"delay,omitempty"`

	// Message to display to users before shutdown.
	// default: "Shut down initiated by Ansible"
	Msg *string `json:"msg,omitempty"`

	// Paths to search on the remote machine for the `shutdown` command.
	// `Only` these paths are searched for the `shutdown` command. `PATH` is
	// ignored in the remote node when searching for the `shutdown` command.
	// default: ["/sbin", "/usr/sbin", "/usr/local/sbin"]
	SearchPaths *[]string `json:"search_paths,omitempty"`
}

// Wrap the `ShutdownParameters into an `rpc.RPCCall`.
func (p ShutdownParameters) ToRPCCall() (rpc.RPCCall[rpc.AnsibleExecuteArgs], error) {
	args, err := cast.AnyToJSONT[map[string]any](p)
	if err != nil {
		return rpc.RPCCall[rpc.AnsibleExecuteArgs]{}, err
	}
	return rpc.RPCCall[rpc.AnsibleExecuteArgs]{
		RPCFunction: rpc.RPCAnsibleExecute,
		Args: rpc.AnsibleExecuteArgs{
			Name: ShutdownName,
			Args: args,
		},
	}, nil
}

// Return values for the `shutdown` Ansible module.
type ShutdownReturn struct {
	AnsibleCommonReturns

	// `true` if the machine has been shut down.
	Shutdown *bool `json:"shutdown,omitempty"`
}

// Unwrap the `rpc.RPCResult` into an `ShutdownReturn`
func ShutdownReturnFromRPCResult(r rpc.RPCResult[rpc.AnsibleExecuteResult]) (ShutdownReturn, error) {
	return cast.AnyToJSONT[ShutdownReturn](r.Result.Result)
}
