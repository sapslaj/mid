package types

import (
	"github.com/pulumi/pulumi-go-provider/infer"

	"github.com/sapslaj/mid/pkg/env"
)

// provider configuration
type Config struct {
	// remote endpoint connection configuration
	Connection *Connection `pulumi:"connection" provider:"secret"`

	// The value passed into the provider config for `deleteUnreachable`. It is
	// generally a better idea to use `GetDeleteUnreachable()` instead of looking
	// at this property directly.
	DeleteUnreachable bool `pulumi:"deleteUnreachable,optional"`
}

func (i *Config) Annotate(a infer.Annotator) {
	a.Describe(&i, "provider configuration")
	a.Describe(&i.Connection, "remote endpoint connection configuration")
	a.Describe(
		&i.DeleteUnreachable,
		`If present and set to true, the provider will delete resources associated
with an unreachable remote endpoint from Pulumi state. It can also be
sourced from the following environment variable:`+
			"`PULUMI_MID_DELETE_UNREACHABLE`",
	)
}

// GetDeleteUnreachable determines if the environment should delete unreachable
// resources or not.
func (config Config) GetDeleteUnreachable() bool {
	if config.DeleteUnreachable {
		return config.DeleteUnreachable
	}
	// XXX: probably don't want to panic here?
	return env.MustGetDefault("PULUMI_MID_DELETE_UNREACHABLE", false)
}
