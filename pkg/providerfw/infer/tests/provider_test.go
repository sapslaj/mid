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

// Package tests contains integration tests of [infer].
package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/blang/semver"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/require"

	p "github.com/sapslaj/mid/pkg/providerfw"
	"github.com/sapslaj/mid/pkg/providerfw/infer"
	"github.com/sapslaj/mid/pkg/providerfw/integration"
)

func urn(typ, name string) resource.URN {
	return resource.NewURN("stack", "proj", "",
		tokens.Type("test:index:"+typ), name)
}

func childUrn(typ, name, parent string) resource.URN {
	return resource.NewURN("stack", "proj", tokens.Type("test:index:"+parent),
		tokens.Type("test:index:"+typ), name)
}

// Increment helps us test the highly suspicious behavior of naming an input the same as
// an output, while giving them different values. This should never be done in practice,
// but we need to accommodate the behavior while we allow it.
type (
	Increment     struct{}
	IncrementArgs struct {
		Number int `pulumi:"int"`
		Other  int `pulumi:"other,optional"`
	}
)

type IncrementOutput struct{ IncrementArgs }

func (*Increment) Create(_ context.Context,
	req infer.CreateRequest[IncrementArgs],
) (infer.CreateResponse[IncrementOutput], error) {
	output := IncrementOutput{IncrementArgs: IncrementArgs{Number: req.Inputs.Number + 1}}
	return infer.CreateResponse[IncrementOutput]{
		ID:     fmt.Sprintf("id-%d", req.Inputs.Number),
		Output: output,
	}, nil
}

type (
	Echo       struct{}
	EchoInputs struct {
		String string            `pulumi:"string"`
		Int    int               `pulumi:"int"`
		Map    map[string]string `pulumi:"strMap,optional"`
	}
)

type EchoOutputs struct {
	EchoInputs
	Name      string            `pulumi:"nameOut"`
	StringOut string            `pulumi:"stringOut"`
	IntOut    int               `pulumi:"intOut"`
	MapOut    map[string]string `pulumi:"strMapOut,optional"`
}

func (*Echo) Create(ctx context.Context,
	req infer.CreateRequest[EchoInputs],
) (infer.CreateResponse[EchoOutputs], error) {
	id := req.Name + "-id"
	state := EchoOutputs{EchoInputs: req.Inputs}

	if req.DryRun {
		return infer.CreateResponse[EchoOutputs]{
			ID:     id,
			Output: state,
		}, nil
	}

	state.Name = req.Name
	state.StringOut = req.Inputs.String
	state.IntOut = req.Inputs.Int
	state.MapOut = req.Inputs.Map

	return infer.CreateResponse[EchoOutputs]{
		ID:     id,
		Output: state,
	}, nil
}

func (*Echo) Update(ctx context.Context,
	req infer.UpdateRequest[EchoInputs, EchoOutputs],
) (infer.UpdateResponse[EchoOutputs], error) {
	if req.DryRun {
		return infer.UpdateResponse[EchoOutputs]{
			Output: req.State,
		}, nil
	}

	return infer.UpdateResponse[EchoOutputs]{
		Output: EchoOutputs{
			EchoInputs: req.Inputs,
			Name:       req.State.Name,
			StringOut:  req.Inputs.String,
			IntOut:     req.Inputs.Int,
			MapOut:     req.Inputs.Map,
		},
	}, nil
}

var _ = (infer.ExplicitDependencies[WiredInputs, WiredOutputs])((*Wired)(nil))

type (
	Wired       struct{}
	WiredInputs struct {
		String string `pulumi:"string"`
		Int    int    `pulumi:"int"`
	}
)

type WiredOutputs struct {
	Name         string `pulumi:"name"`
	StringAndInt string `pulumi:"stringAndInt"`
	StringPlus   string `pulumi:"stringPlus"`
}

func (*Wired) Create(ctx context.Context,
	req infer.CreateRequest[WiredInputs],
) (infer.CreateResponse[WiredOutputs], error) {
	id := req.Name + "-id"
	state := WiredOutputs{Name: "(" + req.Name + ")"}

	if req.DryRun {
		return infer.CreateResponse[WiredOutputs]{
			ID:     id,
			Output: state,
		}, nil
	}

	state.StringPlus = req.Inputs.String + "+"
	state.StringAndInt = fmt.Sprintf("%s-%d", req.Inputs.String, req.Inputs.Int)

	return infer.CreateResponse[WiredOutputs]{
		ID:     id,
		Output: state,
	}, nil
}

