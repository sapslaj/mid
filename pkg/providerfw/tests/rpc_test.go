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

package tests

// This file contains tests for the middleware that projects a [rpc.ResourceProviderServer] into a [p.Provider].
//
// It is intended that Provider is used to wrap legacy native provider implementations
// while they are gradually transferred over to pulumi-go-provider based implementations.

import (
	"context"
	"fmt"
	"testing"

	"github.com/blang/semver"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource/plugin"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	rpc "github.com/pulumi/pulumi/sdk/v3/proto/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"

	p "github.com/sapslaj/mid/pkg/providerfw"
	"github.com/sapslaj/mid/pkg/providerfw/integration"
	wraprpc "github.com/sapslaj/mid/pkg/providerfw/middleware/rpc"
)

func configureResult(
	ret *rpc.ConfigureResponse,
) func(context.Context, *rpc.ConfigureRequest) (*rpc.ConfigureResponse, error) {
	return func(context.Context, *rpc.ConfigureRequest) (*rpc.ConfigureResponse, error) {
		return ret, nil
	}
}

func TestRPCGetSchema(t *testing.T) {
	t.Run("no-error", func(t *testing.T) {
		resp, err := rpcServer(t, rpcTestServer{
			onGetSchema: func(_ context.Context, req *rpc.GetSchemaRequest) (*rpc.GetSchemaResponse, error) {
				assert.Equal(t, int32(4), req.Version)
				return &rpc.GetSchemaResponse{
					Schema: "some-schema",
				}, nil
			},
		}).GetSchema(p.GetSchemaRequest{
			Version: 4,
		})
		require.NoError(t, err)
		assert.Equal(t, p.GetSchemaResponse{
			Schema: "some-schema",
		}, resp)
	})
	t.Run("error", func(t *testing.T) {
		_, err := rpcServer(t, rpcTestServer{
			onGetSchema: func(_ context.Context, req *rpc.GetSchemaRequest) (*rpc.GetSchemaResponse, error) {
				assert.Equal(t, int32(0), req.Version)
				return &rpc.GetSchemaResponse{}, fmt.Errorf("no schema found")
			},
		}).GetSchema(p.GetSchemaRequest{
			Version: 0,
		})
		assert.ErrorContains(t, err, "no schema found")
	})
}

func TestRPCCancel(t *testing.T) {
	t.Run("no-error", func(t *testing.T) {
		var wasCalled bool
		err := rpcServer(t, rpcTestServer{
			onCancel: func(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
				wasCalled = true
				return &emptypb.Empty{}, nil
			},
		}).Cancel()
		assert.NoError(t, err)
		assert.True(t, wasCalled)
	})
	t.Run("error", func(t *testing.T) {
		err := rpcServer(t, rpcTestServer{
			onCancel: func(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
				return &emptypb.Empty{}, fmt.Errorf("cancel failed")
			},
		}).Cancel()
		assert.ErrorContains(t, err, "cancel failed")
	})
}

func TestRPCCheckConfig(t *testing.T) {
	t.Parallel()
	testRPCCheck(t, func(
		f func(context.Context, *rpc.CheckRequest) (*rpc.CheckResponse, error),
	) func(p.CheckRequest) (p.CheckResponse, error) {
		s := rpcServer(t, rpcTestServer{onCheckConfig: f})
		return s.CheckConfig
	})
}

func TestRPCDiffConfig(t *testing.T) {
	t.Parallel()
	testRPCDiff(t, func(
		f func(context.Context, *rpc.DiffRequest) (*rpc.DiffResponse, error),
	) func(p.DiffRequest) (p.DiffResponse, error) {
		s := rpcServer(t, rpcTestServer{onDiffConfig: f})
		return s.DiffConfig
	})
}

