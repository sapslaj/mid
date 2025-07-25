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
	"reflect"
	"unicode"

	pschema "github.com/pulumi/pulumi/pkg/v3/codegen/schema"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"

	p "github.com/sapslaj/mid/pkg/providerfw"
	"github.com/sapslaj/mid/pkg/providerfw/ende"
	"github.com/sapslaj/mid/pkg/providerfw/introspect"
	t "github.com/sapslaj/mid/pkg/providerfw/middleware"
	"github.com/sapslaj/mid/pkg/providerfw/middleware/schema"
)

// FunctionRequest wraps the input type for a function call
type FunctionRequest[I any] struct {
	// Input contains the function input arguments.
	Input I
}

// FunctionResponse wraps the output type for a function call
type FunctionResponse[O any] struct {
	// Output contains the function result.
	Output O
}

// Fn is a function (also called fnvoke) inferred from code. `I` is the function input,
// and `O` is the function output. Both must be structs.
type Fn[I any, O any] interface {
	// Fn is a function (also called an "invoke" or "Provider Function") inferred from code. `I` is the function input,
	// and `O` is the function output. Both must be structs.
	// See: https://www.pulumi.com/docs/iac/concepts/resources/functions/#provider-functions
	Invoke(ctx context.Context, req FunctionRequest[I]) (resp FunctionResponse[O], err error)
}

// InferredFunction is a function inferred from code. See [Function] for creating a
// InferredFunction.
type InferredFunction interface {
	t.Invoke
	schema.Function

	isInferredFunction()
}

// Function infers a function from `F`, which maps `I` to `O`.
func Function[F Fn[I, O], I, O any](fnc F) InferredFunction {
	return &derivedInvokeController[F, I, O]{receiver: fnc}
}

type derivedInvokeController[F Fn[I, O], I, O any] struct {
	receiver F
}

func (derivedInvokeController[F, I, O]) isInferredFunction() {}

func (rc *derivedInvokeController[F, I, O]) GetToken() (tokens.Type, error) {
	// By default, we get resource style tokens:
	//
	//	pkg:index:FizzBuzz
	//
	// Functions use a different capitalization convention, so we need to convert:
	//
	//	pkg:index:fizzBuzz
	//

	// If the receiver implements Annotate, run it to see if we have a custom
	// token set. This doesn't recurse, but that's OK because we only care
	// about the token.
	if r, ok := any(rc.receiver).(Annotated); ok {
		a := introspect.NewAnnotator(r)
		r.Annotate(&a)
		if a.Token != "" {
			return tokens.Type(a.Token), nil
		}
	}
	return getToken[F](fnToken)
}

func fnToken(tk tokens.Type) tokens.Type {
	name := []rune(tk.Name().String())
	for i, r := range name {
		if !unicode.IsUpper(r) {
			break
		}
		if i == 0 || len(name) == i+1 || unicode.IsUpper(name[i+1]) {
			name[i] = unicode.ToLower(r)
		}
	}
	return tokens.NewTypeToken(tk.Module(), tokens.TypeName(name))
}

func (r *derivedInvokeController[F, I, O]) GetSchema(reg schema.RegisterDerivativeType) (pschema.FunctionSpec, error) {
	descriptions := getAnnotated(reflect.TypeOf(new(F)))

	input, err := objectSchema(reflect.TypeOf(new(I)))
	if err != nil {
		return pschema.FunctionSpec{}, err
	}
	output, err := objectSchema(reflect.TypeOf(new(O)))
	if err != nil {
		return pschema.FunctionSpec{}, err
	}

	if err := registerTypes[I](reg); err != nil {
		return pschema.FunctionSpec{}, err
	}
	if err := registerTypes[O](reg); err != nil {
		return pschema.FunctionSpec{}, err
	}

	return pschema.FunctionSpec{
		Description: descriptions.Descriptions[""],
		Inputs:      input,
		Outputs:     output,
	}, nil
}

func objectSchema(t reflect.Type) (*pschema.ObjectTypeSpec, error) {
	descriptions := getAnnotated(t)
	props, required, err := propertyListFromType(t, false, inputType)
	if err != nil {
		return nil, fmt.Errorf("could not serialize input type %s: %w", t, err)
	}
	for n, p := range props {
		props[n] = p
	}
	return &pschema.ObjectTypeSpec{
		Description: descriptions.Descriptions[""],
		Properties:  props,
		Required:    required,
		Type:        "object",
	}, nil
}

func (r *derivedInvokeController[F, I, O]) Invoke(ctx context.Context, req p.InvokeRequest) (p.InvokeResponse, error) {
	encoder, i, mapErr := ende.Decode[I](req.Args)
	mapFailures, err := checkFailureFromMapError(mapErr)
	if err != nil {
		return p.InvokeResponse{}, err
	}
	if len(mapFailures) > 0 {
		return p.InvokeResponse{
			Failures: mapFailures,
		}, nil
	}

	err = applyDefaults(&i)
	if err != nil {
		return p.InvokeResponse{}, fmt.Errorf("unable to apply defaults: %w", err)
	}

	o, err := r.receiver.Invoke(ctx, FunctionRequest[I]{Input: i})
	if err != nil {
		return p.InvokeResponse{}, err
	}
	m, err := encoder.Encode(o.Output)
	if err != nil {
		return p.InvokeResponse{}, err
	}
	return p.InvokeResponse{
		Return: applySecrets[O](m),
	}, nil
}
