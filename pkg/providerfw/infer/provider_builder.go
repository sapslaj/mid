// Copyright 2016-2025, Pulumi Corporation.
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
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	provider "github.com/sapslaj/mid/pkg/providerfw"
	"github.com/sapslaj/mid/pkg/providerfw/middleware/schema"
)

type ProviderBuilder struct {
	metadata   schema.Metadata
	resources  []InferredResource
	components []InferredComponent
	functions  []InferredFunction
	config     InferredConfig
	moduleMap  map[tokens.ModuleName]tokens.ModuleName
	wrapped    provider.Provider
}

// NewProviderBuilder creates an inferred provider which fills as many defaults as possible.
//
// A base set of defaults are provided to create a minimal provider configuration. Further
// customization can be done by chaining method calls on the returned [ProviderBuilder] object.
//
// This is an example of how to create a simple provider with a single component resource:
//
//	type RandomComponent struct {
//	    pulumi.ResourceState
//	    RandomComponentArgs
//
//	    Password pulumi.StringOutput `pulumi:"password"`
//	}
//
//	type RandomComponentArgs struct {
//	    Length pulumi.IntInput `pulumi:"length"`
//	}
//
//	func NewMyComponent(ctx *pulumi.Context, name string,
//	    compArgs RandomComponentArgs, opts ...pulumi.ResourceOption) (*RandomComponent, error) {
//	    // Define your component constructor logic here.
//	}
//
//	func main() {
//	    p, _ := infer.NewProviderBuilder().
//	        WithComponents(
//	            infer.ComponentF(NewMyComponent),
//	        ).
//	        Build()
//	    p.Run(context.Background(), "go-components", "v0.0.1")
//	}
//
// Please note that the initial defaults provided by this function may change with future releases of
// this framework. Currently, we are setting the following defaults:
//
// - LanguageMap: A map of language-specific metadata that is used to generate the SDKs for the provider.
func NewProviderBuilder() *ProviderBuilder {
	defaultMetadata := schema.Metadata{
		LanguageMap: map[string]any{
			"nodejs": map[string]any{
				"respectSchemaVersion": true,
			},
			// See [github.com/pulumi/pulumi/pkg/v3/codegen/go.GoPackageInfo].
			"go": map[string]any{
				"generateResourceContainerTypes": true,
				"respectSchemaVersion":           true,
			},
			// See [github.com/pulumi/pulumi/pkg/v3/codegen/python.PackageInfo].
			"python": map[string]any{
				"respectSchemaVersion": true,
				"pyproject": map[string]any{
					"enabled": true,
				},
			},
			"csharp": map[string]any{
				"respectSchemaVersion": true,
			},
		},
	}

	return &ProviderBuilder{
		metadata: defaultMetadata,
	}
}

// WithResources adds the given custom resources to the provider.
func (pb *ProviderBuilder) WithResources(resources ...InferredResource) *ProviderBuilder {
	pb.resources = append(pb.resources, resources...)
	return pb
}

// WithComponents adds the given components to the provider.
func (pb *ProviderBuilder) WithComponents(components ...InferredComponent) *ProviderBuilder {
	pb.components = append(pb.components, components...)
	return pb
}

// WithFunctions adds the given functions to the provider.
func (pb *ProviderBuilder) WithFunctions(functions ...InferredFunction) *ProviderBuilder {
	pb.functions = append(pb.functions, functions...)
	return pb
}

// WithConfig adds the given config to the provider.
func (pb *ProviderBuilder) WithConfig(config InferredConfig) *ProviderBuilder {
	pb.config = config
	return pb
}

// WithModuleMap adds the given module map to the provider.
func (pb *ProviderBuilder) WithModuleMap(moduleMap map[tokens.ModuleName]tokens.ModuleName) *ProviderBuilder {
	pb.moduleMap = moduleMap
	return pb
}

// WithLanguageMap sets the language map in the provider's metadata.
// The language map is a mapping of language names to language-specific metadata.
// This is used to customize how the provider is exposed in different languages.
func (pb *ProviderBuilder) WithLanguageMap(languageMap map[string]any) *ProviderBuilder {
	pb.metadata.LanguageMap = languageMap
	return pb
}

// WithDescription sets the description for the provider.
func (pb *ProviderBuilder) WithDescription(description string) *ProviderBuilder {
	pb.metadata.Description = description
	return pb
}

// WithDisplayName sets the display name for the provider.
func (pb *ProviderBuilder) WithDisplayName(displayName string) *ProviderBuilder {
	pb.metadata.DisplayName = displayName
	return pb
}