func TestRPCConfigure(t *testing.T) {
	t.Parallel()

	t.Run("args", func(t *testing.T) {
		t.Parallel()
		args, expectedArgs := exampleOlds()
		var didRun bool
		s := rpcServer(t, rpcTestServer{
			onConfigure: func(
				_ context.Context, req *rpc.ConfigureRequest,
			) (*rpc.ConfigureResponse, error) {
				assert.Equal(t, expectedArgs, req.GetArgs().AsMap())
				didRun = true

				return &rpc.ConfigureResponse{}, nil
			},
		})
		err := s.Configure(p.ConfigureRequest{
			Args: resource.FromResourcePropertyValue(resource.NewProperty(args)).AsMap(),
		})
		require.NoError(t, err)
		assert.True(t, didRun)
	})

	t.Run("variables", func(t *testing.T) {
		t.Parallel()
		vars := map[string]string{
			"f1":     "v1",
			"f2":     "123",
			"nested": `{"foo": "bar"}`,
		}
		var didRun bool
		s := rpcServer(t, rpcTestServer{
			onConfigure: func(
				_ context.Context, req *rpc.ConfigureRequest,
			) (*rpc.ConfigureResponse, error) {
				assert.Equal(t, vars, req.GetVariables())
				didRun = true

				return &rpc.ConfigureResponse{}, nil
			},
		})
		err := s.Configure(p.ConfigureRequest{Variables: vars})
		require.NoError(t, err)
		assert.True(t, didRun)
	})

	// Check that we elide secretes when secrets are not supported.
	t.Run("secrets", func(t *testing.T) {
		t.Parallel()
		for _, acceptSecrets := range []bool{true, false} {
			t.Run(fmt.Sprintf("%v", acceptSecrets), func(t *testing.T) {
				s := rpcServer(t, rpcTestServer{
					onConfigure: configureResult(&rpc.ConfigureResponse{
						AcceptSecrets: acceptSecrets,
					}),
					onCreate: func(
						_ context.Context, req *rpc.CreateRequest,
					) (*rpc.CreateResponse, error) {
						m, err := plugin.UnmarshalProperties(
							req.GetProperties(), plugin.MarshalOptions{
								KeepSecrets: true,
							})
						require.NoError(t, err)
						if acceptSecrets {
							assert.Equal(t, resource.PropertyMap{
								"secret": resource.MakeSecret(
									resource.NewProperty("v")),
							}, m)
						} else {
							assert.Equal(t, resource.PropertyMap{
								"secret": resource.NewProperty("v"),
							}, m)
						}

						return &rpc.CreateResponse{Id: "some-id"}, nil
					},
				})

				require.NoError(t, s.Configure(p.ConfigureRequest{}))
				resp, err := s.Create(p.CreateRequest{
					Properties: property.NewMap(map[string]property.Value{
						"secret": property.New("v").WithSecret(true),
					}),
				})
				require.NoError(t, err)
				assert.Equal(t, p.CreateResponse{ID: "some-id"}, resp)
			})
		}
	})

	// Check that we elide output values when outputs are not supported.
	t.Run("outputs", func(t *testing.T) {
		t.Parallel()
		for _, acceptOutputs := range []bool{true, false} {
			t.Run(fmt.Sprintf("%v", acceptOutputs), func(t *testing.T) {
				s := rpcServer(t, rpcTestServer{
					onConfigure: configureResult(&rpc.ConfigureResponse{
						AcceptOutputs: acceptOutputs,
					}),
					onCreate: func(
						_ context.Context, req *rpc.CreateRequest,
					) (*rpc.CreateResponse, error) {
						m, err := plugin.UnmarshalProperties(
							req.GetProperties(), plugin.MarshalOptions{
								KeepUnknowns:     true,
								KeepOutputValues: true,
							})
						require.NoError(t, err)
						if acceptOutputs {
							assert.Equal(t, resource.PropertyMap{
								"output": resource.NewOutputProperty(resource.Output{
									Secret: true,
								}),
								"known": resource.NewOutputProperty(resource.Output{
									Known:   true,
									Element: resource.NewProperty("v1"),
									Dependencies: []resource.URN{
										"had-dep",
									},
								}),
								"unknown": resource.MakeComputed(
									resource.NewProperty(""),
								),
							}, m)
						} else {
							assert.Equal(t, resource.PropertyMap{
								"known": resource.NewProperty("v1"),
								"output": resource.MakeComputed(
									resource.NewProperty(""),
								),
								"unknown": resource.MakeComputed(
									resource.NewProperty(""),
								),
							}, m)
						}

						return &rpc.CreateResponse{Id: "some-id"}, nil
					},
				})

				require.NoError(t, s.Configure(p.ConfigureRequest{}))
				resp, err := s.Create(p.CreateRequest{
					Properties: property.NewMap(map[string]property.Value{
						"output": property.New(property.Computed).WithSecret(true),
						"known": property.New("v1").WithDependencies([]resource.URN{
							"had-dep",
						}),
						"unknown": property.New(property.Computed),
					}),
				})
				require.NoError(t, err)
				assert.Equal(t, p.CreateResponse{ID: "some-id"}, resp)
			})
		}
	})

	// Check that we only call preview functions when a server supports preview, but
	// that we always call non-preview functions.
	t.Run("preview", func(t *testing.T) {
		t.Parallel()
		for _, preview := range []bool{true, false} {
			t.Run(fmt.Sprintf("%v", preview), func(t *testing.T) {
				s := rpcServer(t, rpcTestServer{
					onConfigure: configureResult(&rpc.ConfigureResponse{
						SupportsPreview: preview,
					}),
					onCreate: func(
						_ context.Context, req *rpc.CreateRequest,
					) (*rpc.CreateResponse, error) {
						if !preview && req.GetPreview() {
							assert.Fail(t, "preview should not be called when no preview")
						}
						id := "some-id"
						if req.GetPreview() {
							id = "preview-id"
						}
						return &rpc.CreateResponse{Id: id}, nil
					},
				})

				require.NoError(t, s.Configure(p.ConfigureRequest{}))
				resp, err := s.Create(p.CreateRequest{
					DryRun: true,
				})
				require.NoError(t, err)
				if preview {
					assert.Equal(t, p.CreateResponse{ID: "preview-id"}, resp)
				}
				resp, err = s.Create(p.CreateRequest{})
				require.NoError(t, err)
				assert.Equal(t, p.CreateResponse{ID: "some-id"}, resp)
			})
		}
	})
}

func TestRPCInvoke(t *testing.T) {
	t.Parallel()
	t.Run("inputs", func(t *testing.T) {
		t.Parallel()

		args, expectedArgs := exampleNews()
		_, err := rpcServer(t, rpcTestServer{
			onInvoke: func(_ context.Context, req *rpc.InvokeRequest) (*rpc.InvokeResponse, error) {
				assert.Equal(t, "some-token", req.GetTok())
				assert.Equal(t, expectedArgs, req.GetArgs().AsMap())

				return nil, fmt.Errorf("success")
			},
		}).Invoke(p.InvokeRequest{
			Token: "some-token",
			Args:  resource.FromResourcePropertyValue(resource.NewProperty(args)).AsMap(),
		})
		assert.ErrorContains(t, err, "success")
	})

	t.Run("return", func(t *testing.T) {
		t.Parallel()

		args, expectedArgs := exampleNews()
		resp, err := rpcServer(t, rpcTestServer{
			onInvoke: func(context.Context, *rpc.InvokeRequest) (*rpc.InvokeResponse, error) {
				return &rpc.InvokeResponse{
					Return: must(structpb.NewStruct(expectedArgs)),
				}, nil
			},
		}).Invoke(p.InvokeRequest{})
		require.NoError(t, err)
		assert.Equal(t, resource.FromResourcePropertyValue(resource.NewProperty(args)).AsMap(), resp.Return)
	})

	t.Run("failures", func(t *testing.T) {
		t.Parallel()

		resp, err := rpcServer(t, rpcTestServer{
			onInvoke: func(context.Context, *rpc.InvokeRequest) (*rpc.InvokeResponse, error) {
				return &rpc.InvokeResponse{
					Failures: []*rpc.CheckFailure{
						{Property: "my-prop", Reason: "some-reason"},
						{Property: "my-other-prop", Reason: "some-other-reason"},
					},
				}, nil
			},
		}).Invoke(p.InvokeRequest{})
		require.NoError(t, err)
		assert.Equal(t, p.InvokeResponse{
			Failures: []p.CheckFailure{
				{Property: "my-prop", Reason: "some-reason"},
				{Property: "my-other-prop", Reason: "some-other-reason"},
			},
		}, resp)
	})
}

