// Code generated by ./hack/generate-ansible-types.py DO NOT EDIT
package ansible

import (
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/cast"
)

// This module allows configuring block device partition using the `parted`
// command line tool. For a full description of the fields and the options check
// the GNU parted manual.
const PartedName = "parted"

// Set alignment for newly created partitions. Use `undefined` for parted
// default alignment.
type PartedAlign string

const (
	PartedAlignCylinder  PartedAlign = "cylinder"
	PartedAlignMinimal   PartedAlign = "minimal"
	PartedAlignNone      PartedAlign = "none"
	PartedAlignOptimal   PartedAlign = "optimal"
	PartedAlignUndefined PartedAlign = "undefined"
)

// Convert a supported type to an optional (pointer) PartedAlign
func OptionalPartedAlign[T interface {
	*PartedAlign | PartedAlign | *string | string
}](s T) *PartedAlign {
	switch v := any(s).(type) {
	case *PartedAlign:
		return v
	case PartedAlign:
		return &v
	case *string:
		if v == nil {
			return nil
		}
		val := PartedAlign(*v)
		return &val
	case string:
		val := PartedAlign(v)
		return &val
	default:
		panic("unsupported type")
	}
}

// Selects the current default unit that Parted will use to display locations
// and capacities on the disk and to interpret those given by the user if they
// are not suffixed by an unit.
// When fetching information about a disk, it is recommended to always specify a
// unit.
type PartedUnit string

const (
	PartedUnitS       PartedUnit = "s"
	PartedUnitB       PartedUnit = "B"
	PartedUnitKB      PartedUnit = "KB"
	PartedUnitKiB     PartedUnit = "KiB"
	PartedUnitMB      PartedUnit = "MB"
	PartedUnitMiB     PartedUnit = "MiB"
	PartedUnitGB      PartedUnit = "GB"
	PartedUnitGiB     PartedUnit = "GiB"
	PartedUnitTB      PartedUnit = "TB"
	PartedUnitTiB     PartedUnit = "TiB"
	PartedUnitPercent PartedUnit = "%"
	PartedUnitCyl     PartedUnit = "cyl"
	PartedUnitChs     PartedUnit = "chs"
	PartedUnitCompact PartedUnit = "compact"
)

// Convert a supported type to an optional (pointer) PartedUnit
func OptionalPartedUnit[T interface {
	*PartedUnit | PartedUnit | *string | string
}](s T) *PartedUnit {
	switch v := any(s).(type) {
	case *PartedUnit:
		return v
	case PartedUnit:
		return &v
	case *string:
		if v == nil {
			return nil
		}
		val := PartedUnit(*v)
		return &val
	case string:
		val := PartedUnit(v)
		return &val
	default:
		panic("unsupported type")
	}
}

// Disk label type or partition table to use.
// If `device` already contains a different label, it will be changed to `label`
// and any previous partitions will be lost.
// A `name` must be specified for a `gpt` partition table.
type PartedLabel string

const (
	PartedLabelAix   PartedLabel = "aix"
	PartedLabelAmiga PartedLabel = "amiga"
	PartedLabelBsd   PartedLabel = "bsd"
	PartedLabelDvh   PartedLabel = "dvh"
	PartedLabelGpt   PartedLabel = "gpt"
	PartedLabelLoop  PartedLabel = "loop"
	PartedLabelMac   PartedLabel = "mac"
	PartedLabelMsdos PartedLabel = "msdos"
	PartedLabelPc98  PartedLabel = "pc98"
	PartedLabelSun   PartedLabel = "sun"
)

// Convert a supported type to an optional (pointer) PartedLabel
func OptionalPartedLabel[T interface {
	*PartedLabel | PartedLabel | *string | string
}](s T) *PartedLabel {
	switch v := any(s).(type) {
	case *PartedLabel:
		return v
	case PartedLabel:
		return &v
	case *string:
		if v == nil {
			return nil
		}
		val := PartedLabel(*v)
		return &val
	case string:
		val := PartedLabel(v)
		return &val
	default:
		panic("unsupported type")
	}
}

// May be specified only with `label=msdos` or `label=dvh`.
// Neither `part_type` nor `name` may be used with `label=sun`.
type PartedPartType string

const (
	PartedPartTypeExtended PartedPartType = "extended"
	PartedPartTypeLogical  PartedPartType = "logical"
	PartedPartTypePrimary  PartedPartType = "primary"
)

// Convert a supported type to an optional (pointer) PartedPartType
func OptionalPartedPartType[T interface {
	*PartedPartType | PartedPartType | *string | string
}](s T) *PartedPartType {
	switch v := any(s).(type) {
	case *PartedPartType:
		return v
	case PartedPartType:
		return &v
	case *string:
		if v == nil {
			return nil
		}
		val := PartedPartType(*v)
		return &val
	case string:
		val := PartedPartType(v)
		return &val
	default:
		panic("unsupported type")
	}
}

