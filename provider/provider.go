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
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"

	"github.com/sapslaj/mid/provider/agent"
	"github.com/sapslaj/mid/provider/resource"
	"github.com/sapslaj/mid/provider/types"
)

const Name string = "mid"

func Provider() (p.Provider, error) {
	return infer.NewProviderBuilder().
		WithDisplayName("mid").
		WithDescription("Pulumi-native configuration management").
		WithHomepage("https://github.com/sapslaj/mid").
		WithRepository("https://github.com/sapslaj/mid").
		WithLicense("MIT").
		WithPluginDownloadURL("github://api.github.com/sapslaj/mid").
		WithLanguageMap(map[string]any{
			"go": map[string]any{
				"respectSchemaVersion":           true,
				"generateResourceContainerTypes": true,
				"importBasePath":                 "github.com/sapslaj/mid/sdk/go/mid",
			},
			"nodejs": map[string]any{
				"respectSchemaVersion": true,
				"packageName":          "@sapslaj/pulumi-mid",
			},
			"python": map[string]any{
				"packageName":          "pulumi_mid",
				"respectSchemaVersion": true,
				"pyproject": map[string]any{
					"enabled": true,
				},
			},
		}).
		WithModuleMap(map[tokens.ModuleName]tokens.ModuleName{
			"provider": "index",
		}).
		WithConfig(infer.Config(&types.Config{})).
		WithResources(
			infer.Resource(&resource.AnsibleTaskList{}),
			infer.Resource(&resource.Apt{}),
			infer.Resource(&resource.Exec{}),
			infer.Resource(&resource.File{}),
			infer.Resource(&resource.FileLine{}),
			infer.Resource(&resource.Group{}),
			infer.Resource(&resource.Package{}),
			infer.Resource(&resource.Service{}),
			infer.Resource(&resource.SystemdService{}),
			infer.Resource(&resource.User{}),
		).
		WithFunctions(
			infer.Function(&agent.AgentPing{}),
			infer.Function(&agent.AnsibleExecute{}),
			infer.Function(&agent.Exec{}),
			infer.Function(&agent.FileStat{}),
		).
		Build()
}