// WithWrapped wraps another provider.
func (pb *ProviderBuilder) WithWrapped(provider provider.Provider) *ProviderBuilder {
	pb.wrapped = provider
	return pb
}

// WithKeywords adds the specified keywords to the provider's metadata.
// These keywords can be used to improve discoverability of the provider.
func (pb *ProviderBuilder) WithKeywords(keywords ...string) *ProviderBuilder {
	pb.metadata.Keywords = append(pb.metadata.Keywords, keywords...)
	return pb
}

// WithHomepage sets the homepage field in the provider metadata.
func (pb *ProviderBuilder) WithHomepage(homepage string) *ProviderBuilder {
	pb.metadata.Homepage = homepage
	return pb
}

// WithRepository sets the repository for the provider.
func (pb *ProviderBuilder) WithRepository(repository string) *ProviderBuilder {
	pb.metadata.Repository = repository
	return pb
}

// WithPublisher sets the publisher name for the provider.
func (pb *ProviderBuilder) WithPublisher(publisher string) *ProviderBuilder {
	pb.metadata.Publisher = publisher
	return pb
}

// WithLogoURL sets the logo URL for the provider.
func (pb *ProviderBuilder) WithLogoURL(logoURL string) *ProviderBuilder {
	pb.metadata.LogoURL = logoURL
	return pb
}

// WithLicense sets the license for the provider.
func (pb *ProviderBuilder) WithLicense(license string) *ProviderBuilder {
	pb.metadata.License = license
	return pb
}

// WithPluginDownloadURL sets the URL from which to download the provider's plugin.
func (pb *ProviderBuilder) WithPluginDownloadURL(pluginDownloadURL string) *ProviderBuilder {
	pb.metadata.PluginDownloadURL = pluginDownloadURL
	return pb
}

// WithGoImportPath sets the base import path for the provider's generated SDK.
func (pb *ProviderBuilder) WithGoImportPath(path string) *ProviderBuilder {
	gpi := pb.getGoPackageInfo()
	gpi["importBasePath"] = path
	return pb
}

func (pb *ProviderBuilder) getGoPackageInfo() map[string]any {
	lm := pb.metadata.LanguageMap
	if lm == nil {
		lm = map[string]any{}
		pb.metadata.LanguageMap = lm
	}

	gpi, ok := lm["go"]
	if m, isMap := gpi.(map[string]any); ok && isMap {
		return m
	}
	if gpi == nil {
		m := map[string]any{}
		lm["go"] = m
		return m
	}

	// The user could have given us a GoPackageInfo struct or an arbitrary map.
	// We don't want to take on a gogen dependency here, so round-trip the data
	// via JSON to get a map out the other side.
	bytes, err := json.Marshal(gpi)
	if err != nil {
		panic("invalid go package info: " + err.Error())
	}
	var m map[string]any
	_ = json.Unmarshal(bytes, &m)

	lm["go"] = m
	return m
}

// WithNamespace sets the provider namespace.
func (pb *ProviderBuilder) WithNamespace(namespace string) *ProviderBuilder {
	pb.metadata.Namespace = namespace
	return pb
}

// BuildOptions builds an [Options] object from the provider builder configuration. This
// is useful when a user wants to have more control over the provider creation process.
func (pb *ProviderBuilder) BuildOptions() Options {
	if pb.metadata.DisplayName == "" {
		pb.metadata.DisplayName = "yourdisplayname"
	}
	if pb.metadata.Namespace == "" {
		pb.metadata.Namespace = "yournamespace"
	}

	gpi := pb.getGoPackageInfo()
	if _, ok := gpi["importBasePath"]; !ok {
		path := fmt.Sprintf(
			"github.com/%s/%s/sdk/go/%s",
			pb.metadata.Namespace,
			pb.metadata.DisplayName,
			pb.metadata.DisplayName,
		)
		gpi["importBasePath"] = path
	}

	return Options{
		Metadata:   pb.metadata,
		Resources:  pb.resources,
		Components: pb.components,
		Functions:  pb.functions,
		Config:     pb.config,
		ModuleMap:  pb.moduleMap,
		wrapped:    pb.wrapped,
	}
}

// validate checks if the provider builder configuration is valid.
func (pb *ProviderBuilder) validate() error {
	if len(pb.components) == 0 && len(pb.resources) == 0 && len(pb.functions) == 0 {
		return fmt.Errorf("at least one resource, component, or function is required")
	}

	return nil
}

// Build builds the provider options and validates them., and runs the provider.
func (pb *ProviderBuilder) Build() (provider.Provider, error) {
	if err := pb.validate(); err != nil {
		return provider.Provider{}, err
	}

	opts := pb.BuildOptions()

	return Provider(opts), nil
}