func TestRPCCheck(t *testing.T) {
	t.Parallel()
	testRPCCheck(t, func(
		f func(context.Context, *rpc.CheckRequest) (*rpc.CheckResponse, error),
	) func(p.CheckRequest) (p.CheckResponse, error) {
		return rpcServer(t, rpcTestServer{onCheck: f}).Check
	})
}

func TestRPCDiff(t *testing.T) {
	t.Parallel()
	testRPCDiff(t, func(
		f func(context.Context, *rpc.DiffRequest) (*rpc.DiffResponse, error),
	) func(p.DiffRequest) (p.DiffResponse, error) {
		s := rpcServer(t, rpcTestServer{onDiff: f})
		return s.Diff
	})
}

func TestRPCCreate(t *testing.T) {
	t.Parallel()

	t.Run("inputs", func(t *testing.T) {
		t.Parallel()

		args, expectedArgs := exampleNews()

		resp, err := rpcServer(t, rpcTestServer{
			onCreate: func(_ context.Context, req *rpc.CreateRequest) (*rpc.CreateResponse, error) {

				assert.Equal(t, "some-urn", req.GetUrn())
				assert.Equal(t, 123.0, req.GetTimeout())
				assert.Equal(t, true, req.GetPreview())
				assert.Equal(t, expectedArgs, req.GetProperties().AsMap())

				return &rpc.CreateResponse{Id: "some-id"}, nil
			},
		}).Create(p.CreateRequest{
			Urn:        "some-urn",
			Properties: resource.FromResourcePropertyValue(resource.NewProperty(args)).AsMap(),
			Timeout:    123,
			DryRun:     true,
		})

		require.NoError(t, err)
		assert.Equal(t, p.CreateResponse{ID: "some-id"}, resp)
	})

	t.Run("properties", func(t *testing.T) {
		t.Parallel()
		props, mapProps := exampleOlds()

		resp, err := rpcServer(t, rpcTestServer{
			onCreate: func(context.Context, *rpc.CreateRequest) (*rpc.CreateResponse, error) {
				return &rpc.CreateResponse{
					Id:         "some-id",
					Properties: must(structpb.NewStruct(mapProps)),
				}, nil
			},
		}).Create(p.CreateRequest{})

		require.NoError(t, err)
		assert.Equal(t, p.CreateResponse{
			ID:         "some-id",
			Properties: resource.FromResourcePropertyValue(resource.NewProperty(props)).AsMap(),
		}, resp)
	})
}

func TestRPCRead(t *testing.T) {
	t.Parallel()

	t.Run("inputs", func(t *testing.T) {
		t.Parallel()

		props, expectedProps := exampleOlds()
		inputs, expectedInputs := exampleNews()

		wasCalled := false

		_, err := rpcServer(t, rpcTestServer{
			onRead: func(_ context.Context, req *rpc.ReadRequest) (*rpc.ReadResponse, error) {
				assert.Equal(t, "some-id", req.GetId())
				assert.Equal(t, "some-urn", req.GetUrn())
				assert.Equal(t, expectedProps, req.GetProperties().AsMap())
				assert.Equal(t, expectedInputs, req.GetInputs().AsMap())
				wasCalled = true
				return &rpc.ReadResponse{}, nil
			},
		}).Read(p.ReadRequest{
			ID:         "some-id",
			Urn:        "some-urn",
			Properties: resource.FromResourcePropertyValue(resource.NewProperty(props)).AsMap(),
			Inputs:     resource.FromResourcePropertyValue(resource.NewProperty(inputs)).AsMap(),
		})

		require.NoError(t, err)
		assert.True(t, wasCalled)
	})

	t.Run("outputs", func(t *testing.T) {
		t.Parallel()

		props, expectedProps := exampleOlds()
		inputs, expectedInputs := exampleNews()

		resp, err := rpcServer(t, rpcTestServer{
			onRead: func(context.Context, *rpc.ReadRequest) (*rpc.ReadResponse, error) {
				return &rpc.ReadResponse{
					Id:         "some-id",
					Properties: must(structpb.NewStruct(expectedProps)),
					Inputs:     must(structpb.NewStruct(expectedInputs)),
				}, nil
			},
		}).Read(p.ReadRequest{})
		require.NoError(t, err)
		assert.Equal(t, p.ReadResponse{
			ID:         "some-id",
			Properties: resource.FromResourcePropertyValue(resource.NewProperty(props)).AsMap(),
			Inputs:     resource.FromResourcePropertyValue(resource.NewProperty(inputs)).AsMap(),
		}, resp)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		_, err := rpcServer(t, rpcTestServer{
			onRead: func(context.Context, *rpc.ReadRequest) (*rpc.ReadResponse, error) {
				return nil, fmt.Errorf("on-error")
			},
		}).Read(p.ReadRequest{})

		assert.ErrorContains(t, err, "on-error")
	})

}

