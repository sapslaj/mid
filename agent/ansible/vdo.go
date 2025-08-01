// Code generated by ./hack/generate-ansible-types.py DO NOT EDIT
package ansible

import (
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/cast"
)

// This module controls the VDO dedupe and compression device.
// VDO, or Virtual Data Optimizer, is a device-mapper target that provides
// inline block-level deduplication, compression, and thin provisioning
// capabilities to primary storage.
const VdoName = "vdo"

// Whether this VDO volume should be `present` or `absent`. If a `present` VDO
// volume does not exist, it is created. If a `present` VDO volume already
// exists, it is modified by updating the configuration, which takes effect when
// the VDO volume is restarted. Not all parameters of an existing VDO volume can
// be modified; the `statusparamkeys` list in the code contains the parameters
// that can be modified after creation. If an `absent` VDO volume does not
// exist, it is not removed.
type VdoState string

const (
	VdoStateAbsent  VdoState = "absent"
	VdoStatePresent VdoState = "present"
)

// Convert a supported type to an optional (pointer) VdoState
func OptionalVdoState[T interface {
	*VdoState | VdoState | *string | string
}](s T) *VdoState {
	switch v := any(s).(type) {
	case *VdoState:
		return v
	case VdoState:
		return &v
	case *string:
		if v == nil {
			return nil
		}
		val := VdoState(*v)
		return &val
	case string:
		val := VdoState(v)
		return &val
	default:
		panic("unsupported type")
	}
}

// Configures whether deduplication is enabled. The default for a created volume
// is `enabled`. Existing volumes maintain their previously configured setting
// unless a different value is specified in the playbook.
type VdoDeduplication string

const (
	VdoDeduplicationDisabled VdoDeduplication = "disabled"
	VdoDeduplicationEnabled  VdoDeduplication = "enabled"
)

// Convert a supported type to an optional (pointer) VdoDeduplication
func OptionalVdoDeduplication[T interface {
	*VdoDeduplication | VdoDeduplication | *string | string
}](s T) *VdoDeduplication {
	switch v := any(s).(type) {
	case *VdoDeduplication:
		return v
	case VdoDeduplication:
		return &v
	case *string:
		if v == nil {
			return nil
		}
		val := VdoDeduplication(*v)
		return &val
	case string:
		val := VdoDeduplication(v)
		return &val
	default:
		panic("unsupported type")
	}
}

// Configures whether compression is enabled. The default for a created volume
// is `enabled`. Existing volumes maintain their previously configured setting
// unless a different value is specified in the playbook.
type VdoCompression string

const (
	VdoCompressionDisabled VdoCompression = "disabled"
	VdoCompressionEnabled  VdoCompression = "enabled"
)

// Convert a supported type to an optional (pointer) VdoCompression
func OptionalVdoCompression[T interface {
	*VdoCompression | VdoCompression | *string | string
}](s T) *VdoCompression {
	switch v := any(s).(type) {
	case *VdoCompression:
		return v
	case VdoCompression:
		return &v
	case *string:
		if v == nil {
			return nil
		}
		val := VdoCompression(*v)
		return &val
	case string:
		val := VdoCompression(v)
		return &val
	default:
		panic("unsupported type")
	}
}

// Enables or disables the read cache. The default is `disabled`. Choosing
// `enabled` enables a read cache which may improve performance for workloads of
// high deduplication, read workloads with a high level of compression, or on
// hard disk storage. Existing volumes maintain their previously configured
// setting unless a different value is specified in the playbook.
// The read cache feature is available in VDO 6.1 and older.
type VdoReadcache string

const (
	VdoReadcacheDisabled VdoReadcache = "disabled"
	VdoReadcacheEnabled  VdoReadcache = "enabled"
)

// Convert a supported type to an optional (pointer) VdoReadcache
func OptionalVdoReadcache[T interface {
	*VdoReadcache | VdoReadcache | *string | string
}](s T) *VdoReadcache {
	switch v := any(s).(type) {
	case *VdoReadcache:
		return v
	case VdoReadcache:
		return &v
	case *string:
		if v == nil {
			return nil
		}
		val := VdoReadcache(*v)
		return &val
	case string:
		val := VdoReadcache(v)
		return &val
	default:
		panic("unsupported type")
	}
}