func (*Wired) Update(
	ctx context.Context, req infer.UpdateRequest[WiredInputs, WiredOutputs],
) (infer.UpdateResponse[WiredOutputs], error) {
	return infer.UpdateResponse[WiredOutputs]{
		Output: WiredOutputs{
			Name:         req.ID,
			StringAndInt: fmt.Sprintf("%s-%d", req.Inputs.String, req.Inputs.Int),
			StringPlus:   req.Inputs.String + "++",
		},
	}, nil
}

func (*Wired) WireDependencies(f infer.FieldSelector, a *WiredInputs, s *WiredOutputs) {
	stringIn := f.InputField(&a.String)
	intIn := f.InputField(&a.Int)

	name := f.OutputField(&s.Name)
	stringAndInt := f.OutputField(&s.StringAndInt)
	stringOut := f.OutputField(&s.StringPlus)

	name.AlwaysKnown()            // This is based on the pulumi name, which is always known
	stringOut.DependsOn(stringIn) // Passthrough value with a mutation
	stringAndInt.DependsOn(stringIn)
	stringAndInt.DependsOn(intIn)
}

var _ = (infer.ExplicitDependencies[WiredInputs, WiredOutputs])((*Wired)(nil))

// WiredPlus plus is like wired, but has its inputs embedded with its outputs.
//
// This allows it to remember old inputs when calculating which fields have changed.
type (
	WiredPlus        struct{}
	WiredPlusOutputs struct {
		WiredInputs
		WiredOutputs
	}
)

func (*WiredPlus) Create(
	ctx context.Context, req infer.CreateRequest[WiredInputs],
) (infer.CreateResponse[WiredPlusOutputs], error) {
	r := new(Wired)
	resp, err := r.Create(ctx, req)
	if err != nil {
		return infer.CreateResponse[WiredPlusOutputs]{}, err
	}
	return infer.CreateResponse[WiredPlusOutputs]{
		ID: resp.ID,
		Output: WiredPlusOutputs{
			WiredInputs:  req.Inputs,
			WiredOutputs: resp.Output,
		},
	}, nil
}

func (*WiredPlus) Update(
	ctx context.Context, req infer.UpdateRequest[WiredInputs, WiredPlusOutputs],
) (infer.UpdateResponse[WiredPlusOutputs], error) {
	r := new(Wired)
	updateReq := infer.UpdateRequest[WiredInputs, WiredOutputs]{
		ID:     req.ID,
		State:  req.State.WiredOutputs,
		Inputs: req.Inputs,
		DryRun: req.DryRun,
	}
	resp, err := r.Update(ctx, updateReq)
	if err != nil {
		return infer.UpdateResponse[WiredPlusOutputs]{}, err
	}
	return infer.UpdateResponse[WiredPlusOutputs]{
		Output: WiredPlusOutputs{
			WiredInputs:  req.Inputs,
			WiredOutputs: resp.Output,
		},
	}, nil
}

func (*WiredPlus) WireDependencies(f infer.FieldSelector, a *WiredInputs, s *WiredPlusOutputs) {
	r := new(Wired)
	r.WireDependencies(f, a, &s.WiredOutputs)
}

// Default values are applied by the provider to facilitate integration testing and to
// backstop non-compliment SDKs.

// TODO[pulumi-go-provider#98] Remove the ,optional.

type (
	WithDefaults       struct{}
	WithDefaultsOutput struct{ WithDefaultsArgs }
)

var (
	_ infer.Annotated = (*WithDefaultsArgs)(nil)
	_ infer.Annotated = (*NestedDefaults)(nil)
)

type WithDefaultsArgs struct {
	// We sanity check with some primitive values, but most of this checking is in
	// NestedDefaults.
	String       string                     `pulumi:"s,optional"`
	IntPtr       *int                       `pulumi:"pi,optional"`
	Nested       *NestedDefaults            `pulumi:"nested,optional"`
	NestedPtr    *NestedDefaults            `pulumi:"nestedPtr"`
	OptWithReq   *OptWithReq                `pulumi:"optWithReq,optional"`
	ArrNested    []NestedDefaults           `pulumi:"arrNested,optional"`
	ArrNestedPtr []*NestedDefaults          `pulumi:"arrNestedPtr,optional"`
	MapNested    map[string]NestedDefaults  `pulumi:"mapNested,optional"`
	MapNestedPtr map[string]*NestedDefaults `pulumi:"mapNestedPtr,optional"`

	NoDefaultsPtr *NoDefaults `pulumi:"noDefaults,optional"`
}

