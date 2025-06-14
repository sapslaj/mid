package tests

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

		"expandArgumentVars": {
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
							property.New("/create"),
							property.New("/update"),
						}),
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
								property.New("/create"),
								property.New("/update"),
							}),
						}),
						"expandArgumentVars": property.New(true),
					}),
					AssertCommand: "test -f /update",
				},
			},
			AssertDeleteCommand: "test ! -f /create && test ! -f /update",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tc.Resource = "mid:resource:Exec"

			harness := NewProviderTestHarness(t, testmachine.Config{
				Backend: testmachine.DockerBackend,
			})
			defer harness.Close()

			tc.Run(t, harness)
		})
	}
}

func TestResourceExec_logging(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t, testmachine.Config{
		Backend: testmachine.DockerBackend,
	})
	defer harness.Close()

	props := map[string]property.Value{
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
	}

	createResponse, err := harness.Server.Create(p.CreateRequest{
		Urn:        MakeURN("mid:resource:Exec"),
		Properties: property.NewMap(props),
	})
	require.NoError(t, err)

	assert.Equal(t, "this is create stdout\n", createResponse.Properties.Get("stdout").AsString())
	assert.Equal(t, "this is create stderr\n", createResponse.Properties.Get("stderr").AsString())

	updateResponse, err := harness.Server.Update(p.UpdateRequest{
		Urn:    MakeURN("mid:resource:Exec"),
		State:  createResponse.Properties,
		Inputs: property.NewMap(props),
	})
	require.NoError(t, err)

	assert.Equal(t, "this is update stdout\n", updateResponse.Properties.Get("stdout").AsString())
	assert.Equal(t, "this is update stderr\n", updateResponse.Properties.Get("stderr").AsString())
}
