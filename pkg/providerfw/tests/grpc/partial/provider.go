// Copyright 2024, Pulumi Corporation.
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

package partial

import (
	"context"

	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/contract"

	p "github.com/sapslaj/mid/pkg/providerfw"
	"github.com/sapslaj/mid/pkg/providerfw/infer"
)

func Provider() p.Provider {
	return infer.Provider(infer.Options{
		Resources: []infer.InferredResource{infer.Resource(&Partial{})},
		ModuleMap: map[tokens.ModuleName]tokens.ModuleName{
			"partial": "index",
		},
	})
}

var (
	_ infer.CustomResource[Args, State] = (*Partial)(nil)
	_ infer.CustomUpdate[Args, State]   = (*Partial)(nil)
	_ infer.CustomRead[Args, State]     = (*Partial)(nil)
)

type Partial struct{}
type Args struct {
	S string `pulumi:"s"`
}
type State struct {
	Args

	Out string `pulumi:"out"`
}

func (*Partial) Create(ctx context.Context, req infer.CreateRequest[Args]) (infer.CreateResponse[State], error) {
	if req.DryRun {
		return infer.CreateResponse[State]{}, nil
	}
	contract.Assertf(req.Inputs.S == "for-create", `expected input.S to be "for-create"`)
	return infer.CreateResponse[State]{
			ID: "id",
			Output: State{
				Args: Args{S: "+for-create"},
				Out:  "partial-create",
			},
		}, infer.ResourceInitFailedError{
			Reasons: []string{"create: failed to fully init"},
		}
}

func (*Partial) Update(ctx context.Context, req infer.UpdateRequest[Args, State]) (infer.UpdateResponse[State], error) {
	if req.DryRun {
		return infer.UpdateResponse[State]{}, nil
	}
	contract.Assertf(req.Inputs.S == "for-update", `expected news.S to be "for-update"`)
	contract.Assertf(req.State.S == "+for-create", `expected olds.Out to be "partial-create"`)
	contract.Assertf(req.State.Out == "partial-init", `expected olds.Out to be "partial-create"`)

	return infer.UpdateResponse[State]{
			Output: State{
				Args: Args{
					S: "from-update",
				},
				Out: "partial-update",
			},
		}, infer.ResourceInitFailedError{
			Reasons: []string{"update: failed to continue init"},
		}
}

func (*Partial) Read(ctx context.Context, req infer.ReadRequest[Args, State]) (resp infer.ReadResponse[Args, State], err error) {
	contract.Assertf(req.Inputs.S == "for-read", `expected inputs.S to be "for-read"`)
	contract.Assertf(req.State.S == "from-update", `expected olds.Out to be "partial-create"`)
	contract.Assertf(req.State.Out == "state-for-read", `expected state.Out to be "state-for-read"`)

	return infer.ReadResponse[Args, State]{
			ID: "from-read-id",
			Inputs: Args{
				S: "from-read-input",
			},
			State: State{
				Args: Args{"s-state-from-read"},
				Out:  "out-state-from-read",
			},
		}, infer.ResourceInitFailedError{
			Reasons: []string{"read: failed to finish read"},
		}
}
