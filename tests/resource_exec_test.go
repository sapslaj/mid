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

	tests := map[string]struct {
		props  map[string]property.Value
		create string
		update string
		delete string
	}{
		"lifecycle": {
			props: map[string]property.Value{
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
			},
			create: "test -f /create",
			update: "test -f /update",
			delete: "test ! -f /create && test ! -f /update",
		},
		"dir": {
			props: map[string]property.Value{
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
			},
			create: "test -f /tmp/create",
			update: "test -f /tmp/update",
			delete: "test ! -f /tmp/create && test ! -f /tmp/update",
		},
		"environment": {
			props: map[string]property.Value{
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
			},
			create: "grep -q create /tmp/environment",
			update: "grep -q update /tmp/environment",
			delete: "test ! -f /tmp/environment",
		},
		"expandArgumentVars": {
			props: map[string]property.Value{
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
			},
			create: "test -f /create",
			update: "test -f /update",
			delete: "test ! -f /create && test ! -f /update",
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			harness := NewProviderTestHarness(t, testmachine.Config{
				Backend: testmachine.DockerBackend,
			})
			defer harness.Close()

			t.Logf("%s: sending preview create request", name)
			_, err := harness.Server.Create(p.CreateRequest{
				Urn:        MakeURN("mid:resource:Exec"),
				Properties: property.NewMap(tc.props),
				DryRun:     true,
			})
			if !assert.NoError(t, err) {
				return
			}

			t.Logf("%s: sending create request", name)
			createResponse, err := harness.Server.Create(p.CreateRequest{
				Urn:        MakeURN("mid:resource:Exec"),
				Properties: property.NewMap(tc.props),
			})
			if !assert.NoError(t, err) {
				return
			}

			t.Logf("%s: checking create status", name)
			if !harness.AssertCommand(t, tc.create) {
				return
			}

			t.Logf("%s: sending preview update request", name)
			_, err = harness.Server.Update(p.UpdateRequest{
				Urn:    MakeURN("mid:resource:Exec"),
				State:  createResponse.Properties,
				Inputs: property.NewMap(tc.props),
				DryRun: true,
			})
			if !assert.NoError(t, err) {
				return
			}

			t.Logf("%s: sending update request", name)
			updateResponse, err := harness.Server.Update(p.UpdateRequest{
				Urn:    MakeURN("mid:resource:Exec"),
				State:  createResponse.Properties,
				Inputs: property.NewMap(tc.props),
			})
			if !assert.NoError(t, err) {
				return
			}

			if tc.update == "" {
				t.Logf("%s: update check is same as create", name)
				tc.update = tc.create
			}
			t.Logf("%s: checking update status", name)
			if !harness.AssertCommand(t, tc.update) {
				return
			}

			t.Logf("%s: sending delete request", name)
			err = harness.Server.Delete(p.DeleteRequest{
				Urn:        MakeURN("mid:resource:Exec"),
				Properties: updateResponse.Properties,
			})
			if !assert.NoError(t, err) {
				return
			}

			t.Logf("%s: checking delete status", name)
			if !harness.AssertCommand(t, tc.delete) {
				return
			}
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
