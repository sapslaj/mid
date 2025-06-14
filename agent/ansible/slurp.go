// Code generated by ./hack/generate-ansible-types.py DO NOT EDIT
package ansible

import (
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/cast"
)

// This module works like `ansible.builtin.fetch`. It is used for fetching a
// base64- encoded blob containing the data in a remote file.
// This module is also supported for Windows targets.
const SlurpName = "slurp"

// Parameters for the `slurp` Ansible module.
type SlurpParameters struct {
	// The file on the remote system to fetch. This `must` be a file, not a
	// directory.
	Src string `json:"src"`
}

// Wrap the `SlurpParameters into an `rpc.RPCCall`.
func (p SlurpParameters) ToRPCCall() (rpc.RPCCall[rpc.AnsibleExecuteArgs], error) {
	args, err := cast.AnyToJSONT[map[string]any](p)
	if err != nil {
		return rpc.RPCCall[rpc.AnsibleExecuteArgs]{}, err
	}
	return rpc.RPCCall[rpc.AnsibleExecuteArgs]{
		RPCFunction: rpc.RPCAnsibleExecute,
		Args: rpc.AnsibleExecuteArgs{
			Name: SlurpName,
			Args: args,
		},
	}, nil
}

// Return values for the `slurp` Ansible module.
type SlurpReturn struct {
	AnsibleCommonReturns

	// Encoded file content
	Content *string `json:"content,omitempty"`

	// Type of encoding used for file
	Encoding *string `json:"encoding,omitempty"`

	// Actual path of file slurped
	Source *string `json:"source,omitempty"`
}

// Unwrap the `rpc.RPCResult` into an `SlurpReturn`
func SlurpReturnFromRPCResult(r rpc.RPCResult[rpc.AnsibleExecuteResult]) (SlurpReturn, error) {
	return cast.AnyToJSONT[SlurpReturn](r.Result.Result)
}