// Specifies the write policy of the VDO volume.
// The `sync` mode acknowledges writes only after data is on stable storage.
// The `async` mode acknowledges writes when data has been cached for writing to
// stable storage.
// The default (and highly recommended) `auto` mode checks the storage device to
// determine whether it supports flushes. Devices that support flushes result in
// a VDO volume in `async` mode, while devices that do not support flushes run
// in `sync` mode.
// Existing volumes maintain their previously configured setting unless a
// different value is specified in the playbook.
type VdoWritepolicy string

const (
	VdoWritepolicyAsync VdoWritepolicy = "async"
	VdoWritepolicyAuto  VdoWritepolicy = "auto"
	VdoWritepolicySync  VdoWritepolicy = "sync"
)

// Convert a supported type to an optional (pointer) VdoWritepolicy
func OptionalVdoWritepolicy[T interface {
	*VdoWritepolicy | VdoWritepolicy | *string | string
}](s T) *VdoWritepolicy {
	switch v := any(s).(type) {
	case *VdoWritepolicy:
		return v
	case VdoWritepolicy:
		return &v
	case *string:
		if v == nil {
			return nil
		}
		val := VdoWritepolicy(*v)
		return &val
	case string:
		val := VdoWritepolicy(v)
		return &val
	default:
		panic("unsupported type")
	}
}

// Specifies the index mode of the Albireo index.
// The default is `dense`, which has a deduplication window of 1 GB of index
// memory per 1 TB of incoming data, requiring 10 GB of index data on persistent
// storage.
// The `sparse` mode has a deduplication window of 1 GB of index memory per 10
// TB of incoming data, but requires 100 GB of index data on persistent storage.
// This option is only available when creating a new volume, and cannot be
// changed for an existing volume.
type VdoIndexmode string

const (
	VdoIndexmodeDense  VdoIndexmode = "dense"
	VdoIndexmodeSparse VdoIndexmode = "sparse"
)

// Convert a supported type to an optional (pointer) VdoIndexmode
func OptionalVdoIndexmode[T interface {
	*VdoIndexmode | VdoIndexmode | *string | string
}](s T) *VdoIndexmode {
	switch v := any(s).(type) {
	case *VdoIndexmode:
		return v
	case VdoIndexmode:
		return &v
	case *string:
		if v == nil {
			return nil
		}
		val := VdoIndexmode(*v)
		return &val
	case string:
		val := VdoIndexmode(v)
		return &val
	default:
		panic("unsupported type")
	}
}