func TestRPCUpdate(t *testing.T) {
	t.Parallel()

	t.Run("inputs", func(t *testing.T) {
		t.Parallel()

		olds, expectedOlds := exampleOlds()
		news, expectedNews := exampleNews()

		wasCalled := false

		_, err := rpcServer(t, rpcTestServer{
			onUpdate: func(_ context.Context, req *rpc.UpdateRequest) (*rpc.UpdateResponse, error) {
				assert.Equal(t, "some-id", req.GetId())
				assert.Equal(t, "some-urn", req.GetUrn())
				assert.Equal(t, expectedOlds, req.GetOlds().AsMap())
				assert.Equal(t, expectedNews, req.GetNews().AsMap())
				assert.Equal(t, 1.23, req.GetTimeout())
				assert.Equal(t, []string{"f1"}, req.GetIgnoreChanges())
				assert.Equal(t, true, req.GetPreview())
				wasCalled = true
				return &rpc.UpdateResponse{}, nil
			},
		}).Update(p.UpdateRequest{
			ID:            "some-id",
			Urn:           "some-urn",
			State:         resource.FromResourcePropertyValue(resource.NewProperty(olds)).AsMap(),
			Inputs:        resource.FromResourcePropertyValue(resource.NewProperty(news)).AsMap(),
			Timeout:       1.23,
			IgnoreChanges: []string{"f1"},
			DryRun:        true,
		})

		require.NoError(t, err)
		assert.True(t, wasCalled)
	})

	t.Run("outputs", func(t *testing.T) {
		t.Parallel()

		props, propsMap := exampleOlds()

		resp, err := rpcServer(t, rpcTestServer{
			onUpdate: func(context.Context, *rpc.UpdateRequest) (*rpc.UpdateResponse, error) {
				return &rpc.UpdateResponse{
					Properties: must(structpb.NewStruct(propsMap)),
				}, nil
			},
		}).Update(p.UpdateRequest{})
		require.NoError(t, err)
		assert.Equal(t, p.UpdateResponse{
			Properties: resource.FromResourcePropertyValue(resource.NewProperty(props)).AsMap(),
		}, resp)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		_, err := rpcServer(t, rpcTestServer{
			onUpdate: func(context.Context, *rpc.UpdateRequest) (*rpc.UpdateResponse, error) {
				return nil, fmt.Errorf("on-error")
			},
		}).Update(p.UpdateRequest{})

		assert.ErrorContains(t, err, "on-error")
	})
}

func TestRPCDelete(t *testing.T) {
	t.Parallel()

	t.Run("no-error", func(t *testing.T) {
		t.Parallel()
		props, expectedProps := exampleOlds()
		wasCalled := false

		err := rpcServer(t, rpcTestServer{
			onDelete: func(_ context.Context, req *rpc.DeleteRequest) (*emptypb.Empty, error) {
				assert.Equal(t, "my-id", req.GetId())
				assert.Equal(t, "my-urn", req.GetUrn())
				assert.Equal(t, expectedProps, req.GetProperties().AsMap())
				assert.Equal(t, 7.3, req.GetTimeout())
				wasCalled = true
				return &emptypb.Empty{}, nil
			},
		}).Delete(p.DeleteRequest{
			ID:         "my-id",
			Urn:        "my-urn",
			Properties: resource.FromResourcePropertyValue(resource.NewProperty(props)).AsMap(),
			Timeout:    7.3,
		})

		assert.NoError(t, err)
		assert.True(t, wasCalled)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		err := rpcServer(t, rpcTestServer{
			onDelete: func(_ context.Context, req *rpc.DeleteRequest) (*emptypb.Empty, error) {
				return &emptypb.Empty{}, fmt.Errorf("my-error")
			},
		}).Delete(p.DeleteRequest{})

		assert.ErrorContains(t, err, "my-error")
	})
}

func exampleConstructInputs(acceptOutputs bool) (property.Map, map[string]any) {
	r := property.NewMap(map[string]property.Value{
		"k1": property.New("s"),
		"k2": property.New(property.Computed).WithDependencies([]resource.URN{"urn1", "urn2"}),
		"k3": property.New(property.Null),
	})
	if acceptOutputs {
		return r, map[string]any{
			"k1": "s",
			"k2": map[string]any{"4dabf18193072939515e22adb298388d": "d0e6a833031e9bbcd3f4e8bde6ca49a4", "dependencies": []any{"urn1", "urn2"}},
			"k3": nil,
		}
	} else {
		return r, map[string]any{
			"k1": "s",
			"k2": "04da6b54-80e4-46f7-96ec-b56ff0331ba9",
			"k3": nil,
		}
	}
}

func exampleConstructInputDependencies(acceptOutputs bool) map[string]*rpc.ConstructRequest_PropertyDependencies {
	if acceptOutputs {
		return map[string]*rpc.ConstructRequest_PropertyDependencies{}
	} else {
		return map[string]*rpc.ConstructRequest_PropertyDependencies{
			"k2": {Urns: []string{"urn1", "urn2"}},
		}
	}
}

func exampleConstuctState() (map[string]any, property.Map) {
	return map[string]any{
			"r1": "s",
			"r2": "04da6b54-80e4-46f7-96ec-b56ff0331ba9",
			"r3": map[string]any{"4dabf18193072939515e22adb298388d": "d0e6a833031e9bbcd3f4e8bde6ca49a4", "dependencies": []any{"urn1", "urn2"}},
			"r4": nil,
		},
		property.NewMap(map[string]property.Value{
			"r1": property.New("s"),
			"r2": property.New(property.Computed).WithDependencies([]resource.URN{"urn1", "urn2"}),
			"r3": property.New(property.Computed).WithDependencies([]resource.URN{"urn1", "urn2"}),
			"r4": property.New(property.Null),
		})
}

func exampleConstructStateDependencies() (map[string]*rpc.ConstructResponse_PropertyDependencies, map[string][]resource.URN) {
	return map[string]*rpc.ConstructResponse_PropertyDependencies{
			"r2": {Urns: []string{"urn1", "urn2"}},
		},
		map[string][]resource.URN{
			"r2": {"urn1", "urn2"},
		}
}

