package midtypes

import (
	"context"

	"github.com/sapslaj/mid/pkg/providerfw/infer"
	"github.com/sapslaj/mid/pkg/env"
)

type ResourceConfig struct {
	// The value passed into the provider config for `deleteUnreachable`. It is
	// generally a better idea to use `GetDeleteUnreachable()` instead of looking
	// at this property directly.
	DeleteUnreachable *bool `pulumi:"deleteUnreachable,optional"`

	// Parallel sets the maximum number of parallel tasks to execute against the
	// remote system. If not set or set to `0`, it will use the remote host's
	// number of CPU cores. If set to `-1` it will be unlimited.
	Parallel *int `pulumi:"parallel,optional"`
}

// GetDeleteUnreachable determines if the environment should delete unreachable
// resources or not.
func (config ResourceConfig) GetDeleteUnreachable() bool {
	if config.DeleteUnreachable != nil {
		return *config.DeleteUnreachable
	}
	// XXX: probably don't want to panic here?
	return env.MustGetDefault("PULUMI_MID_DELETE_UNREACHABLE", false)
}

func (config ResourceConfig) GetParallel() int {
	if config.Parallel != nil {
		return *config.Parallel
	}
	return env.MustGetDefault("PULUMI_MID_PARALLEL", 0)
}

// provider configuration
type ProviderConfig struct {
	ResourceConfig
	// remote endpoint connection configuration
	Connection *Connection `pulumi:"connection,optional" provider:"secret"`
}

func GetResourceConfig(ctx context.Context, config *ResourceConfig) ResourceConfig {
	result := ResourceConfig{}
	providerConfig := infer.GetConfig[ProviderConfig](ctx)
	if providerConfig.DeleteUnreachable != nil {
		result.DeleteUnreachable = providerConfig.DeleteUnreachable
	}
	if providerConfig.Parallel != nil {
		result.Parallel = providerConfig.Parallel
	}
	if config != nil {
		if config.DeleteUnreachable != nil {
			result.DeleteUnreachable = config.DeleteUnreachable
		}
		if config.Parallel != nil {
			result.Parallel = config.Parallel
		}
	}
	return result
}
