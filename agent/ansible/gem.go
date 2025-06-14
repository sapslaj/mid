// Code generated by ./hack/generate-ansible-types.py DO NOT EDIT
package ansible

import (
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/cast"
)

// Manage installation and uninstallation of Ruby gems.
const GemName = "gem"

// The desired state of the gem. `latest` ensures that the latest version is
// installed.
type GemState string

const (
	GemStatePresent GemState = "present"
	GemStateAbsent  GemState = "absent"
	GemStateLatest  GemState = "latest"
)

// Convert a supported type to an optional (pointer) GemState
func OptionalGemState[T interface {
	*GemState | GemState | *string | string
}](s T) *GemState {
	switch v := any(s).(type) {
	case *GemState:
		return v
	case GemState:
		return &v
	case *string:
		if v == nil {
			return nil
		}
		val := GemState(*v)
		return &val
	case string:
		val := GemState(v)
		return &val
	default:
		panic("unsupported type")
	}
}

// Parameters for the `gem` Ansible module.
type GemParameters struct {
	// The name of the gem to be managed.
	Name string `json:"name"`

	// The desired state of the gem. `latest` ensures that the latest version is
	// installed.
	// default: GemStatePresent
	State *GemState `json:"state,omitempty"`

	// The path to a local gem used as installation source.
	GemSource *string `json:"gem_source,omitempty"`

	// Whether to include dependencies or not.
	// default: true
	IncludeDependencies *bool `json:"include_dependencies,omitempty"`

	// The repository from which the gem will be installed.
	Repository *string `json:"repository,omitempty"`

	// Install gem in user's local gems cache or for all users.
	// default: true
	UserInstall *bool `json:"user_install,omitempty"`

	// Override the path to the gem executable.
	Executable *string `json:"executable,omitempty"`

	// Install the gems into a specific directory. These gems will be independent
	// from the global installed ones. Specifying this requires user_install to be
	// false.
	InstallDir *string `json:"install_dir,omitempty"`

	// Install executables into a specific directory.
	Bindir *string `json:"bindir,omitempty"`

	// Avoid loading any `.gemrc` file. Ignored for RubyGems prior to 2.5.2.
	// The default changed from `false` to `true` in community.general 6.0.0.
	// default: true
	Norc *bool `json:"norc,omitempty"`

	// Rewrite the shebang line on installed scripts to use /usr/bin/env.
	// default: false
	EnvShebang *bool `json:"env_shebang,omitempty"`

	// Version of the gem to be installed/removed.
	Version *string `json:"version,omitempty"`

	// Allow installation of pre-release versions of the gem.
	// default: false
	PreRelease *bool `json:"pre_release,omitempty"`

	// Install with or without docs.
	// default: false
	IncludeDoc *bool `json:"include_doc,omitempty"`

	// Allow adding build flags for gem compilation.
	BuildFlags *string `json:"build_flags,omitempty"`

	// Force gem to (un-)install, bypassing dependency checks.
	// default: false
	Force *bool `json:"force,omitempty"`
}

// Wrap the `GemParameters into an `rpc.RPCCall`.
func (p GemParameters) ToRPCCall() (rpc.RPCCall[rpc.AnsibleExecuteArgs], error) {
	args, err := cast.AnyToJSONT[map[string]any](p)
	if err != nil {
		return rpc.RPCCall[rpc.AnsibleExecuteArgs]{}, err
	}
	return rpc.RPCCall[rpc.AnsibleExecuteArgs]{
		RPCFunction: rpc.RPCAnsibleExecute,
		Args: rpc.AnsibleExecuteArgs{
			Name: GemName,
			Args: args,
		},
	}, nil
}

// Return values for the `gem` Ansible module.
type GemReturn struct {
	AnsibleCommonReturns
}

// Unwrap the `rpc.RPCResult` into an `GemReturn`
func GemReturnFromRPCResult(r rpc.RPCResult[rpc.AnsibleExecuteResult]) (GemReturn, error) {
	return cast.AnyToJSONT[GemReturn](r.Result.Result)
}