func TestRPCConstruct(t *testing.T) {
	t.Parallel()

	t.Run("no-error", func(t *testing.T) {
		t.Parallel()
		inputs, expectedInputs := exampleConstructInputs(true)
		state, _ := exampleConstuctState()
		stateDeps, _ := exampleConstructStateDependencies()
		wasCalled := false

		s := rpcServer(t, rpcTestServer{
			onConfigure: configureResult(&rpc.ConfigureResponse{
				AcceptOutputs: true,
			}),
			onConstruct: func(_ context.Context, req *rpc.ConstructRequest) (*rpc.ConstructResponse, error) {
				assert.Equal(t, "test:index:Component", req.GetType())
				assert.Equal(t, "component", req.GetName())
				assert.Equal(t, "urn:pulumi:test::test::test:index:Parent::parent", req.GetParent())
				assert.Equal(t, expectedInputs, req.GetInputs().AsMap())
				assert.Equal(t, true, req.AcceptsOutputValues)

				wasCalled = true
				return &rpc.ConstructResponse{
					Urn:               "urn:pulumi:test::test::test:index:Component::component",
					State:             must(structpb.NewStruct(state)),
					StateDependencies: stateDeps,
				}, nil
			},
		})
		require.NoError(t, s.Configure(p.ConfigureRequest{}))
		_, err := s.Construct(p.ConstructRequest{
			Urn:    "urn:pulumi:test::test::test:index:Component::component",
			Parent: "urn:pulumi:test::test::test:index:Parent::parent",
			Inputs: inputs,
		})

		assert.NoError(t, err)
		assert.True(t, wasCalled)
	})

	// Check that we downgrade inputs when outputs are not supported.
	t.Run("inputs", func(t *testing.T) {
		t.Parallel()
		for _, acceptOutputs := range []bool{true, false} {
			inputs, expectedInputs := exampleConstructInputs(acceptOutputs)
			expectedInputDeps := exampleConstructInputDependencies(acceptOutputs)
			state, _ := exampleConstuctState()
			stateDeps, _ := exampleConstructStateDependencies()

			wasCalled := false
			s := rpcServer(t, rpcTestServer{
				onConfigure: configureResult(&rpc.ConfigureResponse{
					AcceptOutputs: acceptOutputs,
				}),
				onConstruct: func(_ context.Context, req *rpc.ConstructRequest) (*rpc.ConstructResponse, error) {
					assert.Equal(t, expectedInputs, req.GetInputs().AsMap())
					assert.Equal(t, expectedInputDeps, req.GetInputDependencies())
					wasCalled = true

					return &rpc.ConstructResponse{
						Urn:               "urn:pulumi:test::test::test:index:Component::component",
						State:             must(structpb.NewStruct(state)),
						StateDependencies: stateDeps,
					}, nil
				},
			})
			require.NoError(t, s.Configure(p.ConfigureRequest{}))
			_, err := s.Construct(p.ConstructRequest{
				Urn:    "urn:pulumi:test::test::test:index:Component::component",
				Parent: "urn:pulumi:test::test::test:index:Parent::parent",
				Inputs: inputs,
			})

			assert.NoError(t, err)
			assert.True(t, wasCalled)
		}
	})

	// Check that we upgrade state values when outputs are not supported.
	t.Run("state", func(t *testing.T) {
		t.Parallel()

		inputs, _ := exampleConstructInputs(true)
		state, expectedState := exampleConstuctState()
		stateDeps, expectedStateDeps := exampleConstructStateDependencies()

		s := rpcServer(t, rpcTestServer{
			onConfigure: configureResult(&rpc.ConfigureResponse{
				AcceptOutputs: true,
			}),
			onConstruct: func(_ context.Context, req *rpc.ConstructRequest) (*rpc.ConstructResponse, error) {
				return &rpc.ConstructResponse{
					Urn:               "urn:pulumi:test::test::test:index:Component::component",
					State:             must(structpb.NewStruct(state)),
					StateDependencies: stateDeps,
				}, nil
			},
		})
		require.NoError(t, s.Configure(p.ConfigureRequest{}))
		resp, err := s.Construct(p.ConstructRequest{
			Urn:    "urn:pulumi:test::test::test:index:Component::component",
			Parent: "urn:pulumi:test::test::test:index:Parent::parent",
			Inputs: inputs,
		})

		assert.NoError(t, err)
		assert.Equal(t, expectedState, resp.State, "state should be the same")
		for name, deps := range expectedStateDeps {
			v, ok := resp.State.GetOk(name)
			if !ok {
				assert.Fail(t, "state value for %q should be present", name)
				continue
			}
			assert.Equal(t, deps, v.Dependencies(), "state dependencies for %q should be the same", name)
		}
	})
}

func exampleCallArgs(acceptOutputs bool) (property.Map, map[string]any) {
	r := property.NewMap(map[string]property.Value{
		"k1": property.New("s"),
		"k2": property.New(property.Computed).WithDependencies([]resource.URN{"urn1", "urn2"}),
		"k3": property.New(property.Null),
	})
	if acceptOutputs {
		return r, map[string]any{
			"k1": "s",
			"k2": map[string]any{"4dabf18193072939515e22adb298388d": "d0e6a833031e9bbcd3f4e8bde6ca49a4", "dependencies": []any{"urn1", "urn2"}},
			"k3": nil,
		}
	} else {
		return r, map[string]any{
			"k1": "s",
			"k2": "04da6b54-80e4-46f7-96ec-b56ff0331ba9",
			"k3": nil,
		}
	}
}

func exampleCallArgDependencies(acceptOutputs bool) map[string]*rpc.CallRequest_ArgumentDependencies {
	if acceptOutputs {
		return map[string]*rpc.CallRequest_ArgumentDependencies{}
	} else {
		return map[string]*rpc.CallRequest_ArgumentDependencies{
			"k2": {Urns: []string{"urn1", "urn2"}},
		}
	}
}

func exampleCallReturns() (map[string]any, property.Map) {
	return map[string]any{
			"r1": "s",
			"r2": "04da6b54-80e4-46f7-96ec-b56ff0331ba9",
			"r3": map[string]any{"4dabf18193072939515e22adb298388d": "d0e6a833031e9bbcd3f4e8bde6ca49a4", "dependencies": []any{"urn1", "urn2"}},
			"r4": nil,
		},
		property.NewMap(map[string]property.Value{
			"r1": property.New("s"),
			"r2": property.New(property.Computed).WithDependencies([]resource.URN{"urn1", "urn2"}),
			"r3": property.New(property.Computed).WithDependencies([]resource.URN{"urn1", "urn2"}),
			"r4": property.New(property.Null),
		})
}

func exampleCallReturnDependencies() (map[string]*rpc.CallResponse_ReturnDependencies, map[string][]resource.URN) {
	return map[string]*rpc.CallResponse_ReturnDependencies{
			"r2": {Urns: []string{"urn1", "urn2"}},
		},
		map[string][]resource.URN{
			"r2": {"urn1", "urn2"},
		}
}

