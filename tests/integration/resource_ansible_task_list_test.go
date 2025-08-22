package tests

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/property"
	p "github.com/sapslaj/mid/pkg/providerfw"
	"github.com/stretchr/testify/assert"

	"github.com/sapslaj/mid/tests/testmachine"
)

func TestResourceAnsibleTaskList(t *testing.T) {
	t.Parallel()

	tests := map[string]LifeCycleTest{
		"runs tasks in order": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"tasks": property.New(map[string]property.Value{
						"create": property.New([]property.Value{
							property.New(map[string]property.Value{
								"module": property.New("file"),
								"args": property.New(map[string]property.Value{
									"path":  property.New("/testing"),
									"state": property.New("touch"),
								}),
							}),
							property.New(map[string]property.Value{
								"module": property.New("blockinfile"),
								"args": property.New(map[string]property.Value{
									"path":  property.New("/testing"),
									"state": property.New("present"),
									"block": property.New("creating"),
								}),
							}),
						}),
						"update": property.New([]property.Value{
							property.New(map[string]property.Value{
								"module": property.New("blockinfile"),
								"args": property.New(map[string]property.Value{
									"path":  property.New("/testing"),
									"state": property.New("present"),
									"block": property.New("creating"),
								}),
							}),
						}),
						"delete": property.New([]property.Value{
							property.New(map[string]property.Value{
								"module": property.New("file"),
								"args": property.New(map[string]property.Value{
									"path":  property.New("/testing"),
									"state": property.New("absent"),
								}),
							}),
						}),
					}),
					"triggers": property.New(map[string]property.Value{
						"refresh": property.New([]property.Value{
							property.New("1"),
						}),
					}),
				}),
				AssertBeforeCommand:      "test ! -f /testing",
				AssertAfterDryRunCommand: "test ! -f /testing",
				AssertCommand: `
					set -eu
					ls -lah /testing
					test -f /testing
					cat /testing
					grep creating /testing
				`,
				Hook: func(t *testing.T, _ property.Map, output property.Map) {
					results := output.Get("results").AsMap()
					tasks := results.Get("tasks").AsArray()

					assert.Equal(t, "create", results.Get("lifecycle").AsString())
					assert.Equal(t, 2, tasks.Len())
					tasks.All(func(_ int, v property.Value) bool {
						res := v.AsMap()
						assert.Equal(t, 0.0, res.Get("exitCode").AsNumber())
						assert.True(t, res.Get("success").AsBool())
						return true
					})
				},
			},
			Updates: []Operation{
				{
					Inputs: property.NewMap(map[string]property.Value{
						"tasks": property.New(map[string]property.Value{
							"create": property.New([]property.Value{
								property.New(map[string]property.Value{
									"module": property.New("file"),
									"args": property.New(map[string]property.Value{
										"path":  property.New("/testing"),
										"state": property.New("touch"),
									}),
								}),
								property.New(map[string]property.Value{
									"module": property.New("blockinfile"),
									"args": property.New(map[string]property.Value{
										"path":  property.New("/testing"),
										"state": property.New("present"),
										"block": property.New("creating"),
									}),
								}),
							}),
							"update": property.New([]property.Value{
								property.New(map[string]property.Value{
									"module": property.New("blockinfile"),
									"args": property.New(map[string]property.Value{
										"path":  property.New("/testing"),
										"state": property.New("present"),
										"block": property.New("creating"),
									}),
								}),
							}),
							"delete": property.New([]property.Value{
								property.New(map[string]property.Value{
									"module": property.New("file"),
									"args": property.New(map[string]property.Value{
										"path":  property.New("/testing"),
										"state": property.New("absent"),
									}),
								}),
							}),
						}),
						"triggers": property.New(map[string]property.Value{
							"refresh": property.New([]property.Value{
								property.New("2"),
							}),
						}),
					}),
					ExpectedDiff: &p.DiffResponse{
						DeleteBeforeReplace: true,
						HasChanges:          true,
						DetailedDiff: map[string]p.PropertyDiff{
							"triggers": {
								Kind:      p.Update,
								InputDiff: true,
							},
						},
					},
					AssertBeforeCommand: "stat -c %y /testing | sudo tee /testing-statmod",
					AssertAfterDryRunCommand: `
						set -eux
						ls -lah /testing
						test -f /testing
						cat /testing
						grep creating /testing
						echo "stat before:"
						cat /testing-statmod
						echo "stat after:"
						stat -c %y /testing
						test "$(cat /testing-statmod)" = "$(stat -c %y /testing)"
					`,
					AssertCommand: `
						set -eux
						ls -lah /testing
						test -f /testing
						cat /testing
						grep creating /testing
						echo "stat before:"
						cat /testing-statmod
						echo "stat after:"
						stat -c %y /testing
						test "$(cat /testing-statmod)" = "$(stat -c %y /testing)"
					`,
					Hook: func(t *testing.T, _ property.Map, output property.Map) {
						results := output.Get("results").AsMap()
						tasks := results.Get("tasks").AsArray()

						assert.Equal(t, "update", results.Get("lifecycle").AsString())
						assert.Equal(t, 1, tasks.Len())
						tasks.All(func(_ int, v property.Value) bool {
							res := v.AsMap()
							assert.Equal(t, 0.0, res.Get("exitCode").AsNumber())
							assert.True(t, res.Get("success").AsBool())
							return true
						})
					},
				},
				{
					Inputs: property.NewMap(map[string]property.Value{
						"tasks": property.New(map[string]property.Value{
							"create": property.New([]property.Value{
								property.New(map[string]property.Value{
									"module": property.New("file"),
									"args": property.New(map[string]property.Value{
										"path":  property.New("/testing"),
										"state": property.New("touch"),
									}),
								}),
								property.New(map[string]property.Value{
									"module": property.New("blockinfile"),
									"args": property.New(map[string]property.Value{
										"path":  property.New("/testing"),
										"state": property.New("present"),
										"block": property.New("updating"),
									}),
								}),
							}),
							"update": property.New([]property.Value{
								property.New(map[string]property.Value{
									"module": property.New("blockinfile"),
									"args": property.New(map[string]property.Value{
										"path":  property.New("/testing"),
										"state": property.New("present"),
										"block": property.New("updating"),
									}),
								}),
							}),
							"delete": property.New([]property.Value{
								property.New(map[string]property.Value{
									"module": property.New("file"),
									"args": property.New(map[string]property.Value{
										"path":  property.New("/testing"),
										"state": property.New("absent"),
									}),
								}),
							}),
						}),
						"triggers": property.New(map[string]property.Value{
							"refresh": property.New([]property.Value{
								property.New("2"),
							}),
						}),
					}),
					ExpectedDiff: &p.DiffResponse{
						DeleteBeforeReplace: true,
						HasChanges:          true,
						DetailedDiff: map[string]p.PropertyDiff{
							"tasks.create[1].args.block": {
								Kind:      p.Update,
								InputDiff: true,
							},
							"tasks.update[0].args.block": {
								Kind:      p.Update,
								InputDiff: true,
							},
						},
					},
					AssertBeforeCommand: "stat -c %y /testing | sudo tee /testing-statmod",
					AssertAfterDryRunCommand: `
						set -eux
						ls -lah /testing
						test -f /testing
						cat /testing
						grep creating /testing
						echo "stat before:"
						cat /testing-statmod
						echo "stat after:"
						stat -c %y /testing
						test "$(cat /testing-statmod)" = "$(stat -c %y /testing)"
					`,
					AssertCommand: `
						set -eux
						ls -lah /testing
						test -f /testing
						cat /testing
						grep updating /testing
						echo "stat before:"
						cat /testing-statmod
						echo "stat after:"
						stat -c %y /testing
						test "$(cat /testing-statmod)" != "$(stat -c %y /testing)"
					`,
					Hook: func(t *testing.T, _ property.Map, output property.Map) {
						results := output.Get("results").AsMap()
						tasks := results.Get("tasks").AsArray()

						assert.Equal(t, "update", results.Get("lifecycle").AsString())
						assert.Equal(t, 1, tasks.Len())
						tasks.All(func(_ int, v property.Value) bool {
							res := v.AsMap()
							assert.Equal(t, 0.0, res.Get("exitCode").AsNumber())
							assert.True(t, res.Get("success").AsBool())
							return true
						})
					},
				},
			},
			AssertBeforeDeleteCommand: `
				set -eu
				ls -lah /testing
				test -f /testing
				cat /testing
			`,
			AssertDeleteCommand: `
				set -eu
				ls -lah /
				test ! -f /testing
			`,
		},

		"ignore errors": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"tasks": property.New(map[string]property.Value{
						"create": property.New([]property.Value{
							property.New(map[string]property.Value{
								"module": property.New("blockinfile"),
								"args": property.New(map[string]property.Value{
									"path":   property.New("/testing"),
									"state":  property.New("present"),
									"block":  property.New("creating"),
									"create": property.New(false),
								}),
								"ignoreErrors": property.New(true),
							}),
						}),
					}),
				}),
				AssertBeforeCommand:      "test ! -f /testing",
				AssertAfterDryRunCommand: "test ! -f /testing",
				AssertCommand:            "test ! -f /testing",
				ExpectFailure:            false,
				Hook: func(t *testing.T, _ property.Map, output property.Map) {
					results := output.Get("results").AsMap()
					tasks := results.Get("tasks").AsArray()

					assert.Equal(t, "create", results.Get("lifecycle").AsString())
					assert.Equal(t, 1, tasks.Len())

					task := tasks.Get(0).AsMap()

					assert.NotEqual(t, 0.0, task.Get("exitCode").AsNumber())
					assert.False(t, task.Get("success").AsBool())
				},
			},
			Updates: []Operation{
				{
					Inputs: property.NewMap(map[string]property.Value{
						"tasks": property.New(map[string]property.Value{
							"create": property.New([]property.Value{
								property.New(map[string]property.Value{
									"module": property.New("blockinfile"),
									"args": property.New(map[string]property.Value{
										"path":   property.New("/testing"),
										"state":  property.New("present"),
										"block":  property.New("creating"),
										"create": property.New(false),
									}),
									"ignoreErrors": property.New(false),
								}),
							}),
						}),
					}),
					ExpectedDiff: &p.DiffResponse{
						DeleteBeforeReplace: true,
						HasChanges:          true,
						DetailedDiff: map[string]p.PropertyDiff{
							"tasks.create[0].ignoreErrors": {
								Kind:      p.Update,
								InputDiff: true,
							},
						},
					},
					AssertBeforeCommand:      "test ! -f /testing",
					AssertAfterDryRunCommand: "test ! -f /testing",
					AssertCommand:            "test ! -f /testing",
					ExpectFailure:            true,
					Hook: func(t *testing.T, _ property.Map, output property.Map) {
						results := output.Get("results").AsMap()
						tasks := results.Get("tasks").AsArray()

						assert.Equal(t, "create", results.Get("lifecycle").AsString())
						assert.Equal(t, 1, tasks.Len())

						task := tasks.Get(0).AsMap()

						assert.NotEqual(t, 0.0, task.Get("exitCode").AsNumber())
						assert.False(t, task.Get("success").AsBool())
					},
				},
			},
		},

		"omitted update will use create": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"tasks": property.New(map[string]property.Value{
						"create": property.New([]property.Value{
							property.New(map[string]property.Value{
								"module": property.New("blockinfile"),
								"args": property.New(map[string]property.Value{
									"path":   property.New("/testing"),
									"state":  property.New("present"),
									"block":  property.New("creating"),
									"create": property.New(true),
								}),
							}),
						}),
					}),
				}),
				AssertBeforeCommand:      "test ! -f /testing",
				AssertAfterDryRunCommand: "test ! -f /testing",
				AssertCommand: `
					set -eu
					ls -lah /testing
					test -f /testing
					cat /testing
					grep creating /testing
				`,
				Hook: func(t *testing.T, _ property.Map, output property.Map) {
					results := output.Get("results").AsMap()
					tasks := results.Get("tasks").AsArray()

					assert.Equal(t, "create", results.Get("lifecycle").AsString())
					assert.Equal(t, 1, tasks.Len())
					tasks.All(func(_ int, v property.Value) bool {
						res := v.AsMap()
						assert.Equal(t, 0.0, res.Get("exitCode").AsNumber())
						assert.True(t, res.Get("success").AsBool())
						return true
					})
				},
			},
			Updates: []Operation{
				{
					Inputs: property.NewMap(map[string]property.Value{
						"tasks": property.New(map[string]property.Value{
							"create": property.New([]property.Value{
								property.New(map[string]property.Value{
									"module": property.New("blockinfile"),
									"args": property.New(map[string]property.Value{
										"path":   property.New("/testing"),
										"state":  property.New("present"),
										"block":  property.New("updating"),
										"create": property.New(true),
									}),
								}),
							}),
							"delete": property.New([]property.Value{
								property.New(map[string]property.Value{
									"module": property.New("file"),
									"args": property.New(map[string]property.Value{
										"path":  property.New("/testing"),
										"state": property.New("absent"),
									}),
								}),
							}),
						}),
					}),
					ExpectedDiff: &p.DiffResponse{
						DeleteBeforeReplace: true,
						HasChanges:          true,
						DetailedDiff: map[string]p.PropertyDiff{
							"tasks.create[0].args.block": {
								Kind:      p.Update,
								InputDiff: true,
							},
							"tasks.delete": {
								Kind:      p.Add,
								InputDiff: true,
							},
						},
					},
					AssertBeforeCommand: `
						set -eu
						ls -lah /testing
						test -f /testing
						cat /testing
						grep creating /testing
					`,
					AssertAfterDryRunCommand: `
						set -eu
						ls -lah /testing
						test -f /testing
						cat /testing
						grep creating /testing
					`,
					AssertCommand: `
						set -eu
						ls -lah /testing
						test -f /testing
						cat /testing
						grep updating /testing
					`,
					Hook: func(t *testing.T, _ property.Map, output property.Map) {
						results := output.Get("results").AsMap()
						tasks := results.Get("tasks").AsArray()

						assert.Equal(t, "update", results.Get("lifecycle").AsString())
						assert.Equal(t, 1, tasks.Len())
						tasks.All(func(_ int, v property.Value) bool {
							res := v.AsMap()
							assert.Equal(t, 0.0, res.Get("exitCode").AsNumber())
							assert.True(t, res.Get("success").AsBool())
							return true
						})
					},
				},
			},
			AssertBeforeDeleteCommand: `
				set -eu
				ls -lah /testing
				test -f /testing
				cat /testing
				grep updating /testing
			`,
			AssertDeleteCommand: `
				set -eu
				ls -lah /
				test ! -f /testing
			`,
		},

		"omitted delete will do nothing": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"tasks": property.New(map[string]property.Value{
						"create": property.New([]property.Value{
							property.New(map[string]property.Value{
								"module": property.New("blockinfile"),
								"args": property.New(map[string]property.Value{
									"path":   property.New("/testing"),
									"state":  property.New("present"),
									"block":  property.New("creating"),
									"create": property.New(true),
								}),
							}),
						}),
					}),
				}),
				AssertBeforeCommand:      "test ! -f /testing",
				AssertAfterDryRunCommand: "test ! -f /testing",
				AssertCommand: `
					set -eu
					ls -lah /testing
					test -f /testing
					cat /testing
					grep creating /testing
				`,
				Hook: func(t *testing.T, _ property.Map, output property.Map) {
					results := output.Get("results").AsMap()
					tasks := results.Get("tasks").AsArray()

					assert.Equal(t, "create", results.Get("lifecycle").AsString())
					assert.Equal(t, 1, tasks.Len())
					tasks.All(func(_ int, v property.Value) bool {
						res := v.AsMap()
						assert.Equal(t, 0.0, res.Get("exitCode").AsNumber())
						assert.True(t, res.Get("success").AsBool())
						return true
					})
				},
			},
			AssertBeforeDeleteCommand: `
				set -eu
				ls -lah /testing
				test -f /testing
				cat /testing
				grep creating /testing
			`,
			AssertDeleteCommand: `
				set -eu
				ls -lah /testing
				test -f /testing
				cat /testing
				grep creating /testing
			`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			harness := NewProviderTestHarness(t, testmachine.Config{
				Backend: testmachine.DockerBackend,
			})
			defer harness.Close()

			tc.Resource = "mid:resource:AnsibleTaskList"
			tc.Run(t, harness)
		})
	}
}
