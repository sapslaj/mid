// Code generated by ./hack/generate-ansible-types.py DO NOT EDIT
package ansible

import (
	"github.com/sapslaj/mid/agent/rpc"
	"github.com/sapslaj/mid/pkg/cast"
)

// Adds or removes `rpm --import` a gpg key to your rpm database.
const RpmKeyName = "rpm_key"

// If the key will be imported or removed from the rpm db.
type RpmKeyState string

const (
	RpmKeyStateAbsent  RpmKeyState = "absent"
	RpmKeyStatePresent RpmKeyState = "present"
)

// Convert a supported type to an optional (pointer) RpmKeyState
func OptionalRpmKeyState[T interface {
	*RpmKeyState | RpmKeyState | *string | string
}](s T) *RpmKeyState {
	switch v := any(s).(type) {
	case *RpmKeyState:
		return v
	case RpmKeyState:
		return &v
	case *string:
		if v == nil {
			return nil
		}
		val := RpmKeyState(*v)
		return &val
	case string:
		val := RpmKeyState(v)
		return &val
	default:
		panic("unsupported type")
	}
}

// Parameters for the `rpm_key` Ansible module.
type RpmKeyParameters struct {
	// Key that will be modified. Can be a url, a file on the managed node, or a
	// keyid if the key already exists in the database.
	Key string `json:"key"`

	// If the key will be imported or removed from the rpm db.
	// default: RpmKeyStatePresent
	State *RpmKeyState `json:"state,omitempty"`

	// If `false` and the `key` is a url starting with `https`, SSL certificates
	// will not be validated.
	// This should only be used on personally controlled sites using self-signed
	// certificates.
	// default: "yes"
	ValidateCerts *bool `json:"validate_certs,omitempty"`

	// The long-form fingerprint of the key being imported.
	// This will be used to verify the specified key.
	Fingerprint *[]string `json:"fingerprint,omitempty"`
}

// Wrap the `RpmKeyParameters into an `rpc.RPCCall`.
func (p RpmKeyParameters) ToRPCCall() (rpc.RPCCall[rpc.AnsibleExecuteArgs], error) {
	args, err := cast.AnyToJSONT[map[string]any](p)
	if err != nil {
		return rpc.RPCCall[rpc.AnsibleExecuteArgs]{}, err
	}
	return rpc.RPCCall[rpc.AnsibleExecuteArgs]{
		RPCFunction: rpc.RPCAnsibleExecute,
		Args: rpc.AnsibleExecuteArgs{
			Name: RpmKeyName,
			Args: args,
		},
	}, nil
}

// Return values for the `rpm_key` Ansible module.
type RpmKeyReturn struct {
	AnsibleCommonReturns
}

// Unwrap the `rpc.RPCResult` into an `RpmKeyReturn`
func RpmKeyReturnFromRPCResult(r rpc.RPCResult[rpc.AnsibleExecuteResult]) (RpmKeyReturn, error) {
	return cast.AnyToJSONT[RpmKeyReturn](r.Result.Result)
}
