// Code generated by ./hack/generate-ansible-types.py DO NOT EDIT
package ansible

import (
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/cast"
)

// Gather facts from ZFS pool properties.
const ZpoolFactsName = "zpool_facts"

// Parameters for the `zpool_facts` Ansible module.
type ZpoolFactsParameters struct {
	// ZFS pool name.
	Name *string `json:"name,omitempty"`

	// Specifies if property values should be displayed in machine friendly format.
	// default: false
	Parsable *bool `json:"parsable,omitempty"`

	// Specifies which dataset properties should be queried in comma-separated
	// format. For more information about dataset properties, check zpool(1M) man
	// page.
	// default: "all"
	Properties *string `json:"properties,omitempty"`
}

// Wrap the `ZpoolFactsParameters into an `rpc.RPCCall`.
func (p ZpoolFactsParameters) ToRPCCall() (rpc.RPCCall[rpc.AnsibleExecuteArgs], error) {
	args, err := cast.AnyToJSONT[map[string]any](p)
	if err != nil {
		return rpc.RPCCall[rpc.AnsibleExecuteArgs]{}, err
	}
	return rpc.RPCCall[rpc.AnsibleExecuteArgs]{
		RPCFunction: rpc.RPCAnsibleExecute,
		Args: rpc.AnsibleExecuteArgs{
			Name: ZpoolFactsName,
			Args: args,
		},
	}, nil
}

// Return values for the `zpool_facts` Ansible module.
type ZpoolFactsReturn struct {
	AnsibleCommonReturns

	// Dictionary containing all the detailed information about the ZFS pool facts.
	AnsibleFacts *any `json:"ansible_facts,omitempty"`

	// ZFS pool name.
	Name *string `json:"name,omitempty"`

	// If parsable output should be provided in machine friendly format.
	Parsable *bool `json:"parsable,omitempty"`
}

// Unwrap the `rpc.RPCResult` into an `ZpoolFactsReturn`
func ZpoolFactsReturnFromRPCResult(r rpc.RPCResult[rpc.AnsibleExecuteResult]) (ZpoolFactsReturn, error) {
	return cast.AnyToJSONT[ZpoolFactsReturn](r.Result.Result)
}