type OptWithReq struct {
	Required *string `pulumi:"req"`
	Optional *string `pulumi:"opt,optional"`
	Empty    *string `pulumi:"empty,optional"`
}

func (o *OptWithReq) Annotate(a infer.Annotator) {
	a.SetDefault(&o.Optional, "default-value")
}

// NoDefaults is a struct that doesn't have an associated default value.
type NoDefaults struct {
	String string `pulumi:"s,optional"`
}

func (w *WithDefaultsArgs) Annotate(a infer.Annotator) {
	a.SetDefault(&w.String, "one")
	a.SetDefault(&w.IntPtr, 2)
}

type NestedDefaults struct {
	// Direct vars. These don't allow setting zero values.
	String string  `pulumi:"s,optional"`
	Float  float64 `pulumi:"f,optional"`
	Int    int     `pulumi:"i,optional"`
	Bool   bool    `pulumi:"b,optional"`

	// Indirect vars. These should allow setting zero values.
	StringPtr *string  `pulumi:"ps,optional"`
	FloatPtr  *float64 `pulumi:"pf,optional"`
	IntPtr    *int     `pulumi:"pi,optional"`
	BoolPtr   *bool    `pulumi:"pb,optional"`

	// A triple indirect value, included to check that we can handle arbitrary
	// indirection.
	IntPtrPtrPtr ***int `pulumi:"pppi,optional"`
}

func (w *NestedDefaults) Annotate(a infer.Annotator) {
	a.SetDefault(&w.String, "two")
	a.SetDefault(&w.Float, 4.0)
	a.SetDefault(&w.Int, 8)
	// It doesn't make much sense to have default values of bools, but we support it.
	a.SetDefault(&w.Bool, true)

	// Now indirect ptrs
	a.SetDefault(&w.StringPtr, "two")
	a.SetDefault(&w.FloatPtr, 4.0)
	a.SetDefault(&w.IntPtr, 8)
	a.SetDefault(&w.BoolPtr, true)

	a.SetDefault(&w.IntPtrPtrPtr, 64)
}

func (w *WithDefaults) Create(
	ctx context.Context, req infer.CreateRequest[WithDefaultsArgs],
) (infer.CreateResponse[WithDefaultsOutput], error) {
	return infer.CreateResponse[WithDefaultsOutput]{
		ID:     "validated",
		Output: WithDefaultsOutput{WithDefaultsArgs: req.Inputs},
	}, nil
}

// ReadEnv has fields with default values filled by environmental variables.
type (
	ReadEnv     struct{}
	ReadEnvArgs struct {
		String  string  `pulumi:"s,optional"`
		Int     int     `pulumi:"i,optional"`
		Float64 float64 `pulumi:"f64,optional"`
		Bool    bool    `pulumi:"b,optional"`
	}
)
type ReadEnvOutput struct{ ReadEnvArgs }

func (w *ReadEnvArgs) Annotate(a infer.Annotator) {
	a.SetDefault(&w.String, nil, "STRING")
	a.SetDefault(&w.Int, nil, "INT")
	a.SetDefault(&w.Float64, nil, "FLOAT64")
	a.SetDefault(&w.Bool, nil, "BOOL")
}

func (w *ReadEnv) Create(
	ctx context.Context, req infer.CreateRequest[ReadEnvArgs],
) (infer.CreateResponse[ReadEnvOutput], error) {
	return infer.CreateResponse[ReadEnvOutput]{
		ID:     "well-read",
		Output: ReadEnvOutput{req.Inputs},
	}, nil
}

type (
	Recursive     struct{}
	RecursiveArgs struct {
		Value string         `pulumi:"value,optional"`
		Other *RecursiveArgs `pulumi:"other,optional"`
	}
)
type RecursiveOutput struct{ RecursiveArgs }

func (w *Recursive) Create(
	ctx context.Context, req infer.CreateRequest[RecursiveArgs],
) (infer.CreateResponse[RecursiveOutput], error) {
	return infer.CreateResponse[RecursiveOutput]{
		ID:     "did-not-overflow-stack",
		Output: RecursiveOutput{req.Inputs},
	}, nil
}

