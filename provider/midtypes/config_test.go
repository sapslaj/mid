package midtypes_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sapslaj/mid/pkg/providerfw/infer"
	"github.com/sapslaj/mid/pkg/ptr"
	"github.com/sapslaj/mid/provider/midtypes"
)

func TestGetResourceConfig(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		providerConfig *midtypes.ProviderConfig
		resourceConfig *midtypes.ResourceConfig
		expect         midtypes.ResourceConfig
	}{
		"config fully from provider with nil resource config": {
			providerConfig: &midtypes.ProviderConfig{
				ResourceConfig: midtypes.ResourceConfig{
					DeleteUnreachable: ptr.Of(true),
					Parallel:          ptr.Of(2),
				},
			},
			resourceConfig: nil,
			expect: midtypes.ResourceConfig{
				DeleteUnreachable: ptr.Of(true),
				Parallel:          ptr.Of(2),
			},
		},

		"config fully from provider with empty resource config": {
			providerConfig: &midtypes.ProviderConfig{
				ResourceConfig: midtypes.ResourceConfig{
					DeleteUnreachable: ptr.Of(true),
					Parallel:          ptr.Of(2),
				},
			},
			resourceConfig: &midtypes.ResourceConfig{},
			expect: midtypes.ResourceConfig{
				DeleteUnreachable: ptr.Of(true),
				Parallel:          ptr.Of(2),
			},
		},

		"config fully from resource": {
			providerConfig: &midtypes.ProviderConfig{},
			resourceConfig: &midtypes.ResourceConfig{
				DeleteUnreachable: ptr.Of(true),
				Parallel:          ptr.Of(2),
			},
			expect: midtypes.ResourceConfig{
				DeleteUnreachable: ptr.Of(true),
				Parallel:          ptr.Of(2),
			},
		},

		"partial from provider config with nil resource config": {
			providerConfig: &midtypes.ProviderConfig{
				ResourceConfig: midtypes.ResourceConfig{
					Parallel: ptr.Of(2),
				},
			},
			resourceConfig: nil,
			expect: midtypes.ResourceConfig{
				DeleteUnreachable: nil,
				Parallel:          ptr.Of(2),
			},
		},

		"partial from provider config with empty resource config": {
			providerConfig: &midtypes.ProviderConfig{
				ResourceConfig: midtypes.ResourceConfig{
					Parallel: ptr.Of(2),
				},
			},
			resourceConfig: &midtypes.ResourceConfig{},
			expect: midtypes.ResourceConfig{
				DeleteUnreachable: nil,
				Parallel:          ptr.Of(2),
			},
		},

		"partial from both provider and resource config": {
			providerConfig: &midtypes.ProviderConfig{
				ResourceConfig: midtypes.ResourceConfig{
					Parallel: ptr.Of(2),
				},
			},
			resourceConfig: &midtypes.ResourceConfig{
				DeleteUnreachable: ptr.Of(true),
			},
			expect: midtypes.ResourceConfig{
				DeleteUnreachable: ptr.Of(true),
				Parallel:          ptr.Of(2),
			},
		},

		"resource config overrides provider config": {
			providerConfig: &midtypes.ProviderConfig{
				ResourceConfig: midtypes.ResourceConfig{
					DeleteUnreachable: ptr.Of(false),
					Parallel:          ptr.Of(2),
				},
			},
			resourceConfig: &midtypes.ResourceConfig{
				DeleteUnreachable: ptr.Of(true),
			},
			expect: midtypes.ResourceConfig{
				DeleteUnreachable: ptr.Of(true),
				Parallel:          ptr.Of(2),
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			inferredConfig := infer.Config(tc.providerConfig)
			ctx := context.WithValue(context.Background(), infer.ConfigKey, inferredConfig)

			got := midtypes.GetResourceConfig(ctx, tc.resourceConfig)

			assert.Equal(t, tc.expect, got)
		})
	}
}
