// Code generated by ./hack/generate-ansible-types.py DO NOT EDIT
package ansible

import (
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/cast"
)

// Control Linux encrypted block devices that are set up during system boot in
// `/etc/crypttab`.
const CrypttabName = "crypttab"

// Use `present` to add a line to `/etc/crypttab` or update its definition if
// already present.
// Use `absent` to remove a line with matching `name`.
// Use `opts_present` to add options to those already present; options with
// different values will be updated.
// Use `opts_absent` to remove options from the existing set.
type CrypttabState string

const (
	CrypttabStateAbsent      CrypttabState = "absent"
	CrypttabStateOptsAbsent  CrypttabState = "opts_absent"
	CrypttabStateOptsPresent CrypttabState = "opts_present"
	CrypttabStatePresent     CrypttabState = "present"
)

// Parameters for the `crypttab` Ansible module.
type CrypttabParameters struct {
	// Name of the encrypted block device as it appears in the `/etc/crypttab`
	// file, or optionally prefixed with `/dev/mapper/`, as it appears in the
	// filesystem. `/dev/mapper/` will be stripped from `name`.
	Name string `json:"name"`

	// Use `present` to add a line to `/etc/crypttab` or update its definition if
	// already present.
	// Use `absent` to remove a line with matching `name`.
	// Use `opts_present` to add options to those already present; options with
	// different values will be updated.
	// Use `opts_absent` to remove options from the existing set.
	State CrypttabState `json:"state"`

	// Path to the underlying block device or file, or the UUID of a block-device
	// prefixed with `UUID=`.
	BackingDevice *string `json:"backing_device,omitempty"`

	// Encryption password, the path to a file containing the password, or `-` or
	// unset if the password should be entered at boot.
	Password *string `json:"password,omitempty"`

	// A comma-delimited list of options. See `crypttab(5\`) for details.
	Opts *string `json:"opts,omitempty"`

	// Path to file to use instead of `/etc/crypttab`.
	// This might be useful in a chroot environment.
	// default: "/etc/crypttab"
	Path *string `json:"path,omitempty"`
}

// Wrap the `CrypttabParameters into an `rpc.RPCCall`.
func (p CrypttabParameters) ToRPCCall() (rpc.RPCCall[rpc.AnsibleExecuteArgs], error) {
	args, err := cast.AnyToJSONT[map[string]any](p)
	if err != nil {
		return rpc.RPCCall[rpc.AnsibleExecuteArgs]{}, err
	}
	return rpc.RPCCall[rpc.AnsibleExecuteArgs]{
		RPCFunction: rpc.RPCAnsibleExecute,
		Args: rpc.AnsibleExecuteArgs{
			Name: CrypttabName,
			Args: args,
		},
	}, nil
}

// Return values for the `crypttab` Ansible module.
type CrypttabReturn struct {
	AnsibleCommonReturns
}

// Unwrap the `rpc.RPCResult` into an `CrypttabReturn`
func CrypttabReturnFromRPCResult(r rpc.RPCResult[rpc.AnsibleExecuteResult]) (CrypttabReturn, error) {
	return cast.AnyToJSONT[CrypttabReturn](r.Result.Result)
}