func (w *RecursiveArgs) Annotate(a infer.Annotator) {
	a.SetDefault(&w.Value, "default-value")
}

type Config struct {
	Value *string `pulumi:"value,optional"`
}

func (c *Config) Annotate(a infer.Annotator) {
	a.Describe(&c, "The provider configuration.")
	a.Describe(&c.Value, "A value that is set in the provider config.")
	a.Deprecate(&c.Value, "A deprecation message.")
}

type (
	ReadConfig       struct{}
	ReadConfigArgs   struct{}
	ReadConfigOutput struct {
		Config string `pulumi:"config"`
	}
)

func (w *ReadConfig) Create(
	ctx context.Context, req infer.CreateRequest[ReadConfigArgs],
) (infer.CreateResponse[ReadConfigOutput], error) {
	c := infer.GetConfig[Config](ctx)
	bytes, err := json.Marshal(c)
	return infer.CreateResponse[ReadConfigOutput]{
		ID:     "read",
		Output: ReadConfigOutput{Config: string(bytes)},
	}, err
}

type (
	GetJoin  struct{}
	JoinArgs struct {
		Elems []string `pulumi:"elems"`
		Sep   *string  `pulumi:"sep,optional"`
	}
)

func (j *JoinArgs) Annotate(a infer.Annotator) {
	a.SetDefault(&j.Sep, ",")
}

type JoinResult struct {
	Result string `pulumi:"result"`
}

func (*GetJoin) Invoke(
	ctx context.Context,
	req infer.FunctionRequest[JoinArgs],
) (infer.FunctionResponse[JoinResult], error) {
	return infer.FunctionResponse[JoinResult]{
		Output: JoinResult{strings.Join(req.Input.Elems, *req.Input.Sep)},
	}, nil
}

type ConfigCustom struct {
	Number  *float64 `pulumi:"number,optional"`
	Squared float64
}

func (c *ConfigCustom) Configure(ctx context.Context) error {
	if c.Number == nil {
		return nil
	}
	// We can perform arbitrary data transformations in the Configure step.  These
	// transformations aren't visible in Pulumi State, but are viable in other methods
	// on the provider.
	square := func(n float64) float64 { return n * n }
	c.Squared = square(*c.Number)
	return nil
}

var _ = (infer.CustomCheck[*ConfigCustom])((*ConfigCustom)(nil))

func (*ConfigCustom) Check(ctx context.Context,
	req infer.CheckRequest,
) (infer.CheckResponse[*ConfigCustom], error) {
	var c ConfigCustom
	if v, ok := req.NewInputs.GetOk("number"); ok {
		number := v.AsNumber() + 0.5
		c.Number = &number
	}

	return infer.CheckResponse[*ConfigCustom]{Inputs: &c}, nil
}

type (
	ReadConfigCustom       struct{}
	ReadConfigCustomArgs   struct{}
	ReadConfigCustomOutput struct {
		Config string `pulumi:"config"`
	}
)

func (w *ReadConfigCustom) Create(
	ctx context.Context, req infer.CreateRequest[ReadConfigCustomArgs],
) (infer.CreateResponse[ReadConfigCustomOutput], error) {
	c := infer.GetConfig[ConfigCustom](ctx)
	bytes, err := json.Marshal(c)
	return infer.CreateResponse[ReadConfigCustomOutput]{
		ID:     "read",
		Output: ReadConfigCustomOutput{Config: string(bytes)},
	}, err
}

type ReadConfigComponentArgs struct{}

type ReadConfigComponent struct {
	pulumi.ResourceState
	ReadConfigComponentArgs
	Config pulumi.StringOutput `pulumi:"config"`
}

func NewReadConfigComponent(ctx *pulumi.Context, name string, args ReadConfigComponentArgs,
	opts ...pulumi.ResourceOption,
) (*ReadConfigComponent, error) {
	comp := &ReadConfigComponent{}
	err := ctx.RegisterComponentResource(p.GetTypeToken(ctx), name, comp, opts...)
	if err != nil {
		return nil, err
	}
	c := infer.GetConfig[Config](ctx.Context())
	bytes, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	comp.Config = pulumi.String(string(bytes)).ToStringOutput()

	return comp, nil
}

