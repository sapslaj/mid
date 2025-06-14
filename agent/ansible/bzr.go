// Code generated by ./hack/generate-ansible-types.py DO NOT EDIT
package ansible

import (
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/cast"
)

// Manage `bzr` branches to deploy files or software.
const BzrName = "bzr"

// Parameters for the `bzr` Ansible module.
type BzrParameters struct {
	// SSH or HTTP protocol address of the parent branch.
	Name string `json:"name"`

	// Absolute path of where the branch should be cloned to.
	Dest string `json:"dest"`

	// What version of the branch to clone. This can be the bzr revno or revid.
	// default: "head"
	Version *string `json:"version,omitempty"`

	// If `true`, any modified files in the working tree will be discarded.
	// default: false
	Force *bool `json:"force,omitempty"`

	// Path to bzr executable to use. If not supplied, the normal mechanism for
	// resolving binary paths will be used.
	Executable *string `json:"executable,omitempty"`
}

// Wrap the `BzrParameters into an `rpc.RPCCall`.
func (p BzrParameters) ToRPCCall() (rpc.RPCCall[rpc.AnsibleExecuteArgs], error) {
	args, err := cast.AnyToJSONT[map[string]any](p)
	if err != nil {
		return rpc.RPCCall[rpc.AnsibleExecuteArgs]{}, err
	}
	return rpc.RPCCall[rpc.AnsibleExecuteArgs]{
		RPCFunction: rpc.RPCAnsibleExecute,
		Args: rpc.AnsibleExecuteArgs{
			Name: BzrName,
			Args: args,
		},
	}, nil
}

// Return values for the `bzr` Ansible module.
type BzrReturn struct {
	AnsibleCommonReturns
}

// Unwrap the `rpc.RPCResult` into an `BzrReturn`
func BzrReturnFromRPCResult(r rpc.RPCResult[rpc.AnsibleExecuteResult]) (BzrReturn, error) {
	return cast.AnyToJSONT[BzrReturn](r.Result.Result)
}
