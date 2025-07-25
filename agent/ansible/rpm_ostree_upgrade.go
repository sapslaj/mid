// Code generated by ./hack/generate-ansible-types.py DO NOT EDIT
package ansible

import (
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/cast"
)

// Manage an rpm-ostree upgrade transactions.
const RpmOstreeUpgradeName = "rpm_ostree_upgrade"

// Parameters for the `rpm_ostree_upgrade` Ansible module.
type RpmOstreeUpgradeParameters struct {
	// The OSNAME upon which to operate.
	// default: ""
	Os *string `json:"os,omitempty"`

	// Perform the transaction using only pre-cached data, do not download.
	// default: false
	CacheOnly *bool `json:"cache_only,omitempty"`

	// Allow for the upgrade to be a chronologically older tree.
	// default: false
	AllowDowngrade *bool `json:"allow_downgrade,omitempty"`

	// Force peer-to-peer connection instead of using a system message bus.
	// default: false
	Peer *bool `json:"peer,omitempty"`
}

// Wrap the `RpmOstreeUpgradeParameters into an `rpc.RPCCall`.
func (p RpmOstreeUpgradeParameters) ToRPCCall() (rpc.RPCCall[rpc.AnsibleExecuteArgs], error) {
	args, err := cast.AnyToJSONT[map[string]any](p)
	if err != nil {
		return rpc.RPCCall[rpc.AnsibleExecuteArgs]{}, err
	}
	return rpc.RPCCall[rpc.AnsibleExecuteArgs]{
		RPCFunction: rpc.RPCAnsibleExecute,
		Args: rpc.AnsibleExecuteArgs{
			Name: RpmOstreeUpgradeName,
			Args: args,
		},
	}, nil
}

// Return values for the `rpm_ostree_upgrade` Ansible module.
type RpmOstreeUpgradeReturn struct {
	AnsibleCommonReturns

	// The command standard output
	Msg *string `json:"msg,omitempty"`
}

// Unwrap the `rpc.RPCResult` into an `RpmOstreeUpgradeReturn`
func RpmOstreeUpgradeReturnFromRPCResult(r rpc.RPCResult[rpc.AnsibleExecuteResult]) (RpmOstreeUpgradeReturn, error) {
	return cast.AnyToJSONT[RpmOstreeUpgradeReturn](r.Result.Result)
}