func TestRPCCall(t *testing.T) {
	t.Parallel()

	t.Run("no-error", func(t *testing.T) {
		t.Parallel()
		args, expectedArgs := exampleCallArgs(true)
		expectedArgDeps := exampleCallArgDependencies(true)
		returns, expectedReturns := exampleCallReturns()
		returnDeps, _ := exampleCallReturnDependencies()
		wasCalled := false

		s := rpcServer(t, rpcTestServer{
			onConfigure: configureResult(&rpc.ConfigureResponse{
				AcceptOutputs: true,
			}),
			onCall: func(_ context.Context, req *rpc.CallRequest) (*rpc.CallResponse, error) {
				assert.Equal(t, "some-token", req.GetTok(), "token should be the same")
				assert.Equal(t, expectedArgs, req.GetArgs().AsMap(), "args should be the same")
				assert.Equal(t, expectedArgDeps, req.GetArgDependencies(), "arg dependencies should be the same")
				assert.Equal(t, "some-project", req.GetProject(), "project should be the same")
				assert.Equal(t, "some-stack", req.GetStack(), "stack should be the same")
				assert.Equal(t, true, req.GetAcceptsOutputValues(), "accepts output values should be the same")

				wasCalled = true
				return &rpc.CallResponse{
					Return:             must(structpb.NewStruct(returns)),
					ReturnDependencies: returnDeps,
				}, nil
			},
		})
		require.NoError(t, s.Configure(p.ConfigureRequest{}))
		resp, err := s.Call(p.CallRequest{
			Tok:     tokens.ModuleMember("some-token"),
			Project: "some-project",
			Stack:   "some-stack",
			Args:    args,
		})

		assert.NoError(t, err)
		assert.True(t, wasCalled)
		assert.Equal(t,
			expectedReturns,
			resp.Return, "return values should be the same")
	})

	// Check that we downgrade args when outputs are not supported.
	t.Run("args", func(t *testing.T) {
		t.Parallel()
		for _, acceptOutputs := range []bool{true, false} {
			t.Run(fmt.Sprintf("%v", acceptOutputs), func(t *testing.T) {
				args, expectedArgs := exampleCallArgs(acceptOutputs)
				expectedArgDeps := exampleCallArgDependencies(acceptOutputs)

				var wasCalled bool
				s := rpcServer(t, rpcTestServer{
					onConfigure: configureResult(&rpc.ConfigureResponse{
						AcceptOutputs: acceptOutputs,
					}),
					onCall: func(_ context.Context, req *rpc.CallRequest) (*rpc.CallResponse, error) {
						wasCalled = true
						assert.Equal(t, expectedArgs, req.GetArgs().AsMap(), "args should be the same")
						assert.Equal(t, expectedArgDeps, req.GetArgDependencies(), "arg dependencies should be the same")
						return &rpc.CallResponse{}, nil
					},
				})
				require.NoError(t, s.Configure(p.ConfigureRequest{}))
				_, err := s.Call(p.CallRequest{
					Tok:     tokens.ModuleMember("some-token"),
					Project: "some-project",
					Stack:   "some-stack",
					Args:    args,
				})

				assert.NoError(t, err)
				assert.True(t, wasCalled)
			})
		}
	})

	// Check that we upgrade return values when outputs are not supported.
	t.Run("returns", func(t *testing.T) {
		t.Parallel()
		_return, expectedReturn := exampleCallReturns()
		returnDeps, expectedReturnDeps := exampleCallReturnDependencies()

		s := rpcServer(t, rpcTestServer{
			onConfigure: configureResult(&rpc.ConfigureResponse{
				AcceptOutputs: true,
			}),
			onCall: func(_ context.Context, req *rpc.CallRequest) (*rpc.CallResponse, error) {
				return &rpc.CallResponse{
					Return:             must(structpb.NewStruct(_return)),
					ReturnDependencies: returnDeps,
				}, nil
			},
		})
		require.NoError(t, s.Configure(p.ConfigureRequest{}))
		resp, err := s.Call(p.CallRequest{
			Tok:     tokens.ModuleMember("some-token"),
			Project: "some-project",
			Stack:   "some-stack",
		})

		assert.NoError(t, err)
		assert.Equal(t, expectedReturn, resp.Return, "return values should be the same")
		for name, deps := range expectedReturnDeps {
			v, ok := resp.Return.GetOk(name)
			if !ok {
				assert.Fail(t, "return value for %q should be present", name)
				continue
			}
			assert.Equal(t, deps, v.Dependencies(), "return dependencies for %q should be the same", name)
		}
	})
}

func must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}

func exampleOlds() (resource.PropertyMap, map[string]any) {
	return resource.PropertyMap{
			"f1": resource.NewProperty("s1"),
			"f2": resource.NewProperty(1.0),
			"f3": resource.NewProperty(resource.PropertyMap{
				"n1": resource.NewProperty("nv1"),
				"n2": resource.NewProperty(true),
			}),
		}, map[string]any{
			"f1": "s1",
			"f2": 1.0,
			"f3": map[string]any{
				"n1": "nv1",
				"n2": true,
			},
		}
}

func exampleNews() (resource.PropertyMap, map[string]any) {
	return resource.PropertyMap{
			"f1": resource.NewProperty("s1"),
			"f3": resource.NewProperty(resource.PropertyMap{
				"n1": resource.NewProperty("nv1"),
				"n2": resource.NewProperty(true),
			}),
			"f4": resource.NewProperty([]resource.PropertyValue{
				resource.NewProperty(2.0),
				resource.NewProperty("e1"),
			}),
		}, map[string]any{
			"f1": "s1",
			"f3": map[string]any{
				"n1": "nv1",
				"n2": true,
			},
			"f4": []any{
				2.0,
				"e1",
			},
		}
}

