package midtypes

import (
	"context"

	"github.com/sapslaj/mid/pkg/env"
	"github.com/sapslaj/mid/pkg/providerfw/infer"
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

	// DryRunCheck enables extra checks and dry-runs during preview. This is
	// enabled by default. Disabling it will speed up preview at the cost of
	// potentially running into unexpected errors during apply.
	DryRunCheck *bool `pulumi:"check,optional"`
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

func (config ResourceConfig) GetDryRunCheck() bool {
	if config.DryRunCheck != nil {
		return *config.DryRunCheck
	}
	return env.MustGetDefault("PULUMI_MID_DRY_RUN_CHECK", true)
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
	if providerConfig.DryRunCheck != nil {
		result.DryRunCheck = providerConfig.DryRunCheck
	}
	if config != nil {
		if config.DeleteUnreachable != nil {
			result.DeleteUnreachable = config.DeleteUnreachable
		}
		if config.Parallel != nil {
			result.Parallel = config.Parallel
		}
		if config.DryRunCheck != nil {
			result.DryRunCheck = config.DryRunCheck
		}
	}
	return result
}
