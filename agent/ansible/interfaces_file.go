// Code generated by ./hack/generate-ansible-types.py DO NOT EDIT
package ansible

import (
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/cast"
)

// Manage (add, remove, change) individual interface options in an interfaces-
// style file without having to manage the file as a whole with, say,
// `ansible.builtin.template` or `ansible.builtin.assemble`. Interface has to be
// presented in a file.
// Read information about interfaces from interfaces-styled files.
const InterfacesFileName = "interfaces_file"

// If set to `absent` the option or section will be removed if present instead
// of created.
type InterfacesFileState string

const (
	InterfacesFileStatePresent InterfacesFileState = "present"
	InterfacesFileStateAbsent  InterfacesFileState = "absent"
)

// Convert a supported type to an optional (pointer) InterfacesFileState
func OptionalInterfacesFileState[T interface {
	*InterfacesFileState | InterfacesFileState | *string | string
}](s T) *InterfacesFileState {
	switch v := any(s).(type) {
	case *InterfacesFileState:
		return v
	case InterfacesFileState:
		return &v
	case *string:
		if v == nil {
			return nil
		}
		val := InterfacesFileState(*v)
		return &val
	case string:
		val := InterfacesFileState(v)
		return &val
	default:
		panic("unsupported type")
	}
}

// Parameters for the `interfaces_file` Ansible module.
type InterfacesFileParameters struct {
	// Path to the interfaces file.
	// default: "/etc/network/interfaces"
	Dest *string `json:"dest,omitempty"`

	// Name of the interface, required for value changes or option remove.
	Iface *string `json:"iface,omitempty"`

	// Address family of the interface, useful if same interface name is used for
	// both `inet` and `inet6`.
	AddressFamily *string `json:"address_family,omitempty"`

	// Name of the option, required for value changes or option remove.
	Option *string `json:"option,omitempty"`

	// If `option` is not presented for the `iface` and `state` is `present` option
	// will be added. If `option` already exists and is not `pre-up`, `up`, `post-
	// up` or `down`, its value will be updated. `pre-up`, `up`, `post-up` and
	// `down` options cannot be updated, only adding new options, removing existing
	// ones or cleaning the whole option set are supported.
	Value *string `json:"value,omitempty"`

	// Create a backup file including the timestamp information so you can get the
	// original file back if you somehow clobbered it incorrectly.
	// default: false
	Backup *bool `json:"backup,omitempty"`

	// If set to `absent` the option or section will be removed if present instead
	// of created.
	// default: InterfacesFileStatePresent
	State *InterfacesFileState `json:"state,omitempty"`

	// The permissions the resulting filesystem object should have.
	// For those used to `/usr/bin/chmod` remember that modes are actually octal
	// numbers. You must give Ansible enough information to parse them correctly.
	// For consistent results, quote octal numbers (for example, `'644'` or
	// `'1777'`) so Ansible receives a string and can do its own conversion from
	// string into number. Adding a leading zero (for example, `0755`) works
	// sometimes, but can fail in loops and some other circumstances.
	// Giving Ansible a number without following either of these rules will end up
	// with a decimal number which will have unexpected results.
	// As of Ansible 1.8, the mode may be specified as a symbolic mode (for
	// example, `u+rwx` or `u=rw,g=r,o=r`).
	// If `mode` is not specified and the destination filesystem object `does not`
	// exist, the default `umask` on the system will be used when setting the mode
	// for the newly created filesystem object.
	// If `mode` is not specified and the destination filesystem object `does`
	// exist, the mode of the existing filesystem object will be used.
	// Specifying `mode` is the best way to ensure filesystem objects are created
	// with the correct permissions. See CVE-2020-1736 for further details.
	Mode *any `json:"mode,omitempty"`

	// Name of the user that should own the filesystem object, as would be fed to
	// `chown`.
	// When left unspecified, it uses the current user unless you are root, in
	// which case it can preserve the previous ownership.
	// Specifying a numeric username will be assumed to be a user ID and not a
	// username. Avoid numeric usernames to avoid this confusion.
	Owner *string `json:"owner,omitempty"`

	// Name of the group that should own the filesystem object, as would be fed to
	// `chown`.
	// When left unspecified, it uses the current group of the current user unless
	// you are root, in which case it can preserve the previous ownership.
	Group *string `json:"group,omitempty"`

	// The user part of the SELinux filesystem object context.
	// By default it uses the `system` policy, where applicable.
	// When set to `_default`, it will use the `user` portion of the policy if
	// available.
	Seuser *string `json:"seuser,omitempty"`

	// The role part of the SELinux filesystem object context.
	// When set to `_default`, it will use the `role` portion of the policy if
	// available.
	Serole *string `json:"serole,omitempty"`

	// The type part of the SELinux filesystem object context.
	// When set to `_default`, it will use the `type` portion of the policy if
	// available.
	Setype *string `json:"setype,omitempty"`

	// The level part of the SELinux filesystem object context.
	// This is the MLS/MCS attribute, sometimes known as the `range`.
	// When set to `_default`, it will use the `level` portion of the policy if
	// available.
	Selevel *string `json:"selevel,omitempty"`

	// Influence when to use atomic operation to prevent data corruption or
	// inconsistent reads from the target filesystem object.
	// By default this module uses atomic operations to prevent data corruption or
	// inconsistent reads from the target filesystem objects, but sometimes systems
	// are configured or just broken in ways that prevent this. One example is
	// docker mounted filesystem objects, which cannot be updated atomically from
	// inside the container and can only be written in an unsafe manner.
	// This option allows Ansible to fall back to unsafe methods of updating
	// filesystem objects when atomic operations fail (however, it doesn't force
	// Ansible to perform unsafe writes).
	// IMPORTANT! Unsafe writes are subject to race conditions and can lead to data
	// corruption.
	// default: false
	UnsafeWrites *bool `json:"unsafe_writes,omitempty"`

	// The attributes the resulting filesystem object should have.
	// To get supported flags look at the man page for `chattr` on the target
	// system.
	// This string should contain the attributes in the same order as the one
	// displayed by `lsattr`.
	// The `=` operator is assumed as default, otherwise `+` or `-` operators need
	// to be included in the string.
	Attributes *string `json:"attributes,omitempty"`
}

// Wrap the `InterfacesFileParameters into an `rpc.RPCCall`.
func (p InterfacesFileParameters) ToRPCCall() (rpc.RPCCall[rpc.AnsibleExecuteArgs], error) {
	args, err := cast.AnyToJSONT[map[string]any](p)
	if err != nil {
		return rpc.RPCCall[rpc.AnsibleExecuteArgs]{}, err
	}
	return rpc.RPCCall[rpc.AnsibleExecuteArgs]{
		RPCFunction: rpc.RPCAnsibleExecute,
		Args: rpc.AnsibleExecuteArgs{
			Name: InterfacesFileName,
			Args: args,
		},
	}, nil
}

// Return values for the `interfaces_file` Ansible module.
type InterfacesFileReturn struct {
	AnsibleCommonReturns

	// Destination file/path.
	Dest *string `json:"dest,omitempty"`

	// Interfaces dictionary.
	Ifaces *map[string]any `json:"ifaces,omitempty"`
}

// Unwrap the `rpc.RPCResult` into an `InterfacesFileReturn`
func InterfacesFileReturnFromRPCResult(r rpc.RPCResult[rpc.AnsibleExecuteResult]) (InterfacesFileReturn, error) {
	return cast.AnyToJSONT[InterfacesFileReturn](r.Result.Result)
}
