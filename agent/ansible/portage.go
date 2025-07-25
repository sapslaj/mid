// Code generated by ./hack/generate-ansible-types.py DO NOT EDIT
package ansible

import (
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/cast"
)

// Manages Gentoo packages.
const PortageName = "portage"

// State of the package atom.
type PortageState string

const (
	PortageStatePresent   PortageState = "present"
	PortageStateInstalled PortageState = "installed"
	PortageStateEmerged   PortageState = "emerged"
	PortageStateAbsent    PortageState = "absent"
	PortageStateRemoved   PortageState = "removed"
	PortageStateUnmerged  PortageState = "unmerged"
	PortageStateLatest    PortageState = "latest"
)

// Convert a supported type to an optional (pointer) PortageState
func OptionalPortageState[T interface {
	*PortageState | PortageState | *string | string
}](s T) *PortageState {
	switch v := any(s).(type) {
	case *PortageState:
		return v
	case PortageState:
		return &v
	case *string:
		if v == nil {
			return nil
		}
		val := PortageState(*v)
		return &val
	case string:
		val := PortageState(v)
		return &val
	default:
		panic("unsupported type")
	}
}

// Sync package repositories first.
// If `yes`, perform `emerge --sync`.
// If `web`, perform `emerge-webrsync`.
type PortageSync string

const (
	PortageSyncWeb PortageSync = "web"
	PortageSyncYes PortageSync = "yes"
	PortageSyncNo  PortageSync = "no"
)

// Convert a supported type to an optional (pointer) PortageSync
func OptionalPortageSync[T interface {
	*PortageSync | PortageSync | *string | string
}](s T) *PortageSync {
	switch v := any(s).(type) {
	case *PortageSync:
		return v
	case PortageSync:
		return &v
	case *string:
		if v == nil {
			return nil
		}
		val := PortageSync(*v)
		return &val
	case string:
		val := PortageSync(v)
		return &val
	default:
		panic("unsupported type")
	}
}

// Parameters for the `portage` Ansible module.
type PortageParameters struct {
	// Package atom or set, for example `sys-apps/foo` or `>foo-2.13` or `@world`.
	Package *[]string `json:"package,omitempty"`

	// State of the package atom.
	// default: PortageStatePresent
	State *PortageState `json:"state,omitempty"`

	// Update packages to the best version available (`--update`).
	// default: false
	Update *bool `json:"update,omitempty"`

	// Set backtrack value (`--backtrack`).
	Backtrack *int `json:"backtrack,omitempty"`

	// Consider the entire dependency tree of packages (`--deep`).
	// default: false
	Deep *bool `json:"deep,omitempty"`

	// Include installed packages where USE flags have changed (`--newuse`).
	// default: false
	Newuse *bool `json:"newuse,omitempty"`

	// Include installed packages where USE flags have changed, except when.
	// Flags that the user has not enabled are added or removed.
	// (`--changed-use`).
	// default: false
	ChangedUse *bool `json:"changed_use,omitempty"`

	// Do not add the packages to the world file (`--oneshot`).
	// default: false
	Oneshot *bool `json:"oneshot,omitempty"`

	// Do not re-emerge installed packages (`--noreplace`).
	// default: true
	Noreplace *bool `json:"noreplace,omitempty"`

	// Only merge packages but not their dependencies (`--nodeps`).
	// default: false
	Nodeps *bool `json:"nodeps,omitempty"`

	// Only merge packages' dependencies but not the packages (`--onlydeps`).
	// default: false
	Onlydeps *bool `json:"onlydeps,omitempty"`

	// Remove packages not needed by explicitly merged packages (`--depclean`).
	// If no package is specified, clean up the world's dependencies.
	// Otherwise, `--depclean` serves as a dependency aware version of `--unmerge`.
	// default: false
	Depclean *bool `json:"depclean,omitempty"`

	// Run emerge in quiet mode (`--quiet`).
	// default: false
	Quiet *bool `json:"quiet,omitempty"`

	// Run emerge in verbose mode (`--verbose`).
	// default: false
	Verbose *bool `json:"verbose,omitempty"`

	// If set to `true`, explicitely add the package to the world file.
	// Please note that this option is not used for idempotency, it is only used
	// when actually installing a package.
	Select *bool `json:"select,omitempty"`

	// Sync package repositories first.
	// If `yes`, perform `emerge --sync`.
	// If `web`, perform `emerge-webrsync`.
	Sync *PortageSync `json:"sync,omitempty"`

	// Merge only packages specified at `PORTAGE_BINHOST` in `make.conf`.
	// default: false
	Getbinpkgonly *bool `json:"getbinpkgonly,omitempty"`

	// Prefer packages specified at `PORTAGE_BINHOST` in `make.conf`.
	// default: false
	Getbinpkg *bool `json:"getbinpkg,omitempty"`

	// Merge only binaries (no compiling).
	// default: false
	Usepkgonly *bool `json:"usepkgonly,omitempty"`

	// Tries to use the binary package(s) in the locally available packages
	// directory.
	// default: false
	Usepkg *bool `json:"usepkg,omitempty"`

	// Continue as much as possible after an error.
	// default: false
	Keepgoing *bool `json:"keepgoing,omitempty"`

	// Specifies the number of packages to build simultaneously.
	// Since version 2.6: Value of `0` or `false` resets any previously added
	// `--jobs` setting values.
	Jobs *int `json:"jobs,omitempty"`

	// Specifies that no new builds should be started if there are other builds
	// running and the load average is at least LOAD.
	// Since version 2.6: Value of 0 or False resets any previously added `--load-
	// average` setting values.
	Loadavg *float64 `json:"loadavg,omitempty"`

	// Specifies that build time dependencies should be installed.
	Withbdeps *bool `json:"withbdeps,omitempty"`

	// Redirect all build output to logs alone, and do not display it on stdout
	// (`--quiet-build`).
	// default: false
	Quietbuild *bool `json:"quietbuild,omitempty"`

	// Suppresses display of the build log on stdout (--quiet-fail).
	// Only the die message and the path of the build log will be displayed on
	// stdout.
	// default: false
	Quietfail *bool `json:"quietfail,omitempty"`
}

// Wrap the `PortageParameters into an `rpc.RPCCall`.
func (p PortageParameters) ToRPCCall() (rpc.RPCCall[rpc.AnsibleExecuteArgs], error) {
	args, err := cast.AnyToJSONT[map[string]any](p)
	if err != nil {
		return rpc.RPCCall[rpc.AnsibleExecuteArgs]{}, err
	}
	return rpc.RPCCall[rpc.AnsibleExecuteArgs]{
		RPCFunction: rpc.RPCAnsibleExecute,
		Args: rpc.AnsibleExecuteArgs{
			Name: PortageName,
			Args: args,
		},
	}, nil
}

// Return values for the `portage` Ansible module.
type PortageReturn struct {
	AnsibleCommonReturns
}

// Unwrap the `rpc.RPCResult` into an `PortageReturn`
func PortageReturnFromRPCResult(r rpc.RPCResult[rpc.AnsibleExecuteResult]) (PortageReturn, error) {
	return cast.AnyToJSONT[PortageReturn](r.Result.Result)
}