// testRPCCheck tests a check function against a series of inputs.
func testRPCCheck(
	t *testing.T,
	setup func(
		func(context.Context, *rpc.CheckRequest) (*rpc.CheckResponse, error),
	) func(p.CheckRequest) (p.CheckResponse, error),
) {
	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		olds, expectedOlds := exampleOlds()
		news, expectedNews := exampleNews()
		resp, err := setup(func(_ context.Context, req *rpc.CheckRequest) (*rpc.CheckResponse, error) {
			assert.Equal(t, "some-urn", req.GetUrn())
			assert.Equal(t, []byte("12345"), req.GetRandomSeed())
			assert.Equal(t, expectedOlds, req.GetOlds().AsMap())
			assert.Equal(t, expectedNews, req.GetNews().AsMap())
			return &rpc.CheckResponse{
				Inputs: must(structpb.NewStruct(map[string]any{
					"r1": []any{
						"e1",
						"e2",
					},
					"r2": false,
				})),
			}, nil
		})(p.CheckRequest{
			Urn:        "some-urn",
			State:      resource.FromResourcePropertyValue(resource.NewProperty(olds)).AsMap(),
			Inputs:     resource.FromResourcePropertyValue(resource.NewProperty(news)).AsMap(),
			RandomSeed: []byte("12345"),
		})
		require.NoError(t, err)
		assert.Equal(t, p.CheckResponse{
			Inputs: property.NewMap(map[string]property.Value{
				"r1": property.New([]property.Value{
					property.New("e1"),
					property.New("e2"),
				}),
				"r2": property.New(false),
			}),
		}, resp)
	})
	t.Run("failures", func(t *testing.T) {
		t.Parallel()
		resp, err := setup(func(context.Context, *rpc.CheckRequest) (*rpc.CheckResponse, error) {
			return &rpc.CheckResponse{
				Failures: []*rpc.CheckFailure{
					{Property: "some-prop", Reason: "some-reason"},
					{Property: "another", Reason: "second"},
					{Property: "empty"}, {Reason: "empty"},
				},
			}, nil
		})(p.CheckRequest{
			Urn: "some-urn",
		})
		require.NoError(t, err)
		assert.Equal(t, p.CheckResponse{
			Failures: []p.CheckFailure{
				{Property: "some-prop", Reason: "some-reason"},
				{Property: "another", Reason: "second"},
				{Property: "empty"}, {Reason: "empty"},
			},
		}, resp)
	})
	t.Run("error", func(t *testing.T) {
		t.Parallel()
		_, err := setup(func(context.Context, *rpc.CheckRequest) (*rpc.CheckResponse, error) {
			return nil, fmt.Errorf("check didn't work")
		})(p.CheckRequest{
			Urn: "some-urn",
		})
		assert.ErrorContains(t, err, "check didn't work")
	})
}

func noDetailedDiff(m map[string]p.PropertyDiff) map[string]p.PropertyDiff {
	if m == nil {
		m = map[string]p.PropertyDiff{}
	}
	m["__x-force-no-detailed-diff"] = p.PropertyDiff{}
	return m
}

// testRPCDiff tests a check function against a series of inputs.
func testRPCDiff(
	t *testing.T,
	setup func(
		func(context.Context, *rpc.DiffRequest) (*rpc.DiffResponse, error),
	) func(p.DiffRequest) (p.DiffResponse, error),
) {
	t.Run("translate-inputs", func(t *testing.T) {
		t.Parallel()

		olds, expectedOlds := exampleOlds()
		news, expectedNews := exampleNews()

		resp, err := setup(func(_ context.Context, req *rpc.DiffRequest) (*rpc.DiffResponse, error) {
			assert.Equal(t, "some-id", req.GetId())
			assert.Equal(t, "my-urn", req.GetUrn())
			assert.Equal(t, expectedOlds, req.GetOlds().AsMap())
			assert.Equal(t, expectedNews, req.GetNews().AsMap())
			assert.Equal(t, []string{"field1", "field2"}, req.GetIgnoreChanges())

			return &rpc.DiffResponse{DeleteBeforeReplace: true}, nil
		})(p.DiffRequest{
			ID:            "some-id",
			Urn:           "my-urn",
			State:         resource.FromResourcePropertyValue(resource.NewProperty(olds)).AsMap(),
			Inputs:        resource.FromResourcePropertyValue(resource.NewProperty(news)).AsMap(),
			IgnoreChanges: []string{"field1", "field2"},
		})

		require.NoError(t, err)
		assert.Equal(t, p.DiffResponse{
			DeleteBeforeReplace: true,
			DetailedDiff:        noDetailedDiff(nil),
		}, resp)
	})

	t.Run("detailed-diff", func(t *testing.T) {
		t.Parallel()

		resp, err := setup(func(context.Context, *rpc.DiffRequest) (*rpc.DiffResponse, error) {
			return &rpc.DiffResponse{
				HasDetailedDiff: true,
				Changes:         rpc.DiffResponse_DIFF_SOME,
				DetailedDiff: map[string]*rpc.PropertyDiff{
					"add":            {Kind: rpc.PropertyDiff_ADD, InputDiff: true},
					"add_replace":    {Kind: rpc.PropertyDiff_ADD_REPLACE},
					"delete":         {Kind: rpc.PropertyDiff_DELETE},
					"delete_replace": {Kind: rpc.PropertyDiff_DELETE_REPLACE},
					"update":         {Kind: rpc.PropertyDiff_UPDATE},
					"update_replace": {Kind: rpc.PropertyDiff_UPDATE_REPLACE},
					"nested.field":   {Kind: rpc.PropertyDiff_UPDATE, InputDiff: true},
				},
			}, nil
		})(p.DiffRequest{})

		require.NoError(t, err)
		assert.Equal(t, p.DiffResponse{
			HasChanges: true,
			DetailedDiff: map[string]p.PropertyDiff{
				"add":            {Kind: p.Add, InputDiff: true},
				"add_replace":    {Kind: p.AddReplace},
				"delete":         {Kind: p.Delete},
				"delete_replace": {Kind: p.DeleteReplace},
				"update":         {Kind: p.Update},
				"update_replace": {Kind: p.UpdateReplace},
				"nested.field":   {Kind: p.Update, InputDiff: true},
			},
		}, resp)
	})

	t.Run("no-diff", func(t *testing.T) {
		t.Parallel()

		resp, err := setup(func(context.Context, *rpc.DiffRequest) (*rpc.DiffResponse, error) {
			return &rpc.DiffResponse{
				Stables: []string{"f1", "f2"},
				Changes: rpc.DiffResponse_DIFF_NONE,
			}, nil
		})(p.DiffRequest{})

		require.NoError(t, err)
		assert.Equal(t, p.DiffResponse{
			HasChanges:   false,
			DetailedDiff: noDetailedDiff(nil),
		}, resp)

	})
	t.Run("simple-diff", func(t *testing.T) {
		t.Parallel()

		resp, err := setup(func(context.Context, *rpc.DiffRequest) (*rpc.DiffResponse, error) {
			return &rpc.DiffResponse{
				Replaces:        []string{"r1", "r2"},
				Stables:         []string{"f1", "f2"},
				Changes:         rpc.DiffResponse_DIFF_SOME,
				Diffs:           []string{"r1", "r2", "f3"},
				HasDetailedDiff: false,
			}, nil
		})(p.DiffRequest{})

		require.NoError(t, err)
		assert.Equal(t, p.DiffResponse{
			HasChanges: true,
			DetailedDiff: noDetailedDiff(map[string]p.PropertyDiff{
				"r1": {Kind: p.UpdateReplace},
				"r2": {Kind: p.UpdateReplace},
				"f3": {Kind: p.Update},
			}),
		}, resp)
	})
	t.Run("error", func(t *testing.T) {
		t.Parallel()

		_, err := setup(func(context.Context, *rpc.DiffRequest) (*rpc.DiffResponse, error) {
			return nil, fmt.Errorf("the diff went wrong")
		})(p.DiffRequest{})

		assert.ErrorContains(t, err, "the diff went wrong")
	})
}

