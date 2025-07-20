package tests

import (
	"testing"

	p "github.com/sapslaj/mid/pkg/providerfw"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/assert"

	"github.com/sapslaj/mid/tests/testmachine"
)

func TestResourceExec(t *testing.T) {
	t.Parallel()

	tests := map[string]LifeCycleTest{
		"lifecycle": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"create": property.New(map[string]property.Value{
						"command": property.New([]property.Value{
							property.New("touch"),
							property.New("/create"),
						}),
					}),
					"update": property.New(map[string]property.Value{
						"command": property.New([]property.Value{
							property.New("touch"),
							property.New("/update"),
						}),
					}),
					"delete": property.New(map[string]property.Value{
						"command": property.New([]property.Value{
							property.New("rm"),
							property.New("-rf"),
							property.New("/create"),
							property.New("/update"),
						}),
					}),
				}),
				AssertCommand: "test -f /create",
			},
			Updates: []Operation{
				{
					Inputs: property.NewMap(map[string]property.Value{
						"create": property.New(map[string]property.Value{
							"command": property.New([]property.Value{
								property.New("touch"),
								property.New("/create"),
							}),
						}),
						"update": property.New(map[string]property.Value{
							"command": property.New([]property.Value{
								property.New("touch"),
								property.New("/update"),
							}),
						}),
						"delete": property.New(map[string]property.Value{
							"command": property.New([]property.Value{
								property.New("rm"),
								property.New("-rf"),
								property.New("/create"),
								property.New("/update"),
							}),
						}),
					}),
					AssertCommand: "test -f /update",
				},
			},
			AssertDeleteCommand: "test ! -f /create && test ! -f /update",
		},

		"configurable deleteBeforeReplace": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"create": property.New(map[string]property.Value{
						"command": property.New([]property.Value{
							property.New("touch"),
							property.New("/create"),
						}),
					}),
					"deleteBeforeReplace": property.New(true),
					"triggers": property.New(property.NewMap(map[string]property.Value{
						"refresh": property.New(property.NewArray([]property.Value{
							property.New("1"),
						})),
					})),
				}),
			},
			Updates: []Operation{
				{
					Inputs: property.NewMap(map[string]property.Value{
						"create": property.New(map[string]property.Value{
							"command": property.New([]property.Value{
								property.New("touch"),
								property.New("/create"),
							}),
						}),
						"deleteBeforeReplace": property.New(true),
						"triggers": property.New(property.NewMap(map[string]property.Value{
							"refresh": property.New(property.NewArray([]property.Value{
								property.New("1"),
							})),
						})),
					}),
					ExpectedDiff: &p.DiffResponse{
						DeleteBeforeReplace: true,
						HasChanges:          false,
						DetailedDiff:        map[string]p.PropertyDiff{},
					},
				},
				{
					Inputs: property.NewMap(map[string]property.Value{
						"create": property.New(map[string]property.Value{
							"command": property.New([]property.Value{
								property.New("touch"),
								property.New("/create"),
							}),
						}),
						"deleteBeforeReplace": property.New(false),
						"triggers": property.New(property.NewMap(map[string]property.Value{
							"refresh": property.New(property.NewArray([]property.Value{
								property.New("2"),
							})),
						})),
					}),
					ExpectedDiff: &p.DiffResponse{
						DeleteBeforeReplace: false,
						HasChanges:          true,
						DetailedDiff: map[string]p.PropertyDiff{
							"triggers": {
								Kind:      p.Update,
								InputDiff: true,
							},
						},
					},
				},
			},
		},

		"dir": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"create": property.New(map[string]property.Value{
						"command": property.New([]property.Value{
							property.New("touch"),
							property.New("create"),
						}),
					}),
					"update": property.New(map[string]property.Value{
						"command": property.New([]property.Value{
							property.New("touch"),
							property.New("./tmp/update"),
						}),
						"dir": property.New("/"),
					}),
					"delete": property.New(map[string]property.Value{
						"command": property.New([]property.Value{
							property.New("rm"),
							property.New("-rf"),
							property.New("create"),
							property.New("update"),
						}),
					}),
					"dir": property.New("/tmp"),
				}),
				AssertCommand: "test -f /tmp/create",
			},
			Updates: []Operation{
				{
					Inputs: property.NewMap(map[string]property.Value{
						"create": property.New(map[string]property.Value{
							"command": property.New([]property.Value{
								property.New("touch"),
								property.New("create"),
							}),
						}),
						"update": property.New(map[string]property.Value{
							"command": property.New([]property.Value{
								property.New("touch"),
								property.New("./tmp/update"),
							}),
							"dir": property.New("/"),
						}),
						"delete": property.New(map[string]property.Value{
							"command": property.New([]property.Value{
								property.New("rm"),
								property.New("-rf"),
								property.New("create"),
								property.New("update"),
							}),
						}),
						"dir": property.New("/tmp"),
					}),
					AssertCommand: "test -f /tmp/update",
				},
			},
			AssertDeleteCommand: "test ! -f /tmp/create && test ! -f /tmp/update",
		},

		"environment": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"create": property.New(map[string]property.Value{
						"command": property.New([]property.Value{
							property.New("/bin/sh"),
							property.New("-c"),
							property.New("echo $OP > $FILE"),
						}),
						"environment": property.New(map[string]property.Value{
							"OP": property.New("create"),
						}),
					}),
					"update": property.New(map[string]property.Value{
						"command": property.New([]property.Value{
							property.New("/bin/sh"),
							property.New("-c"),
							property.New("echo $OP > $FILE"),
						}),
						"environment": property.New(map[string]property.Value{
							"OP": property.New("update"),
						}),
					}),
					"delete": property.New(map[string]property.Value{
						"command": property.New([]property.Value{
							property.New("/bin/sh"),
							property.New("-c"),
							property.New("rm -f $FILE"),
						}),
					}),
					"environment": property.New(map[string]property.Value{
						"FILE": property.New("/tmp/environment"),
					}),
				}),
				AssertCommand: "grep -q create /tmp/environment",
			},
			Updates: []Operation{
				{
					Inputs: property.NewMap(map[string]property.Value{
						"create": property.New(map[string]property.Value{
							"command": property.New([]property.Value{
								property.New("/bin/sh"),
								property.New("-c"),
								property.New("echo $OP > $FILE"),
							}),
							"environment": property.New(map[string]property.Value{
								"OP": property.New("create"),
							}),
						}),
						"update": property.New(map[string]property.Value{
							"command": property.New([]property.Value{
								property.New("/bin/sh"),
								property.New("-c"),
								property.New("echo $OP > $FILE"),
							}),
							"environment": property.New(map[string]property.Value{
								"OP": property.New("update"),
							}),
						}),
						"delete": property.New(map[string]property.Value{
							"command": property.New([]property.Value{
								property.New("/bin/sh"),
								property.New("-c"),
								property.New("rm -f $FILE"),
							}),
						}),
						"environment": property.New(map[string]property.Value{
							"FILE": property.New("/tmp/environment"),
						}),
					}),
					AssertCommand: "grep -q update /tmp/environment",
				},
			},
			AssertDeleteCommand: "test ! -f /tmp/environment",
		},

		"environment with expandArgumentVars": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"create": property.New(map[string]property.Value{
						"command": property.New([]property.Value{
							property.New("touch"),
							property.New("$FILE"),
						}),
						"environment": property.New(map[string]property.Value{
							"FILE": property.New("/create"),
						}),
					}),
					"update": property.New(map[string]property.Value{
						"command": property.New([]property.Value{
							property.New("touch"),
							property.New("$FILE"),
						}),
						"environment": property.New(map[string]property.Value{
							"FILE": property.New("/update"),
						}),
					}),
					"delete": property.New(map[string]property.Value{
						"command": property.New([]property.Value{
							property.New("rm"),
							property.New("-rf"),
							property.New("$CREATE"),
							property.New("$UPDATE"),
						}),
					}),
					"environment": property.New(map[string]property.Value{
						"CREATE": property.New("/create"),
						"UPDATE": property.New("/update"),
					}),
					"expandArgumentVars": property.New(true),
				}),
				AssertCommand: "test -f /create",
			},
			Updates: []Operation{
				{
					Inputs: property.NewMap(map[string]property.Value{
						"create": property.New(map[string]property.Value{
							"command": property.New([]property.Value{
								property.New("touch"),
								property.New("$FILE"),
							}),
							"environment": property.New(map[string]property.Value{
								"FILE": property.New("/create"),
							}),
						}),
						"update": property.New(map[string]property.Value{
							"command": property.New([]property.Value{
								property.New("touch"),
								property.New("$FILE"),
							}),
							"environment": property.New(map[string]property.Value{
								"FILE": property.New("/update"),
							}),
						}),
						"delete": property.New(map[string]property.Value{
							"command": property.New([]property.Value{
								property.New("rm"),
								property.New("-rf"),
								property.New("$CREATE"),
								property.New("$UPDATE"),
							}),
						}),
						"environment": property.New(map[string]property.Value{
							"CREATE": property.New("/create"),
							"UPDATE": property.New("/update"),
						}),
						"expandArgumentVars": property.New(true),
					}),
					AssertCommand: "test -f /update",
				},
			},
			AssertDeleteCommand: "test ! -f /create && test ! -f /update",
		},

		"only create": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"create": property.New(map[string]property.Value{
						"command": property.New([]property.Value{
							property.New("touch"),
							property.New("/create"),
						}),
					}),
					"triggers": property.New(property.NewMap(map[string]property.Value{
						"refresh": property.New(property.NewArray([]property.Value{
							property.New("1"),
						})),
					})),
				}),
				AssertCommand: "test -f /create",
			},
			Updates: []Operation{
				{
					Inputs: property.NewMap(map[string]property.Value{
						"create": property.New(map[string]property.Value{
							"command": property.New([]property.Value{
								property.New("touch"),
								property.New("/create"),
							}),
						}),
						"triggers": property.New(property.NewMap(map[string]property.Value{
							"refresh": property.New(property.NewArray([]property.Value{
								property.New("2"),
							})),
						})),
					}),
					AssertBeforeCommand: "sudo rm -f /create",
					AssertCommand:       "test -f /create",
				},
			},
			AssertDeleteCommand: "test -f /create",
		},

		"logging with defaults": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"create": property.New(map[string]property.Value{
						"command": property.New([]property.Value{
							property.New("/bin/sh"),
							property.New("-c"),
							property.New("echo this is create stdout\necho this is create stderr 1>&2\n"),
						}),
					}),
					"update": property.New(map[string]property.Value{
						"command": property.New([]property.Value{
							property.New("/bin/sh"),
							property.New("-c"),
							property.New("echo this is update stdout\necho this is update stderr 1>&2\n"),
						}),
					}),
					"triggers": property.New(property.NewMap(map[string]property.Value{
						"refresh": property.New(property.NewArray([]property.Value{
							property.New("1"),
						})),
					})),
				}),
				Hook: func(t *testing.T, inputs property.Map, output property.Map) {
					assert.Equal(t, "this is create stdout\n", output.Get("stdout").AsString())
					assert.Equal(t, "this is create stderr\n", output.Get("stderr").AsString())
				},
			},
			Updates: []Operation{
				{
					Inputs: property.NewMap(map[string]property.Value{
						"create": property.New(map[string]property.Value{
							"command": property.New([]property.Value{
								property.New("/bin/sh"),
								property.New("-c"),
								property.New("echo this is create stdout\necho this is create stderr 1>&2\n"),
							}),
						}),
						"update": property.New(map[string]property.Value{
							"command": property.New([]property.Value{
								property.New("/bin/sh"),
								property.New("-c"),
								property.New("echo this is update stdout\necho this is update stderr 1>&2\n"),
							}),
						}),
						"triggers": property.New(property.NewMap(map[string]property.Value{
							"refresh": property.New(property.NewArray([]property.Value{
								property.New("2"),
							})),
						})),
					}),
					Hook: func(t *testing.T, inputs property.Map, output property.Map) {
						assert.Equal(t, "this is update stdout\n", output.Get("stdout").AsString())
						assert.Equal(t, "this is update stderr\n", output.Get("stderr").AsString())
					},
				},
			},
		},

		"logging with ExecLoggingStdoutAndStderr": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"create": property.New(map[string]property.Value{
						"command": property.New([]property.Value{
							property.New("/bin/sh"),
							property.New("-c"),
							property.New("echo this is create stdout\necho this is create stderr 1>&2\n"),
						}),
					}),
					"update": property.New(map[string]property.Value{
						"command": property.New([]property.Value{
							property.New("/bin/sh"),
							property.New("-c"),
							property.New("echo this is update stdout\necho this is update stderr 1>&2\n"),
						}),
					}),
					"logging": property.New("stdoutAndStderr"),
					"triggers": property.New(property.NewMap(map[string]property.Value{
						"refresh": property.New(property.NewArray([]property.Value{
							property.New("1"),
						})),
					})),
				}),
				Hook: func(t *testing.T, inputs property.Map, output property.Map) {
					assert.Equal(t, "this is create stdout\n", output.Get("stdout").AsString())
					assert.Equal(t, "this is create stderr\n", output.Get("stderr").AsString())
				},
			},
			Updates: []Operation{
				{
					Inputs: property.NewMap(map[string]property.Value{
						"create": property.New(map[string]property.Value{
							"command": property.New([]property.Value{
								property.New("/bin/sh"),
								property.New("-c"),
								property.New("echo this is create stdout\necho this is create stderr 1>&2\n"),
							}),
						}),
						"update": property.New(map[string]property.Value{
							"command": property.New([]property.Value{
								property.New("/bin/sh"),
								property.New("-c"),
								property.New("echo this is update stdout\necho this is update stderr 1>&2\n"),
							}),
						}),
						"logging": property.New("stdoutAndStderr"),
						"triggers": property.New(property.NewMap(map[string]property.Value{
							"refresh": property.New(property.NewArray([]property.Value{
								property.New("2"),
							})),
						})),
					}),
					Hook: func(t *testing.T, inputs property.Map, output property.Map) {
						assert.Equal(t, "this is update stdout\n", output.Get("stdout").AsString())
						assert.Equal(t, "this is update stderr\n", output.Get("stderr").AsString())
					},
				},
			},
		},

		"logging with ExecLoggingNone": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"create": property.New(map[string]property.Value{
						"command": property.New([]property.Value{
							property.New("/bin/sh"),
							property.New("-c"),
							property.New("echo this is create stdout\necho this is create stderr 1>&2\n"),
						}),
					}),
					"update": property.New(map[string]property.Value{
						"command": property.New([]property.Value{
							property.New("/bin/sh"),
							property.New("-c"),
							property.New("echo this is update stdout\necho this is update stderr 1>&2\n"),
						}),
					}),
					"logging": property.New("none"),
					"triggers": property.New(property.NewMap(map[string]property.Value{
						"refresh": property.New(property.NewArray([]property.Value{
							property.New("1"),
						})),
					})),
				}),
				Hook: func(t *testing.T, inputs property.Map, output property.Map) {
					assert.Equal(t, "", output.Get("stdout").AsString())
					assert.Equal(t, "", output.Get("stderr").AsString())
				},
			},
			Updates: []Operation{
				{
					Inputs: property.NewMap(map[string]property.Value{
						"create": property.New(map[string]property.Value{
							"command": property.New([]property.Value{
								property.New("/bin/sh"),
								property.New("-c"),
								property.New("echo this is create stdout\necho this is create stderr 1>&2\n"),
							}),
						}),
						"update": property.New(map[string]property.Value{
							"command": property.New([]property.Value{
								property.New("/bin/sh"),
								property.New("-c"),
								property.New("echo this is update stdout\necho this is update stderr 1>&2\n"),
							}),
						}),
						"logging": property.New("none"),
						"triggers": property.New(property.NewMap(map[string]property.Value{
							"refresh": property.New(property.NewArray([]property.Value{
								property.New("2"),
							})),
						})),
					}),
					Hook: func(t *testing.T, inputs property.Map, output property.Map) {
						assert.Equal(t, "", output.Get("stdout").AsString())
						assert.Equal(t, "", output.Get("stderr").AsString())
					},
				},
			},
		},

		"logging with ExecLoggingStdout": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"create": property.New(map[string]property.Value{
						"command": property.New([]property.Value{
							property.New("/bin/sh"),
							property.New("-c"),
							property.New("echo this is create stdout\necho this is create stderr 1>&2\n"),
						}),
					}),
					"update": property.New(map[string]property.Value{
						"command": property.New([]property.Value{
							property.New("/bin/sh"),
							property.New("-c"),
							property.New("echo this is update stdout\necho this is update stderr 1>&2\n"),
						}),
					}),
					"logging": property.New("stdout"),
					"triggers": property.New(property.NewMap(map[string]property.Value{
						"refresh": property.New(property.NewArray([]property.Value{
							property.New("1"),
						})),
					})),
				}),
				Hook: func(t *testing.T, inputs property.Map, output property.Map) {
					assert.Equal(t, "this is create stdout\n", output.Get("stdout").AsString())
					assert.Equal(t, "", output.Get("stderr").AsString())
				},
			},
			Updates: []Operation{
				{
					Inputs: property.NewMap(map[string]property.Value{
						"create": property.New(map[string]property.Value{
							"command": property.New([]property.Value{
								property.New("/bin/sh"),
								property.New("-c"),
								property.New("echo this is create stdout\necho this is create stderr 1>&2\n"),
							}),
						}),
						"update": property.New(map[string]property.Value{
							"command": property.New([]property.Value{
								property.New("/bin/sh"),
								property.New("-c"),
								property.New("echo this is update stdout\necho this is update stderr 1>&2\n"),
							}),
						}),
						"logging": property.New("stdout"),
						"triggers": property.New(property.NewMap(map[string]property.Value{
							"refresh": property.New(property.NewArray([]property.Value{
								property.New("2"),
							})),
						})),
					}),
					Hook: func(t *testing.T, inputs property.Map, output property.Map) {
						assert.Equal(t, "this is update stdout\n", output.Get("stdout").AsString())
						assert.Equal(t, "", output.Get("stderr").AsString())
					},
				},
			},
		},

		"logging with ExecLoggingStderr": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"create": property.New(map[string]property.Value{
						"command": property.New([]property.Value{
							property.New("/bin/sh"),
							property.New("-c"),
							property.New("echo this is create stdout\necho this is create stderr 1>&2\n"),
						}),
					}),
					"update": property.New(map[string]property.Value{
						"command": property.New([]property.Value{
							property.New("/bin/sh"),
							property.New("-c"),
							property.New("echo this is update stdout\necho this is update stderr 1>&2\n"),
						}),
					}),
					"logging": property.New("stderr"),
					"triggers": property.New(property.NewMap(map[string]property.Value{
						"refresh": property.New(property.NewArray([]property.Value{
							property.New("1"),
						})),
					})),
				}),
				Hook: func(t *testing.T, inputs property.Map, output property.Map) {
					assert.Equal(t, "", output.Get("stdout").AsString())
					assert.Equal(t, "this is create stderr\n", output.Get("stderr").AsString())
				},
			},
			Updates: []Operation{
				{
					Inputs: property.NewMap(map[string]property.Value{
						"create": property.New(map[string]property.Value{
							"command": property.New([]property.Value{
								property.New("/bin/sh"),
								property.New("-c"),
								property.New("echo this is create stdout\necho this is create stderr 1>&2\n"),
							}),
						}),
						"update": property.New(map[string]property.Value{
							"command": property.New([]property.Value{
								property.New("/bin/sh"),
								property.New("-c"),
								property.New("echo this is update stdout\necho this is update stderr 1>&2\n"),
							}),
						}),
						"logging": property.New("stderr"),
						"triggers": property.New(property.NewMap(map[string]property.Value{
							"refresh": property.New(property.NewArray([]property.Value{
								property.New("2"),
							})),
						})),
					}),
					Hook: func(t *testing.T, inputs property.Map, output property.Map) {
						assert.Equal(t, "", output.Get("stdout").AsString())
						assert.Equal(t, "this is update stderr\n", output.Get("stderr").AsString())
					},
				},
			},
		},

		"failure": {
			Create: Operation{
				Inputs: property.NewMap(map[string]property.Value{
					"create": property.New(map[string]property.Value{
						"command": property.New([]property.Value{
							property.New("/bin/sh"),
							property.New("-c"),
							property.New("false"),
						}),
					}),
				}),
				ExpectFailure: true,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			harness := NewProviderTestHarness(t, testmachine.Config{
				Backend: testmachine.DockerBackend,
			})
			defer harness.Close()

			tc.Resource = "mid:resource:Exec"
			tc.Run(t, harness)
		})
	}
}
