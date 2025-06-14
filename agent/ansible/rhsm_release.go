// Code generated by ./hack/generate-ansible-types.py DO NOT EDIT
package ansible

import (
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/cast"
)

// Sets or unsets the release version used by RHSM repositories.
const RhsmReleaseName = "rhsm_release"

// Parameters for the `rhsm_release` Ansible module.
type RhsmReleaseParameters struct {
	// RHSM release version to use.
	// To unset either pass `null` for this option, or omit this option.
	Release *string `json:"release,omitempty"`
}

// Wrap the `RhsmReleaseParameters into an `rpc.RPCCall`.
func (p RhsmReleaseParameters) ToRPCCall() (rpc.RPCCall[rpc.AnsibleExecuteArgs], error) {
	args, err := cast.AnyToJSONT[map[string]any](p)
	if err != nil {
		return rpc.RPCCall[rpc.AnsibleExecuteArgs]{}, err
	}
	return rpc.RPCCall[rpc.AnsibleExecuteArgs]{
		RPCFunction: rpc.RPCAnsibleExecute,
		Args: rpc.AnsibleExecuteArgs{
			Name: RhsmReleaseName,
			Args: args,
		},
	}, nil
}

// Return values for the `rhsm_release` Ansible module.
type RhsmReleaseReturn struct {
	AnsibleCommonReturns

	// The current RHSM release version value.
	CurrentRelease *string `json:"current_release,omitempty"`
}

// Unwrap the `rpc.RPCResult` into an `RhsmReleaseReturn`
func RhsmReleaseReturnFromRPCResult(r rpc.RPCResult[rpc.AnsibleExecuteResult]) (RhsmReleaseReturn, error) {
	return cast.AnyToJSONT[RhsmReleaseReturn](r.Result.Result)
}
