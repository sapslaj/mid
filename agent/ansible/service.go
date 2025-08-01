// Code generated by ./hack/generate-ansible-types.py DO NOT EDIT
package ansible

import (
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/cast"
)

// Controls services on remote hosts. Supported init systems include BSD init,
// OpenRC, SysV, Solaris SMF, systemd, upstart.
// This module acts as a proxy to the underlying service manager module. While
// all arguments will be passed to the underlying module, not all modules
// support the same arguments. This documentation only covers the minimum
// intersection of module arguments that all service manager modules support.
// This module is a proxy for multiple more specific service manager modules
// (such as `ansible.builtin.systemd` and `ansible.builtin.sysvinit`). This
// allows management of a heterogeneous environment of machines without creating
// a specific task for each service manager. The module to be executed is
// determined by the `use` option, which defaults to the service manager
// discovered by `ansible.builtin.setup`.  If `ansible.builtin.setup` was not
// yet run, this module may run it.
// For Windows targets, use the `ansible.windows.win_service` module instead.
const ServiceName = "service"

// `started`/`stopped` are idempotent actions that will not run commands unless
// necessary.
// `restarted` will always bounce the service.
// `reloaded` will always reload.
// At least one of `state` and `enabled` are required.
// Note that `reloaded` will start the service if it is not already started,
// even if your chosen init system wouldn't normally.
type ServiceState string

const (
	ServiceStateReloaded  ServiceState = "reloaded"
	ServiceStateRestarted ServiceState = "restarted"
	ServiceStateStarted   ServiceState = "started"
	ServiceStateStopped   ServiceState = "stopped"
)

// Convert a supported type to an optional (pointer) ServiceState
func OptionalServiceState[T interface {
	*ServiceState | ServiceState | *string | string
}](s T) *ServiceState {
	switch v := any(s).(type) {
	case *ServiceState:
		return v
	case ServiceState:
		return &v
	case *string:
		if v == nil {
			return nil
		}
		val := ServiceState(*v)
		return &val
	case string:
		val := ServiceState(v)
		return &val
	default:
		panic("unsupported type")
	}
}

// Parameters for the `service` Ansible module.
type ServiceParameters struct {
	// Name of the service.
	Name string `json:"name"`

	// `started`/`stopped` are idempotent actions that will not run commands unless
	// necessary.
	// `restarted` will always bounce the service.
	// `reloaded` will always reload.
	// At least one of `state` and `enabled` are required.
	// Note that `reloaded` will start the service if it is not already started,
	// even if your chosen init system wouldn't normally.
	State *ServiceState `json:"state,omitempty"`

	// If the service is being `restarted` then sleep this many seconds between the
	// stop and start command.
	// This helps to work around badly-behaving init scripts that exit immediately
	// after signaling a process to stop.
	// Not all service managers support sleep, i.e when using systemd this setting
	// will be ignored.
	Sleep *int `json:"sleep,omitempty"`

	// If the service does not respond to the status command, name a substring to
	// look for as would be found in the output of the `ps` command as a stand-in
	// for a status result.
	// If the string is found, the service will be assumed to be started.
	// While using remote hosts with systemd this setting will be ignored.
	Pattern *string `json:"pattern,omitempty"`

	// Whether the service should start on boot.
	// At least one of `state` and `enabled` are required.
	Enabled *bool `json:"enabled,omitempty"`

	// For OpenRC init scripts (e.g. Gentoo) only.
	// The runlevel that this service belongs to.
	// While using remote hosts with systemd this setting will be ignored.
	// default: "default"
	Runlevel *string `json:"runlevel,omitempty"`

	// Additional arguments provided on the command line.
	// While using remote hosts with systemd this setting will be ignored.
	// default: ""
	Arguments *string `json:"arguments,omitempty"`

	// The service module actually uses system specific modules, normally through
	// auto detection, this setting can force a specific module.
	// Normally it uses the value of the `ansible_service_mgr` fact and falls back
	// to the `ansible.legacy.service` module when none matching is found.
	// The 'old service module' still uses autodetection and in no way does it
	// correspond to the `service` command.
	// default: "auto"
	Use *string `json:"use,omitempty"`
}

// Wrap the `ServiceParameters into an `rpc.RPCCall`.
func (p ServiceParameters) ToRPCCall() (rpc.RPCCall[rpc.AnsibleExecuteArgs], error) {
	args, err := cast.AnyToJSONT[map[string]any](p)
	if err != nil {
		return rpc.RPCCall[rpc.AnsibleExecuteArgs]{}, err
	}
	return rpc.RPCCall[rpc.AnsibleExecuteArgs]{
		RPCFunction: rpc.RPCAnsibleExecute,
		Args: rpc.AnsibleExecuteArgs{
			Name: ServiceName,
			Args: args,
		},
	}, nil
}

// Return values for the `service` Ansible module.
type ServiceReturn struct {
	AnsibleCommonReturns
}

// Unwrap the `rpc.RPCResult` into an `ServiceReturn`
func ServiceReturnFromRPCResult(r rpc.RPCResult[rpc.AnsibleExecuteResult]) (ServiceReturn, error) {
	return cast.AnyToJSONT[ServiceReturn](r.Result.Result)
}