// Whether to create or delete a partition.
// If set to `info` the module will only return the device information.
type PartedState string

const (
	PartedStateAbsent  PartedState = "absent"
	PartedStatePresent PartedState = "present"
	PartedStateInfo    PartedState = "info"
)

// Convert a supported type to an optional (pointer) PartedState
func OptionalPartedState[T interface {
	*PartedState | PartedState | *string | string
}](s T) *PartedState {
	switch v := any(s).(type) {
	case *PartedState:
		return v
	case PartedState:
		return &v
	case *string:
		if v == nil {
			return nil
		}
		val := PartedState(*v)
		return &val
	case string:
		val := PartedState(v)
		return &val
	default:
		panic("unsupported type")
	}
}

// Parameters for the `parted` Ansible module.
type PartedParameters struct {
	// The block device (disk) where to operate.
	// Regular files can also be partitioned, but it is recommended to create a
	// loopback device using `losetup` to easily access its partitions.
	Device string `json:"device"`

	// Set alignment for newly created partitions. Use `undefined` for parted
	// default alignment.
	// default: PartedAlignOptimal
	Align *PartedAlign `json:"align,omitempty"`

	// The partition number being affected.
	// Required when performing any action on the disk, except fetching
	// information.
	Number *int `json:"number,omitempty"`

	// Selects the current default unit that Parted will use to display locations
	// and capacities on the disk and to interpret those given by the user if they
	// are not suffixed by an unit.
	// When fetching information about a disk, it is recommended to always specify
	// a unit.
	// default: PartedUnitKib
	Unit *PartedUnit `json:"unit,omitempty"`

	// Disk label type or partition table to use.
	// If `device` already contains a different label, it will be changed to
	// `label` and any previous partitions will be lost.
	// A `name` must be specified for a `gpt` partition table.
	// default: PartedLabelMsdos
	Label *PartedLabel `json:"label,omitempty"`

	// May be specified only with `label=msdos` or `label=dvh`.
	// Neither `part_type` nor `name` may be used with `label=sun`.
	// default: PartedPartTypePrimary
	PartType *PartedPartType `json:"part_type,omitempty"`

	// Where the partition will start as offset from the beginning of the disk,
	// that is, the "distance" from the start of the disk. Negative numbers specify
	// distance from the end of the disk.
	// The distance can be specified with all the units supported by parted (except
	// compat) and it is case sensitive, for example `10GiB`, `15%`.
	// Using negative values may require setting of `fs_type` (see notes).
	// default: "0%"
	PartStart *string `json:"part_start,omitempty"`

	// Where the partition will end as offset from the beginning of the disk, that
	// is, the "distance" from the start of the disk. Negative numbers specify
	// distance from the end of the disk.
	// The distance can be specified with all the units supported by parted (except
	// compat) and it is case sensitive, for example `10GiB`, `15%`.
	// default: "100%"
	PartEnd *string `json:"part_end,omitempty"`

	// Sets the name for the partition number (GPT, Mac, MIPS and PC98 only).
	Name *string `json:"name,omitempty"`

	// A list of the flags that has to be set on the partition.
	Flags *[]string `json:"flags,omitempty"`

	// Whether to create or delete a partition.
	// If set to `info` the module will only return the device information.
	// default: PartedStateInfo
	State *PartedState `json:"state,omitempty"`

	// If specified and the partition does not exist, will set filesystem type to
	// given partition.
	// Parameter optional, but see notes below about negative `part_start` values.
	FsType *string `json:"fs_type,omitempty"`

	// Call `resizepart` on existing partitions to match the size specified by
	// `part_end`.
	// default: false
	Resize *bool `json:"resize,omitempty"`
}

// Wrap the `PartedParameters into an `rpc.RPCCall`.
func (p PartedParameters) ToRPCCall() (rpc.RPCCall[rpc.AnsibleExecuteArgs], error) {
	args, err := cast.AnyToJSONT[map[string]any](p)
	if err != nil {
		return rpc.RPCCall[rpc.AnsibleExecuteArgs]{}, err
	}
	return rpc.RPCCall[rpc.AnsibleExecuteArgs]{
		RPCFunction: rpc.RPCAnsibleExecute,
		Args: rpc.AnsibleExecuteArgs{
			Name: PartedName,
			Args: args,
		},
	}, nil
}

// Return values for the `parted` Ansible module.
type PartedReturn struct {
	AnsibleCommonReturns

	// Current partition information.
	PartitionInfo *any `json:"partition_info,omitempty"`
}

// Unwrap the `rpc.RPCResult` into an `PartedReturn`
func PartedReturnFromRPCResult(r rpc.RPCResult[rpc.AnsibleExecuteResult]) (PartedReturn, error) {
	return cast.AnyToJSONT[PartedReturn](r.Result.Result)
}
