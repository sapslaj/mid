// Code generated by ./hack/generate-ansible-types.py DO NOT EDIT
package ansible

import (
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/cast"
)

// IPS packages are the native packages in Solaris 11 and higher.
const Pkg5Name = "pkg5"

// Whether to install (`present`, `latest`), or remove (`absent`) a package.
type Pkg5State string

const (
	Pkg5StateAbsent      Pkg5State = "absent"
	Pkg5StateLatest      Pkg5State = "latest"
	Pkg5StatePresent     Pkg5State = "present"
	Pkg5StateInstalled   Pkg5State = "installed"
	Pkg5StateRemoved     Pkg5State = "removed"
	Pkg5StateUninstalled Pkg5State = "uninstalled"
)

// Convert a supported type to an optional (pointer) Pkg5State
func OptionalPkg5State[T interface {
	*Pkg5State | Pkg5State | *string | string
}](s T) *Pkg5State {
	switch v := any(s).(type) {
	case *Pkg5State:
		return v
	case Pkg5State:
		return &v
	case *string:
		if v == nil {
			return nil
		}
		val := Pkg5State(*v)
		return &val
	case string:
		val := Pkg5State(v)
		return &val
	default:
		panic("unsupported type")
	}
}

// Parameters for the `pkg5` Ansible module.
type Pkg5Parameters struct {
	// An FRMI of the package(s) to be installed/removed/updated.
	// Multiple packages may be specified, separated by `,`.
	Name []string `json:"name"`

	// Whether to install (`present`, `latest`), or remove (`absent`) a package.
	// default: Pkg5StatePresent
	State *Pkg5State `json:"state,omitempty"`

	// Accept any licences.
	// default: false
	AcceptLicenses *bool `json:"accept_licenses,omitempty"`

	// Creates a new boot environment with the given name.
	BeName *string `json:"be_name,omitempty"`

	// Refresh publishers before execution.
	// default: true
	Refresh *bool `json:"refresh,omitempty"`

	// Set to `true` to disable quiet execution.
	// default: false
	Verbose *bool `json:"verbose,omitempty"`
}

// Wrap the `Pkg5Parameters into an `rpc.RPCCall`.
func (p Pkg5Parameters) ToRPCCall() (rpc.RPCCall[rpc.AnsibleExecuteArgs], error) {
	args, err := cast.AnyToJSONT[map[string]any](p)
	if err != nil {
		return rpc.RPCCall[rpc.AnsibleExecuteArgs]{}, err
	}
	return rpc.RPCCall[rpc.AnsibleExecuteArgs]{
		RPCFunction: rpc.RPCAnsibleExecute,
		Args: rpc.AnsibleExecuteArgs{
			Name: Pkg5Name,
			Args: args,
		},
	}, nil
}

// Return values for the `pkg5` Ansible module.
type Pkg5Return struct {
	AnsibleCommonReturns
}

// Unwrap the `rpc.RPCResult` into an `Pkg5Return`
func Pkg5ReturnFromRPCResult(r rpc.RPCResult[rpc.AnsibleExecuteResult]) (Pkg5Return, error) {
	return cast.AnyToJSONT[Pkg5Return](r.Result.Result)
}