type rpcTestServer struct {
	rpc.UnimplementedResourceProviderServer
	onGetSchema   func(context.Context, *rpc.GetSchemaRequest) (*rpc.GetSchemaResponse, error)
	onCancel      func(context.Context, *emptypb.Empty) (*emptypb.Empty, error)
	onCheckConfig func(context.Context, *rpc.CheckRequest) (*rpc.CheckResponse, error)
	onCheck       func(context.Context, *rpc.CheckRequest) (*rpc.CheckResponse, error)
	onDiffConfig  func(context.Context, *rpc.DiffRequest) (*rpc.DiffResponse, error)
	onDiff        func(context.Context, *rpc.DiffRequest) (*rpc.DiffResponse, error)
	onConfigure   func(context.Context, *rpc.ConfigureRequest) (*rpc.ConfigureResponse, error)
	onCreate      func(context.Context, *rpc.CreateRequest) (*rpc.CreateResponse, error)
	onInvoke      func(context.Context, *rpc.InvokeRequest) (*rpc.InvokeResponse, error)
	onDelete      func(context.Context, *rpc.DeleteRequest) (*emptypb.Empty, error)
	onRead        func(context.Context, *rpc.ReadRequest) (*rpc.ReadResponse, error)
	onUpdate      func(context.Context, *rpc.UpdateRequest) (*rpc.UpdateResponse, error)
	onConstruct   func(context.Context, *rpc.ConstructRequest) (*rpc.ConstructResponse, error)
	onCall        func(context.Context, *rpc.CallRequest) (*rpc.CallResponse, error)
}

func (r rpcTestServer) GetSchema(ctx context.Context, req *rpc.GetSchemaRequest) (*rpc.GetSchemaResponse, error) {
	return r.onGetSchema(ctx, req)
}

func (r rpcTestServer) Cancel(ctx context.Context, e *emptypb.Empty) (*emptypb.Empty, error) {
	return r.onCancel(ctx, e)
}

func (r rpcTestServer) CheckConfig(ctx context.Context, req *rpc.CheckRequest) (*rpc.CheckResponse, error) {
	return r.onCheckConfig(ctx, req)
}

func (r rpcTestServer) Check(ctx context.Context, req *rpc.CheckRequest) (*rpc.CheckResponse, error) {
	return r.onCheck(ctx, req)
}

func (r rpcTestServer) DiffConfig(ctx context.Context, req *rpc.DiffRequest) (*rpc.DiffResponse, error) {
	return r.onDiffConfig(ctx, req)
}

func (r rpcTestServer) Diff(ctx context.Context, req *rpc.DiffRequest) (*rpc.DiffResponse, error) {
	return r.onDiff(ctx, req)
}

func (r rpcTestServer) Configure(ctx context.Context, req *rpc.ConfigureRequest) (*rpc.ConfigureResponse, error) {
	return r.onConfigure(ctx, req)
}

func (r rpcTestServer) Create(ctx context.Context, req *rpc.CreateRequest) (*rpc.CreateResponse, error) {
	return r.onCreate(ctx, req)
}

func (r rpcTestServer) Invoke(ctx context.Context, req *rpc.InvokeRequest) (*rpc.InvokeResponse, error) {
	return r.onInvoke(ctx, req)
}

func (r rpcTestServer) Delete(ctx context.Context, req *rpc.DeleteRequest) (*emptypb.Empty, error) {
	return r.onDelete(ctx, req)
}

func (r rpcTestServer) Read(ctx context.Context, req *rpc.ReadRequest) (*rpc.ReadResponse, error) {
	return r.onRead(ctx, req)
}

func (r rpcTestServer) Update(ctx context.Context, req *rpc.UpdateRequest) (*rpc.UpdateResponse, error) {
	return r.onUpdate(ctx, req)
}

func (r rpcTestServer) Construct(ctx context.Context, req *rpc.ConstructRequest) (*rpc.ConstructResponse, error) {
	return r.onConstruct(ctx, req)
}

func (r rpcTestServer) Call(ctx context.Context, req *rpc.CallRequest) (*rpc.CallResponse, error) {
	return r.onCall(ctx, req)
}

func rpcServer(t *testing.T, server rpcTestServer) integration.Server {
	t.Helper()

	s, err := integration.NewServer(t.Context(), "test",
		semver.Version{Major: 1},
		integration.WithProvider(wraprpc.Provider(server)),
	)
	require.NoError(t, err)

	return s
}
