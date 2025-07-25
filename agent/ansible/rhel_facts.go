// Code generated by ./hack/generate-ansible-types.py DO NOT EDIT
package ansible

import (
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/cast"
)

// Compatibility layer for using the `ansible.builtin.package` module for rpm-
// ostree based systems via setting the `pkg_mgr` fact correctly.
const RhelFactsName = "rhel_facts"

// Parameters for the `rhel_facts` Ansible module.
type RhelFactsParameters struct {
}

// Wrap the `RhelFactsParameters into an `rpc.RPCCall`.
func (p RhelFactsParameters) ToRPCCall() (rpc.RPCCall[rpc.AnsibleExecuteArgs], error) {
	args, err := cast.AnyToJSONT[map[string]any](p)
	if err != nil {
		return rpc.RPCCall[rpc.AnsibleExecuteArgs]{}, err
	}
	return rpc.RPCCall[rpc.AnsibleExecuteArgs]{
		RPCFunction: rpc.RPCAnsibleExecute,
		Args: rpc.AnsibleExecuteArgs{
			Name: RhelFactsName,
			Args: args,
		},
	}, nil
}

// Return values for the `rhel_facts` Ansible module.
type RhelFactsReturn struct {
	AnsibleCommonReturns

	// Relevant Ansible Facts
	AnsibleFacts *any `json:"ansible_facts,omitempty"`
}

// Unwrap the `rpc.RPCResult` into an `RhelFactsReturn`
func RhelFactsReturnFromRPCResult(r rpc.RPCResult[rpc.AnsibleExecuteResult]) (RhelFactsReturn, error) {
	return cast.AnyToJSONT[RhelFactsReturn](r.Result.Result)
}
