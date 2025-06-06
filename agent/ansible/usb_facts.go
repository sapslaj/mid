// Code generated by ./hack/generate-ansible-types.py DO NOT EDIT
package ansible

import (
	"github.com/sapslaj/mid/agent/rpc"
)

// Allows retrieving information about available USB devices through `lsusb`.
const UsbFactsName = "usb_facts"

// Parameters for the `usb_facts` Ansible module.
type UsbFactsParameters struct {
}

// Wrap the `UsbFactsParameters into an `rpc.RPCCall`.
func (p UsbFactsParameters) ToRPCCall() (rpc.RPCCall[rpc.AnsibleExecuteArgs], error) {
	args, err := rpc.AnyToJSONT[map[string]any](p)
	if err != nil {
		return rpc.RPCCall[rpc.AnsibleExecuteArgs]{}, err
	}
	return rpc.RPCCall[rpc.AnsibleExecuteArgs]{
		RPCFunction: rpc.RPCAnsibleExecute,
		Args: rpc.AnsibleExecuteArgs{
			Name: UsbFactsName,
			Args: args,
		},
	}, nil
}

// Return values for the `usb_facts` Ansible module.
type UsbFactsReturn struct {
	AnsibleCommonReturns

	// Dictionary containing details of connected USB devices.
	AnsibleFacts *map[string]any `json:"ansible_facts,omitempty"`
}

// Unwrap the `rpc.RPCResult` into an `UsbFactsReturn`
func UsbFactsReturnFromRPCResult(r rpc.RPCResult[rpc.AnsibleExecuteResult]) (UsbFactsReturn, error) {
	return rpc.AnyToJSONT[UsbFactsReturn](r.Result.Result)
}
