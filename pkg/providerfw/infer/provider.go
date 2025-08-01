// Copyright 2022, Pulumi Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package infer

import (
	"context"
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/contract"
	p "github.com/sapslaj/mid/pkg/providerfw"
	t "github.com/sapslaj/mid/pkg/providerfw/middleware"
	"github.com/sapslaj/mid/pkg/providerfw/middleware/cancel"
	"github.com/sapslaj/mid/pkg/providerfw/middleware/complexconfig" //nolint:staticcheck
	mContext "github.com/sapslaj/mid/pkg/providerfw/middleware/context"
	"github.com/sapslaj/mid/pkg/providerfw/middleware/dispatch"
	"github.com/sapslaj/mid/pkg/providerfw/middleware/schema"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const InferStateKeyName = "pulumi-go-provider-infer"

type ConfigKeyType struct{}

var ConfigKey ConfigKeyType

// Options to configure an inferred provider.
//
// See [Provider] to turn a set of Options into a [p.Provider].
type Options struct {
	// Metadata describes provider level metadata for the schema.
	//
	// Look at [schema.Metadata] to see the set of configurable options.
	//
	// It does not contain runtime details for the provider.
	schema.Metadata

	// The set of custom resources served by the provider.
	//
	// To create an [InferredResource], use [Resource].
	Resources []InferredResource

	// The set of component resources served by the provider.
	//
	// To create an [InferredComponent], use [Component].
	Components []InferredComponent

	// The set of functions served by the provider.
	//
	// To create an [InferredFunction], use [Function].
	Functions []InferredFunction

	// The config used by the provider, if any.
	//
	// To create an [InferredConfig], use [Config].
	Config InferredConfig

	// ModuleMap provides a mapping between go modules and pulumi modules.
	//
	// For example, given a provider `pkg` with defines resources `foo.Foo`, `foo.Bar`, and
	// `fizz.Buzz` the provider will expose resources at `pkg:foo:Foo`, `pkg:foo:Bar` and
	// `pkg:fizz:Buzz`. Adding
	//
	//	`opts.ModuleMap = map[tokens.ModuleName]tokens.ModuleName{"foo": "bar"}`
	//
	// will instead result in exposing the same resources at `pkg:bar:Foo`, `pkg:bar:Bar` and
	// `pkg:fizz:Buzz`.
	ModuleMap map[tokens.ModuleName]tokens.ModuleName

	// wrapped is an optional provider which this new provider wraps.
	wrapped p.Provider
}

func (o Options) dispatch() dispatch.Options {
	functions := map[tokens.Type]t.Invoke{}
	for _, r := range o.Functions {
		typ, err := r.GetToken()
		contract.AssertNoErrorf(err, "failed to get token for function %v", r)
		functions[typ] = r
	}
	customs := map[tokens.Type]t.CustomResource{}
	for _, r := range o.Resources {
		typ, err := r.GetToken()
		contract.AssertNoErrorf(err, "failed to get token for resource %v", r)
		customs[typ] = r
	}
	components := map[tokens.Type]t.ComponentResource{}
	for _, r := range o.Components {
		typ, err := r.GetToken()
		contract.AssertNoErrorf(err, "failed to get token for component %v", r)
		components[typ] = r
	}
	return dispatch.Options{
		Customs:    customs,
		Components: components,
		Invokes:    functions,
		ModuleMap:  o.ModuleMap,
	}
}

func (o Options) schema() schema.Options {
	resources := make([]schema.Resource, len(o.Resources)+len(o.Components))
	for i, r := range o.Resources {
		resources[i] = r
	}
	for i, c := range o.Components {
		resources[i+len(o.Resources)] = c
	}
	functions := make([]schema.Function, len(o.Functions))
	for i, f := range o.Functions {
		functions[i] = f
	}

	return schema.Options{
		Resources: resources,
		Invokes:   functions,
		Provider:  o.Config,
		Metadata:  o.Metadata,
		ModuleMap: o.ModuleMap,
	}
}

// Provider creates a new inferred provider from `opts`.
//
// To customize the resulting provider, including setting resources, functions, config options and other
// schema metadata, look at the [Options] struct.
func Provider(opts Options) p.Provider {
	return Wrap(opts.wrapped, opts)
}

// Wrap wraps a compatible underlying provider in an inferred provider (as described by options).
//
// The resulting provider will respond to resources and functions that are described in `opts`, delegating
// unknown calls to the underlying provider.
func Wrap(provider p.Provider, opts Options) p.Provider {
	provider = dispatch.Wrap(provider, opts.dispatch())
	provider = schema.Wrap(provider, opts.schema())

	config := opts.Config
	if config != nil {
		if prev := provider.Configure; prev != nil {
			provider.Configure = func(ctx context.Context, req p.ConfigureRequest) error {
				err := config.configure(ctx, req)
				if err != nil {
					return err
				}
				err = prev(ctx, req)
				if status.Code(err) == codes.Unimplemented {
					return nil
				}
				return err
			}
		} else {
			provider.Configure = config.configure
		}
		provider.DiffConfig = config.diffConfig
		provider.CheckConfig = config.checkConfig
		provider = mContext.Wrap(provider, func(ctx context.Context) context.Context {
			return context.WithValue(ctx, ConfigKey, opts.Config)
		})
	}

	provider = complexconfig.Wrap(provider)
	return cancel.Wrap(provider)
}

// GetConfig retrieves the configuration of this provider.
//
// Note: GetConfig will panic if the type of T does not match the type of the config or if
// the provider has not supplied a config.
func GetConfig[T any](ctx context.Context) T {
	v := ctx.Value(ConfigKey)
	var t T
	if v == nil {
		panic(fmt.Sprintf("Config[%T] called on a provider without a config", t))
	}
	c := v.(InferredConfig)
	if c, ok := c.(*config[T]); ok {
		return *c.receiver
	}
	if c, ok := c.(*config[*T]); ok {
		return **c.receiver
	}
	panic(fmt.Sprintf("Config[%T] called but the correct config type is %s", t, c.underlyingType()))
}
