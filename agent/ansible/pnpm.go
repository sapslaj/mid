// Code generated by ./hack/generate-ansible-types.py DO NOT EDIT
package ansible

import (
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/cast"
)

// Manage Node.js packages with the `pnpm package manager, https://pnpm.io/`.
const PnpmName = "pnpm"

// Installation state of the named Node.js library.
// If `absent` is selected, a name option must be provided.
type PnpmState string

const (
	PnpmStatePresent PnpmState = "present"
	PnpmStateAbsent  PnpmState = "absent"
	PnpmStateLatest  PnpmState = "latest"
)

// Convert a supported type to an optional (pointer) PnpmState
func OptionalPnpmState[T interface {
	*PnpmState | PnpmState | *string | string
}](s T) *PnpmState {
	switch v := any(s).(type) {
	case *PnpmState:
		return v
	case PnpmState:
		return &v
	case *string:
		if v == nil {
			return nil
		}
		val := PnpmState(*v)
		return &val
	case string:
		val := PnpmState(v)
		return &val
	default:
		panic("unsupported type")
	}
}

// Parameters for the `pnpm` Ansible module.
type PnpmParameters struct {
	// The name of a Node.js library to install.
	// All packages in `package.json` are installed if not provided.
	Name *string `json:"name,omitempty"`

	// Alias of the Node.js library.
	Alias *string `json:"alias,omitempty"`

	// The base path to install the Node.js libraries.
	Path *string `json:"path,omitempty"`

	// The version of the library to be installed, in semver format.
	Version *string `json:"version,omitempty"`

	// Install the Node.js library globally.
	// default: false
	Global *bool `json:"global,omitempty"`

	// The executable location for pnpm.
	// The default location it searches for is `PATH`, fails if not set.
	Executable *string `json:"executable,omitempty"`

	// Use the `--ignore-scripts` flag when installing.
	// default: false
	IgnoreScripts *bool `json:"ignore_scripts,omitempty"`

	// Do not install optional packages, equivalent to `--no-optional`.
	// default: false
	NoOptional *bool `json:"no_optional,omitempty"`

	// Install dependencies in production mode.
	// Pnpm will ignore any dependencies under `devDependencies` in package.json.
	// default: false
	Production *bool `json:"production,omitempty"`

	// Install dependencies in development mode.
	// Pnpm will ignore any regular dependencies in `package.json`.
	// default: false
	Dev *bool `json:"dev,omitempty"`

	// Install dependencies in optional mode.
	// default: false
	Optional *bool `json:"optional,omitempty"`

	// Installation state of the named Node.js library.
	// If `absent` is selected, a name option must be provided.
	// default: PnpmStatePresent
	State *PnpmState `json:"state,omitempty"`
}

// Wrap the `PnpmParameters into an `rpc.RPCCall`.
func (p PnpmParameters) ToRPCCall() (rpc.RPCCall[rpc.AnsibleExecuteArgs], error) {
	args, err := cast.AnyToJSONT[map[string]any](p)
	if err != nil {
		return rpc.RPCCall[rpc.AnsibleExecuteArgs]{}, err
	}
	return rpc.RPCCall[rpc.AnsibleExecuteArgs]{
		RPCFunction: rpc.RPCAnsibleExecute,
		Args: rpc.AnsibleExecuteArgs{
			Name: PnpmName,
			Args: args,
		},
	}, nil
}

// Return values for the `pnpm` Ansible module.
type PnpmReturn struct {
	AnsibleCommonReturns
}

// Unwrap the `rpc.RPCResult` into an `PnpmReturn`
func PnpmReturnFromRPCResult(r rpc.RPCResult[rpc.AnsibleExecuteResult]) (PnpmReturn, error) {
	return cast.AnyToJSONT[PnpmReturn](r.Result.Result)
}
