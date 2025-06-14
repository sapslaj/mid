// Code generated by ./hack/generate-ansible-types.py DO NOT EDIT
package ansible

import (
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/cast"
)

// Install or uninstall overlay additional packages using `rpm-ostree` command.
const RpmOstreePkgName = "rpm_ostree_pkg"

// State of the overlay package.
// `present` simply ensures that a desired package is installed.
// `absent` removes the specified package.
type RpmOstreePkgState string

const (
	RpmOstreePkgStateAbsent  RpmOstreePkgState = "absent"
	RpmOstreePkgStatePresent RpmOstreePkgState = "present"
)

// Convert a supported type to an optional (pointer) RpmOstreePkgState
func OptionalRpmOstreePkgState[T interface {
	*RpmOstreePkgState | RpmOstreePkgState | *string | string
}](s T) *RpmOstreePkgState {
	switch v := any(s).(type) {
	case *RpmOstreePkgState:
		return v
	case RpmOstreePkgState:
		return &v
	case *string:
		if v == nil {
			return nil
		}
		val := RpmOstreePkgState(*v)
		return &val
	case string:
		val := RpmOstreePkgState(v)
		return &val
	default:
		panic("unsupported type")
	}
}

// Parameters for the `rpm_ostree_pkg` Ansible module.
type RpmOstreePkgParameters struct {
	// Name of overlay package to install or remove.
	Name []string `json:"name"`

	// State of the overlay package.
	// `present` simply ensures that a desired package is installed.
	// `absent` removes the specified package.
	// default: RpmOstreePkgStatePresent
	State *RpmOstreePkgState `json:"state,omitempty"`

	// Adds the options `--apply-live` when `state=present`.
	// Option is ignored when `state=absent`.
	// For more information, please see `https://coreos.github.io/rpm-ostree/apply-
	// live/`.
	// default: false
	ApplyLive *bool `json:"apply_live,omitempty"`
}

// Wrap the `RpmOstreePkgParameters into an `rpc.RPCCall`.
func (p RpmOstreePkgParameters) ToRPCCall() (rpc.RPCCall[rpc.AnsibleExecuteArgs], error) {
	args, err := cast.AnyToJSONT[map[string]any](p)
	if err != nil {
		return rpc.RPCCall[rpc.AnsibleExecuteArgs]{}, err
	}
	return rpc.RPCCall[rpc.AnsibleExecuteArgs]{
		RPCFunction: rpc.RPCAnsibleExecute,
		Args: rpc.AnsibleExecuteArgs{
			Name: RpmOstreePkgName,
			Args: args,
		},
	}, nil
}

// Return values for the `rpm_ostree_pkg` Ansible module.
type RpmOstreePkgReturn struct {
	AnsibleCommonReturns

	// Return code of rpm-ostree command.
	Rc *int `json:"rc,omitempty"`

	// State changes.
	Changed *bool `json:"changed,omitempty"`

	// Action performed.
	Action *string `json:"action,omitempty"`

	// A list of packages specified.
	Packages *[]any `json:"packages,omitempty"`

	// Stdout of rpm-ostree command.
	Stdout *string `json:"stdout,omitempty"`

	// Stderr of rpm-ostree command.
	Stderr *string `json:"stderr,omitempty"`

	// Full command used for performed action.
	Cmd *string `json:"cmd,omitempty"`

	// Determine if machine needs a reboot to apply current changes.
	NeedsReboot *bool `json:"needs_reboot,omitempty"`
}

// Unwrap the `rpc.RPCResult` into an `RpmOstreePkgReturn`
func RpmOstreePkgReturnFromRPCResult(r rpc.RPCResult[rpc.AnsibleExecuteResult]) (RpmOstreePkgReturn, error) {
	return cast.AnyToJSONT[RpmOstreePkgReturn](r.Result.Result)
}
