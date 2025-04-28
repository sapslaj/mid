// Copyright 2016-2023, Pulumi Corporation.
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

package provider

import (
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi-go-provider/middleware/schema"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"

	"github.com/sapslaj/mid/provider/resource"
	"github.com/sapslaj/mid/provider/types"
)

// Version is initialized by the Go linker to contain the semver of this build.
var Version string

const Name string = "mid"

func Provider() p.Provider {
	// We tell the provider what resources it needs to support.
	// In this case, a single resource and component
	return infer.Provider(infer.Options{
		Metadata: schema.Metadata{
			DisplayName:       "mid",
			Description:       "Pulumi-native configuration management",
			Homepage:          "https://github.com/sapslaj/mid",
			Repository:        "https://github.com/sapslaj/mid",
			License:           "MIT",
			PluginDownloadURL: "github://api.github.com/sapslaj/mid",
			LanguageMap: map[string]any{
				"go": map[string]any{
					"respectSchemaVersion":           true,
					"generateResourceContainerTypes": true,
					"importBasePath":                 "github.com/sapslaj/mid/sdk/go/mid",
				},
				"nodejs": map[string]any{
					"respectSchemaVersion": true,
					"packageName":          "@sapslaj/pulumi-mid",
				},
			},
		},
		Resources: []infer.InferredResource{
			infer.Resource[resource.Apt, resource.AptArgs, resource.AptState](),
			infer.Resource[resource.Exec, resource.ExecArgs, resource.ExecState](),
			infer.Resource[resource.File, resource.FileArgs, resource.FileState](),
			infer.Resource[resource.FileLine, resource.FileLineArgs, resource.FileLineState](),
			infer.Resource[resource.Group, resource.GroupArgs, resource.GroupState](),
			infer.Resource[resource.Package, resource.PackageArgs, resource.PackageState](),
			infer.Resource[resource.Service, resource.ServiceArgs, resource.ServiceState](),
			infer.Resource[resource.SystemdService, resource.SystemdServiceArgs, resource.SystemdServiceState](),
			infer.Resource[resource.User, resource.UserArgs, resource.UserState](),
		},
		Config: infer.Config[types.Config](),
		ModuleMap: map[tokens.ModuleName]tokens.ModuleName{
			"provider": "index",
		},
	})
}
