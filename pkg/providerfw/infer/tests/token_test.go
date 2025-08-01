// Copyright 2023, Pulumi Corporation.
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

package tests

import (
	"context"
	"testing"

	"github.com/blang/semver"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	p "github.com/sapslaj/mid/pkg/providerfw"
	"github.com/sapslaj/mid/pkg/providerfw/infer"
	"github.com/sapslaj/mid/pkg/providerfw/integration"
)

type CustomToken struct{}

func (c *CustomToken) Annotate(a infer.Annotator) { a.SetToken("overwritten", "Tk") }

func (*CustomToken) Create(
	context.Context, infer.CreateRequest[TokenArgs],
) (infer.CreateResponse[TokenResult], error) {
	panic("unimplemented")
}

type TokenArgs struct {
	Array []ObjectToken `pulumi:"arr"`

	Single ObjectToken `pulumi:"single"`
}
type TokenResult struct {
	Map map[string]ObjectToken `pulumi:"m"`
}

type TokenComponent struct{ pulumi.ResourceState }

// Check that we allow other capitalization schemes
func (c *TokenComponent) Annotate(a infer.Annotator) { a.SetToken("cmp", "tK") }

func Construct(
	ctx *pulumi.Context, name string, inputs TokenArgs, opts ...pulumi.ResourceOption,
) (*TokenComponent, error) {
	panic("unimplemented")
}

type FnToken struct{}

func (c *FnToken) Annotate(a infer.Annotator) { a.SetToken("fn", "TK") }

func (*FnToken) Invoke(
	ctx context.Context,
	_ infer.FunctionRequest[TokenArgs],
) (output infer.FunctionResponse[TokenResult], err error) {
	panic("unimplemented")
}

type ObjectToken struct {
	Value string `pulumi:"value"`
}

func (c *ObjectToken) Annotate(a infer.Annotator) { a.SetToken("obj", "Customized") }

func TestTokens(t *testing.T) {
	t.Parallel()

	provider := infer.Provider(infer.Options{
		Resources: []infer.InferredResource{
			infer.Resource(&CustomToken{}),
		},
		Components: []infer.InferredComponent{
			infer.ComponentF(Construct),
		},
		Functions: []infer.InferredFunction{
			infer.Function(&FnToken{}),
		},
		ModuleMap: map[tokens.ModuleName]tokens.ModuleName{"overwritten": "index"},
	})
	server, err := integration.NewServer(t.Context(),
		"test",
		semver.MustParse("1.0.0"), integration.WithProvider(provider),
	)
	require.NoError(t, err)

	schema, err := server.GetSchema(p.GetSchemaRequest{})
	require.NoError(t, err)

	assert.JSONEq(t, `{
  "name": "test",
  "version": "1.0.0",
  "config": {},
  "types": {
    "test:obj:Customized": {
      "properties": {
        "value": {
          "type": "string"
        }
      },
      "type": "object",
      "required": [
        "value"
      ]
    }
  },
  "provider": {},
  "resources": {
    "test:cmp:tK": {
      "inputProperties": {
        "arr": {
          "type": "array",
          "items": {
            "$ref": "#/types/test:obj:Customized"
          }
        },
        "single": {
          "$ref": "#/types/test:obj:Customized"
        }
      },
      "requiredInputs": [
        "arr",
        "single"
      ],
      "isComponent": true
    },
    "test:index:Tk": {
      "properties": {
        "m": {
          "type": "object",
          "additionalProperties": {
            "$ref": "#/types/test:obj:Customized"
          }
        }
      },
      "required": [
        "m"
      ],
      "inputProperties": {
        "arr": {
          "type": "array",
          "items": {
            "$ref": "#/types/test:obj:Customized"
          }
        },
        "single": {
          "$ref": "#/types/test:obj:Customized"
        }
      },
      "requiredInputs": [
        "arr",
        "single"
      ]
    }
  },
  "functions": {
    "test:fn:TK": {
      "inputs": {
        "properties": {
          "arr": {
            "type": "array",
            "items": {
              "$ref": "#/types/test:obj:Customized"
            }
          },
          "single": {
            "$ref": "#/types/test:obj:Customized"
          }
        },
        "type": "object",
        "required": [
          "arr",
          "single"
        ]
      },
      "outputs": {
        "properties": {
          "m": {
            "type": "object",
            "additionalProperties": {
              "$ref": "#/types/test:obj:Customized"
            }
          }
        },
        "type": "object",
        "required": [
          "m"
        ]
      }
    }
  }
}`, schema.Schema)
}

type (
	MyResource       struct{}
	MyResourceArgs   struct{}
	MyResourceOutput struct{}
)

func (MyResource) Create(
	ctx context.Context,
	_ infer.CreateRequest[MyResourceArgs],
) (infer.CreateResponse[MyResourceOutput], error) {
	return infer.CreateResponse[MyResourceOutput]{}, nil
}

type MyComponentArgs struct{}

type MyComponentOutput struct {
	pulumi.ResourceState
}

func (MyComponentArgs) Create(
	ctx context.Context,
	_ infer.CreateRequest[MyComponentArgs],
) (infer.CreateResponse[MyComponentOutput], error) {
	return infer.CreateResponse[MyComponentOutput]{}, nil
}

type MyComponent struct{}

func (MyComponent) Construct(
	ctx *pulumi.Context,
	name string,
	typ string,
	args MyComponentArgs,
	opts pulumi.ResourceOption,
) (*MyComponentOutput, error) {
	return &MyComponentOutput{}, nil
}

func TestGetToken(t *testing.T) {
	t.Parallel()

	t.Run("component", func(t *testing.T) {
		t.Parallel()

		component := infer.Component(&MyComponent{})
		tok, err := component.GetToken()
		require.NoError(t, err)
		assert.Equal(t, tokens.TypeName("MyComponent"), tok.Name())
	})

	t.Run("resource", func(t *testing.T) {
		t.Parallel()

		resource := infer.Resource(&MyResource{})
		tok, err := resource.GetToken()
		require.NoError(t, err)
		assert.Equal(t, tokens.TypeName("MyResource"), tok.Name())
	})
}
