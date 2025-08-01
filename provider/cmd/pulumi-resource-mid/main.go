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

package main

import (
	"context"

	p "github.com/sapslaj/mid/pkg/providerfw"

	"github.com/sapslaj/mid/pkg/telemetry"
	mid "github.com/sapslaj/mid/provider"
	"github.com/sapslaj/mid/provider/executor"
	"github.com/sapslaj/mid/version"
)

// Serve the provider against Pulumi's Provider protocol.
func main() {
	ctx := context.Background()
	ts := telemetry.StartTelemetry(ctx)
	defer ts.Shutdown()

	defer executor.DisconnectAll(context.Background())

	provider, err := mid.Provider()
	if err != nil {
		panic(err)
	}

	p.RunProvider(ctx, mid.Name, version.Version, provider)
}