type RandomComponentArgs struct {
	Prefix pulumi.StringInput `pulumi:"prefix"`
}

type RandomComponent struct {
	pulumi.ResourceState
	RandomComponentArgs
	Result pulumi.StringOutput `pulumi:"result"`
}

func NewRandomComponent(ctx *pulumi.Context, name string, args RandomComponentArgs,
	opts ...pulumi.ResourceOption,
) (*RandomComponent, error) {
	comp := &RandomComponent{}
	err := ctx.RegisterComponentResource(p.GetTypeToken(ctx), name, comp, opts...)
	if err != nil {
		return nil, err
	}

	if args.Prefix == nil {
		args.Prefix = pulumi.String("default-")
	}

	comp.Result = args.Prefix.ToStringOutput().ApplyT(func(prefix string) string {
		return prefix + "12345"
	}).(pulumi.StringOutput)

	return comp, nil
}

var (
	_ infer.CustomResource[CustomCheckNoDefaultsArgs, CustomCheckNoDefaultsOutput] = &CustomCheckNoDefaults{}
	_ infer.CustomCheck[CustomCheckNoDefaultsArgs]                                 = &CustomCheckNoDefaults{}
)

type (
	CustomCheckNoDefaults     struct{}
	CustomCheckNoDefaultsArgs struct {
		Input string `pulumi:"input" provider:"secret"`
	}
	CustomCheckNoDefaultsOutput struct{ CustomCheckNoDefaultsArgs }
)

func (w *CustomCheckNoDefaults) Check(_ context.Context,
	req infer.CheckRequest,
) (infer.CheckResponse[CustomCheckNoDefaultsArgs], error) {
	input := req.NewInputs.Get("input").AsString()
	return infer.CheckResponse[CustomCheckNoDefaultsArgs]{
		Inputs: CustomCheckNoDefaultsArgs{Input: input},
	}, nil
}

func (w *CustomCheckNoDefaults) Create(
	ctx context.Context, req infer.CreateRequest[CustomCheckNoDefaultsArgs],
) (infer.CreateResponse[CustomCheckNoDefaultsOutput], error) {
	return infer.CreateResponse[CustomCheckNoDefaultsOutput]{
		ID:     "id",
		Output: CustomCheckNoDefaultsOutput{req.Inputs},
	}, nil
}

func providerOpts(config infer.InferredConfig) infer.Options {
	return infer.Options{
		Config: config,
		Resources: []infer.InferredResource{
			infer.Resource(&Echo{}),
			infer.Resource(&Wired{}),
			infer.Resource(&WiredPlus{}),
			infer.Resource(&Increment{}),
			infer.Resource(&WithDefaults{}),
			infer.Resource(&ReadEnv{}),
			infer.Resource(&Recursive{}),
			infer.Resource(&ReadConfig{}),
			infer.Resource(&ReadConfigCustom{}),
			infer.Resource(&CustomCheckNoDefaults{}),
		},
		Components: []infer.InferredComponent{
			infer.ComponentF(NewRandomComponent),
			infer.ComponentF(NewReadConfigComponent),
		},
		Functions: []infer.InferredFunction{
			infer.Function(&GetJoin{}),
		},
		ModuleMap: map[tokens.ModuleName]tokens.ModuleName{"tests": "index"},
	}
}

func provider(t testing.TB) integration.Server {
	p := infer.Provider(providerOpts(nil))
	s, err := integration.NewServer(t.Context(),
		"test",
		semver.MustParse("1.0.0"),
		integration.WithProvider(p),
	)
	require.NoError(t, err)

	return s
}

func providerWithConfig[T any](t testing.TB, cfg T) integration.Server {
	p := infer.Provider(providerOpts(infer.Config(cfg)))
	s, err := integration.NewServer(t.Context(), "test", semver.MustParse("1.0.0"), integration.WithProvider(p))
	require.NoError(t, err)
	return s
}

func providerWithMocks[T any](t testing.TB, cfg T, mocks pulumi.MockResourceMonitor) integration.Server {
	p := infer.Provider(providerOpts(infer.Config(cfg)))
	s, err := integration.NewServer(
		t.Context(),
		"test",
		semver.MustParse("1.0.0"),
		integration.WithProvider(p),
		integration.WithMocks(mocks),
	)
	require.NoError(t, err)
	return s
}
