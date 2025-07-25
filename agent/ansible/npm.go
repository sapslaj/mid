// Code generated by ./hack/generate-ansible-types.py DO NOT EDIT
package ansible

import (
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/cast"
)

// Manage node.js packages with Node Package Manager (npm).
const NpmName = "npm"

// The state of the node.js library.
type NpmState string

const (
	NpmStatePresent NpmState = "present"
	NpmStateAbsent  NpmState = "absent"
	NpmStateLatest  NpmState = "latest"
)

// Convert a supported type to an optional (pointer) NpmState
func OptionalNpmState[T interface {
	*NpmState | NpmState | *string | string
}](s T) *NpmState {
	switch v := any(s).(type) {
	case *NpmState:
		return v
	case NpmState:
		return &v
	case *string:
		if v == nil {
			return nil
		}
		val := NpmState(*v)
		return &val
	case string:
		val := NpmState(v)
		return &val
	default:
		panic("unsupported type")
	}
}

// Parameters for the `npm` Ansible module.
type NpmParameters struct {
	// The name of a node.js library to install.
	Name *string `json:"name,omitempty"`

	// The base path where to install the node.js libraries.
	Path *string `json:"path,omitempty"`

	// The version to be installed.
	Version *string `json:"version,omitempty"`

	// Install the node.js library globally.
	// default: false
	Global *bool `json:"global,omitempty"`

	// The executable location for npm.
	// This is useful if you are using a version manager, such as nvm.
	Executable *string `json:"executable,omitempty"`

	// Use the `--ignore-scripts` flag when installing.
	// default: false
	IgnoreScripts *bool `json:"ignore_scripts,omitempty"`

	// Use the `--unsafe-perm` flag when installing.
	// default: false
	UnsafePerm *bool `json:"unsafe_perm,omitempty"`

	// Install packages based on package-lock file, same as running `npm ci`.
	// default: false
	Ci *bool `json:"ci,omitempty"`

	// Install dependencies in production mode, excluding devDependencies.
	// default: false
	Production *bool `json:"production,omitempty"`

	// The registry to install modules from.
	Registry *string `json:"registry,omitempty"`

	// The state of the node.js library.
	// default: NpmStatePresent
	State *NpmState `json:"state,omitempty"`

	// Use the `--no-optional` flag when installing.
	// default: false
	NoOptional *bool `json:"no_optional,omitempty"`

	// Use the `--no-bin-links` flag when installing.
	// default: false
	NoBinLinks *bool `json:"no_bin_links,omitempty"`

	// Use the `--force` flag when installing.
	// default: false
	Force *bool `json:"force,omitempty"`
}

// Wrap the `NpmParameters into an `rpc.RPCCall`.
func (p NpmParameters) ToRPCCall() (rpc.RPCCall[rpc.AnsibleExecuteArgs], error) {
	args, err := cast.AnyToJSONT[map[string]any](p)
	if err != nil {
		return rpc.RPCCall[rpc.AnsibleExecuteArgs]{}, err
	}
	return rpc.RPCCall[rpc.AnsibleExecuteArgs]{
		RPCFunction: rpc.RPCAnsibleExecute,
		Args: rpc.AnsibleExecuteArgs{
			Name: NpmName,
			Args: args,
		},
	}, nil
}

// Return values for the `npm` Ansible module.
type NpmReturn struct {
	AnsibleCommonReturns
}

// Unwrap the `rpc.RPCResult` into an `NpmReturn`
func NpmReturnFromRPCResult(r rpc.RPCResult[rpc.AnsibleExecuteResult]) (NpmReturn, error) {
	return cast.AnyToJSONT[NpmReturn](r.Result.Result)
}
