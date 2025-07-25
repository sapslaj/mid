// Code generated by ./hack/generate-ansible-types.py DO NOT EDIT
package ansible

import (
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/cast"
)

// The `ansible.builtin.unarchive` module unpacks an archive. It will not unpack
// a compressed file that does not contain an archive.
// By default, it will copy the source file from the local system to the target
// before unpacking.
// Set `remote_src=yes` to unpack an archive which already exists on the target.
// If checksum validation is desired, use `ansible.builtin.get_url` or
// `ansible.builtin.uri` instead to fetch the file and set `remote_src=yes`.
// For Windows targets, use the `community.windows.win_unzip` module instead.
const UnarchiveName = "unarchive"

// Parameters for the `unarchive` Ansible module.
type UnarchiveParameters struct {
	// If `remote_src=no` (default), local path to archive file to copy to the
	// target server; can be absolute or relative. If `remote_src=yes`, path on the
	// target server to existing archive file to unpack.
	// If `remote_src=yes` and `src` contains `://`, the remote machine will
	// download the file from the URL first. (version_added 2.0). This is only for
	// simple cases, for full download support use the `ansible.builtin.get_url`
	// module.
	Src string `json:"src"`

	// Remote absolute path where the archive should be unpacked.
	// The given path must exist. Base directory is not created by this module.
	Dest string `json:"dest"`

	// If true, the file is copied from local controller to the managed (remote)
	// node, otherwise, the plugin will look for src archive on the managed
	// machine.
	// This option has been deprecated in favor of `remote_src`.
	// This option is mutually exclusive with `remote_src`.
	// default: true
	Copy *bool `json:"copy,omitempty"`

	// If the specified absolute path (file or directory) already exists, this step
	// will `not` be run.
	// The specified absolute path (file or directory) must be below the base path
	// given with `dest`.
	Creates *string `json:"creates,omitempty"`

	// Size of the volatile memory buffer that is used for extracting files from
	// the archive in bytes.
	// default: 65536
	IoBufferSize *int `json:"io_buffer_size,omitempty"`

	// If set to True, return the list of files that are contained in the tarball.
	// default: false
	ListFiles *bool `json:"list_files,omitempty"`

	// List the directory and file entries that you would like to exclude from the
	// unarchive action.
	// Mutually exclusive with `include`.
	// default: []
	Exclude *[]string `json:"exclude,omitempty"`

	// List of directory and file entries that you would like to extract from the
	// archive. If `include` is not empty, only files listed here will be
	// extracted.
	// Mutually exclusive with `exclude`.
	// default: []
	Include *[]string `json:"include,omitempty"`

	// Do not replace existing files that are newer than files from the archive.
	// default: false
	KeepNewer *bool `json:"keep_newer,omitempty"`

	// Specify additional options by passing in an array.
	// Each space-separated command-line option should be a new element of the
	// array. See examples.
	// Command-line options with multiple elements must use multiple lines in the
	// array, one for each element.
	// default: []
	ExtraOpts *[]string `json:"extra_opts,omitempty"`

	// Set to `true` to indicate the archived file is already on the remote system
	// and not local to the Ansible controller.
	// This option is mutually exclusive with `copy`.
	// default: false
	RemoteSrc *bool `json:"remote_src,omitempty"`

	// This only applies if using a https URL as the source of the file.
	// This should only set to `false` used on personally controlled sites using
	// self-signed certificate.
	// Prior to 2.2 the code worked as if this was set to `true`.
	// default: true
	ValidateCerts *bool `json:"validate_certs,omitempty"`

	// This option controls the auto-decryption of source files using vault.
	// default: true
	Decrypt *bool `json:"decrypt,omitempty"`

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

// Wrap the `UnarchiveParameters into an `rpc.RPCCall`.
func (p UnarchiveParameters) ToRPCCall() (rpc.RPCCall[rpc.AnsibleExecuteArgs], error) {
	args, err := cast.AnyToJSONT[map[string]any](p)
	if err != nil {
		return rpc.RPCCall[rpc.AnsibleExecuteArgs]{}, err
	}
	return rpc.RPCCall[rpc.AnsibleExecuteArgs]{
		RPCFunction: rpc.RPCAnsibleExecute,
		Args: rpc.AnsibleExecuteArgs{
			Name: UnarchiveName,
			Args: args,
		},
	}, nil
}

// Return values for the `unarchive` Ansible module.
type UnarchiveReturn struct {
	AnsibleCommonReturns

	// Path to the destination directory.
	Dest *string `json:"dest,omitempty"`

	// List of all the files in the archive.
	Files *[]any `json:"files,omitempty"`

	// Numerical ID of the group that owns the destination directory.
	Gid *int `json:"gid,omitempty"`

	// Name of the group that owns the destination directory.
	Group *string `json:"group,omitempty"`

	// Archive software handler used to extract and decompress the archive.
	Handler *string `json:"handler,omitempty"`

	// String that represents the octal permissions of the destination directory.
	Mode *string `json:"mode,omitempty"`

	// Name of the user that owns the destination directory.
	Owner *string `json:"owner,omitempty"`

	// The size of destination directory in bytes. Does not include the size of
	// files or subdirectories contained within.
	Size *int `json:"size,omitempty"`

	// The source archive's path.
	// If `src` was a remote web URL, or from the local ansible controller, this
	// shows the temporary location where the download was stored.
	Src *string `json:"src,omitempty"`

	// State of the destination. Effectively always "directory".
	State *string `json:"state,omitempty"`

	// Numerical ID of the user that owns the destination directory.
	Uid *int `json:"uid,omitempty"`
}

// Unwrap the `rpc.RPCResult` into an `UnarchiveReturn`
func UnarchiveReturnFromRPCResult(r rpc.RPCResult[rpc.AnsibleExecuteResult]) (UnarchiveReturn, error) {
	return cast.AnyToJSONT[UnarchiveReturn](r.Result.Result)
}