// Parameters for the `vdo` Ansible module.
type VdoParameters struct {
	// The name of the VDO volume.
	Name string `json:"name"`

	// Whether this VDO volume should be `present` or `absent`. If a `present` VDO
	// volume does not exist, it is created. If a `present` VDO volume already
	// exists, it is modified by updating the configuration, which takes effect
	// when the VDO volume is restarted. Not all parameters of an existing VDO
	// volume can be modified; the `statusparamkeys` list in the code contains the
	// parameters that can be modified after creation. If an `absent` VDO volume
	// does not exist, it is not removed.
	// default: VdoStatePresent
	State *VdoState `json:"state,omitempty"`

	// The `activate` status for a VDO volume. If this is set to `false`, the VDO
	// volume cannot be started, and it will not start on system startup. However,
	// on initial creation, a VDO volume with "activated" set to "off" will be
	// running, until stopped. This is the default behavior of the `vdo create`
	// command; it provides the user an opportunity to write a base amount of
	// metadata (filesystem, LVM headers, and so on) to the VDO volume prior to
	// stopping the volume, and leaving it deactivated until ready to use.
	Activated *bool `json:"activated,omitempty"`

	// Whether this VDO volume is running.
	// A VDO volume must be activated in order to be started.
	Running *bool `json:"running,omitempty"`

	// The full path of the device to use for VDO storage.
	// This is required if `state=present`.
	Device *string `json:"device,omitempty"`

	// The logical size of the VDO volume (in megabytes, or LVM suffix format). If
	// not specified for a new volume, this defaults to the same size as the
	// underlying storage device, which is specified in the `device` parameter.
	// Existing volumes maintain their size if the `logicalsize` parameter is not
	// specified, or is smaller than or identical to the current size. If the
	// specified size is larger than the current size, a `growlogical` operation is
	// performed.
	Logicalsize *string `json:"logicalsize,omitempty"`

	// Configures whether deduplication is enabled. The default for a created
	// volume is `enabled`. Existing volumes maintain their previously configured
	// setting unless a different value is specified in the playbook.
	Deduplication *VdoDeduplication `json:"deduplication,omitempty"`

	// Configures whether compression is enabled. The default for a created volume
	// is `enabled`. Existing volumes maintain their previously configured setting
	// unless a different value is specified in the playbook.
	Compression *VdoCompression `json:"compression,omitempty"`

	// The amount of memory allocated for caching block map pages, in megabytes (or
	// may be issued with an LVM-style suffix of K, M, G, or T). The default (and
	// minimum) value is `128M`. The value specifies the size of the cache; there
	// is a 15% memory usage overhead. Each 1.25G of block map covers 1T of logical
	// blocks, therefore a small amount of block map cache memory can cache a
	// significantly large amount of block map data.
	// Existing volumes maintain their previously configured setting unless a
	// different value is specified in the playbook.
	Blockmapcachesize *string `json:"blockmapcachesize,omitempty"`

	// Enables or disables the read cache. The default is `disabled`. Choosing
	// `enabled` enables a read cache which may improve performance for workloads
	// of high deduplication, read workloads with a high level of compression, or
	// on hard disk storage. Existing volumes maintain their previously configured
	// setting unless a different value is specified in the playbook.
	// The read cache feature is available in VDO 6.1 and older.
	Readcache *VdoReadcache `json:"readcache,omitempty"`

	// Specifies the extra VDO device read cache size in megabytes. This is in
	// addition to a system-defined minimum. Using a value with a suffix of K, M,
	// G, or T is optional. The default value is `0`. 1.125 MB of memory per bio
	// thread is used per 1 MB of read cache specified (for example, a VDO volume
	// configured with 4 bio threads has a read cache memory usage overhead of 4.5
	// MB per 1 MB of read cache specified). Existing volumes maintain their
	// previously configured setting unless a different value is specified in the
	// playbook.
	// The read cache feature is available in VDO 6.1 and older.
	Readcachesize *string `json:"readcachesize,omitempty"`

	// Enables 512-byte emulation mode, allowing drivers or filesystems to access
	// the VDO volume at 512-byte granularity, instead of the default 4096-byte
	// granularity.
	// Only recommended when a driver or filesystem requires 512-byte sector level
	// access to a device.
	// This option is only available when creating a new volume, and cannot be
	// changed for an existing volume.
	// default: false
	Emulate512 *bool `json:"emulate512,omitempty"`

	// Specifies whether to attempt to execute a `growphysical` operation, if there
	// is enough unused space on the device. A `growphysical` operation is executed
	// if there is at least 64 GB of free space, relative to the previous physical
	// size of the affected VDO volume.
	// default: false
	Growphysical *bool `json:"growphysical,omitempty"`

	// The size of the increment by which the physical size of a VDO volume is
	// grown, in megabytes (or may be issued with an LVM-style suffix of K, M, G,
	// or T). Must be a power of two between 128M and 32G. The default is `2G`,
	// which supports volumes having a physical size up to 16T. The maximum, `32G`,
	// supports a physical size of up to 256T. This option is only available when
	// creating a new volume, and cannot be changed for an existing volume.
	Slabsize *string `json:"slabsize,omitempty"`

	// Specifies the write policy of the VDO volume.
	// The `sync` mode acknowledges writes only after data is on stable storage.
	// The `async` mode acknowledges writes when data has been cached for writing
	// to stable storage.
	// The default (and highly recommended) `auto` mode checks the storage device
	// to determine whether it supports flushes. Devices that support flushes
	// result in a VDO volume in `async` mode, while devices that do not support
	// flushes run in `sync` mode.
	// Existing volumes maintain their previously configured setting unless a
	// different value is specified in the playbook.
	Writepolicy *VdoWritepolicy `json:"writepolicy,omitempty"`

	// Specifies the amount of index memory in gigabytes. The default is `0.25`.
	// The special decimal values `0.25`, `0.5`, and `0.75` can be used, as can any
	// positive integer. This option is only available when creating a new volume,
	// and cannot be changed for an existing volume.
	Indexmem *string `json:"indexmem,omitempty"`

	// Specifies the index mode of the Albireo index.
	// The default is `dense`, which has a deduplication window of 1 GB of index
	// memory per 1 TB of incoming data, requiring 10 GB of index data on
	// persistent storage.
	// The `sparse` mode has a deduplication window of 1 GB of index memory per 10
	// TB of incoming data, but requires 100 GB of index data on persistent
	// storage.
	// This option is only available when creating a new volume, and cannot be
	// changed for an existing volume.
	Indexmode *VdoIndexmode `json:"indexmode,omitempty"`

	// Specifies the number of threads to use for acknowledging completion of
	// requested VDO I/O operations. Valid values are integer values from `1` to
	// `100` (lower numbers are preferable due to overhead). The default is `1`.
	// Existing volumes maintain their previously configured setting unless a
	// different value is specified in the playbook.
	Ackthreads *string `json:"ackthreads,omitempty"`

	// Specifies the number of threads to use for submitting I/O operations to the
	// storage device. Valid values are integer values from `1` to `100` (lower
	// numbers are preferable due to overhead). The default is `4`. Existing
	// volumes maintain their previously configured setting unless a different
	// value is specified in the playbook.
	Biothreads *string `json:"biothreads,omitempty"`

	// Specifies the number of threads to use for CPU-intensive work such as
	// hashing or compression. Valid values are integer values from `1` to `100`
	// (lower numbers are preferable due to overhead). The default is `2`. Existing
	// volumes maintain their previously configured setting unless a different
	// value is specified in the playbook.
	Cputhreads *string `json:"cputhreads,omitempty"`

	// Specifies the number of threads across which to subdivide parts of the VDO
	// processing based on logical block addresses. Valid values are integer values
	// from `1` to `100` (lower numbers are preferable due to overhead). The
	// default is `1`. Existing volumes maintain their previously configured
	// setting unless a different value is specified in the playbook.
	Logicalthreads *string `json:"logicalthreads,omitempty"`

	// Specifies the number of threads across which to subdivide parts of the VDO
	// processing based on physical block addresses. Valid values are integer
	// values from `1` to `16` (lower numbers are preferable due to overhead). The
	// physical space used by the VDO volume must be larger than (`slabsize` *
	// `physicalthreads`). The default is `1`. Existing volumes maintain their
	// previously configured setting unless a different value is specified in the
	// playbook.
	Physicalthreads *string `json:"physicalthreads,omitempty"`

	// When creating a volume, ignores any existing file system or VDO signature
	// already present in the storage device. When stopping or removing a VDO
	// volume, first unmounts the file system stored on the device if mounted.
	// `Warning:` Since this parameter removes all safety checks it is important to
	// make sure that all parameters provided are accurate and intentional.
	// default: false
	Force *bool `json:"force,omitempty"`
}

// Wrap the `VdoParameters into an `rpc.RPCCall`.
func (p VdoParameters) ToRPCCall() (rpc.RPCCall[rpc.AnsibleExecuteArgs], error) {
	args, err := cast.AnyToJSONT[map[string]any](p)
	if err != nil {
		return rpc.RPCCall[rpc.AnsibleExecuteArgs]{}, err
	}
	return rpc.RPCCall[rpc.AnsibleExecuteArgs]{
		RPCFunction: rpc.RPCAnsibleExecute,
		Args: rpc.AnsibleExecuteArgs{
			Name: VdoName,
			Args: args,
		},
	}, nil
}

// Return values for the `vdo` Ansible module.
type VdoReturn struct {
	AnsibleCommonReturns
}

// Unwrap the `rpc.RPCResult` into an `VdoReturn`
func VdoReturnFromRPCResult(r rpc.RPCResult[rpc.AnsibleExecuteResult]) (VdoReturn, error) {
	return cast.AnyToJSONT[VdoReturn](r.Result.Result)
}
