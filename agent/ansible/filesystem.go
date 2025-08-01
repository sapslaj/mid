// Code generated by ./hack/generate-ansible-types.py DO NOT EDIT
package ansible

import (
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/cast"
)

// This module creates a filesystem.
const FilesystemName = "filesystem"

// If `state=present`, the filesystem is created if it does not already exist,
// that is the default behaviour if `state` is omitted.
// If `state=absent`, filesystem signatures on `dev` are wiped if it contains a
// filesystem (as known by `blkid`).
// When `state=absent`, all other options but `dev` are ignored, and the module
// does not fail if the device `dev` does not actually exist.
type FilesystemState string

const (
	FilesystemStatePresent FilesystemState = "present"
	FilesystemStateAbsent  FilesystemState = "absent"
)

// Convert a supported type to an optional (pointer) FilesystemState
func OptionalFilesystemState[T interface {
	*FilesystemState | FilesystemState | *string | string
}](s T) *FilesystemState {
	switch v := any(s).(type) {
	case *FilesystemState:
		return v
	case FilesystemState:
		return &v
	case *string:
		if v == nil {
			return nil
		}
		val := FilesystemState(*v)
		return &val
	case string:
		val := FilesystemState(v)
		return &val
	default:
		panic("unsupported type")
	}
}

// Filesystem type to be created. This option is required with `state=present`
// (or if `state` is omitted).
// Ufs support has been added in community.general 3.4.0.
// Bcachefs support has been added in community.general 8.6.0.
type FilesystemFstype string

const (
	FilesystemFstypeBcachefs FilesystemFstype = "bcachefs"
	FilesystemFstypeBtrfs    FilesystemFstype = "btrfs"
	FilesystemFstypeExt2     FilesystemFstype = "ext2"
	FilesystemFstypeExt3     FilesystemFstype = "ext3"
	FilesystemFstypeExt4     FilesystemFstype = "ext4"
	FilesystemFstypeExt4dev  FilesystemFstype = "ext4dev"
	FilesystemFstypeF2fs     FilesystemFstype = "f2fs"
	FilesystemFstypeLvm      FilesystemFstype = "lvm"
	FilesystemFstypeOcfs2    FilesystemFstype = "ocfs2"
	FilesystemFstypeReiserfs FilesystemFstype = "reiserfs"
	FilesystemFstypeXfs      FilesystemFstype = "xfs"
	FilesystemFstypeVfat     FilesystemFstype = "vfat"
	FilesystemFstypeSwap     FilesystemFstype = "swap"
	FilesystemFstypeUfs      FilesystemFstype = "ufs"
)

// Convert a supported type to an optional (pointer) FilesystemFstype
func OptionalFilesystemFstype[T interface {
	*FilesystemFstype | FilesystemFstype | *string | string
}](s T) *FilesystemFstype {
	switch v := any(s).(type) {
	case *FilesystemFstype:
		return v
	case FilesystemFstype:
		return &v
	case *string:
		if v == nil {
			return nil
		}
		val := FilesystemFstype(*v)
		return &val
	case string:
		val := FilesystemFstype(v)
		return &val
	default:
		panic("unsupported type")
	}
}

// Parameters for the `filesystem` Ansible module.
type FilesystemParameters struct {
	// If `state=present`, the filesystem is created if it does not already exist,
	// that is the default behaviour if `state` is omitted.
	// If `state=absent`, filesystem signatures on `dev` are wiped if it contains a
	// filesystem (as known by `blkid`).
	// When `state=absent`, all other options but `dev` are ignored, and the module
	// does not fail if the device `dev` does not actually exist.
	// default: FilesystemStatePresent
	State *FilesystemState `json:"state,omitempty"`

	// Filesystem type to be created. This option is required with `state=present`
	// (or if `state` is omitted).
	// Ufs support has been added in community.general 3.4.0.
	// Bcachefs support has been added in community.general 8.6.0.
	Fstype *FilesystemFstype `json:"fstype,omitempty"`

	// Target path to block device (Linux) or character device (FreeBSD) or regular
	// file (both).
	// When setting Linux-specific filesystem types on FreeBSD, this module only
	// works when applying to regular files, aka disk images.
	// Currently `lvm` (Linux-only) and `ufs` (FreeBSD-only) do not support a
	// regular file as their target `dev`.
	// Support for character devices on FreeBSD has been added in community.general
	// 3.4.0.
	Dev string `json:"dev"`

	// If `true`, allows to create new filesystem on devices that already has
	// filesystem.
	// default: false
	Force *bool `json:"force,omitempty"`

	// If `true`, if the block device and filesystem size differ, grow the
	// filesystem into the space.
	// Supported for `bcachefs`, `btrfs`, `ext2`, `ext3`, `ext4`, `ext4dev`,
	// `f2fs`, `lvm`, `xfs`, `ufs` and `vfat` filesystems. Attempts to resize other
	// filesystem types will fail.
	// XFS Will only grow if mounted. Currently, the module is based on commands
	// from `util-linux` package to perform operations, so resizing of XFS is not
	// supported on FreeBSD systems.
	// VFAT will likely fail if `fatresize < 1.04`.
	// Mutually exclusive with `uuid`.
	// default: false
	Resizefs *bool `json:"resizefs,omitempty"`

	// List of options to be passed to `mkfs` command.
	Opts *string `json:"opts,omitempty"`

	// Set filesystem's UUID to the given value.
	// The UUID options specified in `opts` take precedence over this value.
	// See xfs_admin(8) (`xfs`), tune2fs(8) (`ext2`, `ext3`, `ext4`, `ext4dev`) for
	// possible values.
	// For `fstype=lvm` the value is ignored, it resets the PV UUID if set.
	// Supported for `fstype` being one of `bcachefs`, `ext2`, `ext3`, `ext4`,
	// `ext4dev`, `lvm`, or `xfs`.
	// This is `not idempotent`. Specifying this option will always result in a
	// change.
	// Mutually exclusive with `resizefs`.
	Uuid *string `json:"uuid,omitempty"`
}

// Wrap the `FilesystemParameters into an `rpc.RPCCall`.
func (p FilesystemParameters) ToRPCCall() (rpc.RPCCall[rpc.AnsibleExecuteArgs], error) {
	args, err := cast.AnyToJSONT[map[string]any](p)
	if err != nil {
		return rpc.RPCCall[rpc.AnsibleExecuteArgs]{}, err
	}
	return rpc.RPCCall[rpc.AnsibleExecuteArgs]{
		RPCFunction: rpc.RPCAnsibleExecute,
		Args: rpc.AnsibleExecuteArgs{
			Name: FilesystemName,
			Args: args,
		},
	}, nil
}

// Return values for the `filesystem` Ansible module.
type FilesystemReturn struct {
	AnsibleCommonReturns
}

// Unwrap the `rpc.RPCResult` into an `FilesystemReturn`
func FilesystemReturnFromRPCResult(r rpc.RPCResult[rpc.AnsibleExecuteResult]) (FilesystemReturn, error) {
	return cast.AnyToJSONT[FilesystemReturn](r.Result.Result)
}
