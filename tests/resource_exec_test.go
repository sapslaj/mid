package tests

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sapslaj/mid/tests/testmachine"
)

func TestResourceExec(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		props  resource.PropertyMap
		create string
		update string
		delete string
	}{
		"lifecycle": {
			props: resource.PropertyMap{
				"create": resource.NewObjectProperty(resource.PropertyMap{
					"command": resource.NewArrayProperty([]resource.PropertyValue{
						resource.NewStringProperty("touch"),
						resource.NewStringProperty("/create"),
					}),
				}),
				"update": resource.NewObjectProperty(resource.PropertyMap{
					"command": resource.NewArrayProperty([]resource.PropertyValue{
						resource.NewStringProperty("touch"),
						resource.NewStringProperty("/update"),
					}),
				}),
				"delete": resource.NewObjectProperty(resource.PropertyMap{
					"command": resource.NewArrayProperty([]resource.PropertyValue{
						resource.NewStringProperty("rm"),
						resource.NewStringProperty("-rf"),
						resource.NewStringProperty("/create"),
						resource.NewStringProperty("/update"),
					}),
				}),
			},
			create: "test -f /create",
			update: "test -f /update",
			delete: "test ! -f /create && test ! -f /update",
		},
		"dir": {
			props: resource.PropertyMap{
				"create": resource.NewObjectProperty(resource.PropertyMap{
					"command": resource.NewArrayProperty([]resource.PropertyValue{
						resource.NewStringProperty("touch"),
						resource.NewStringProperty("create"),
					}),
				}),
				"update": resource.NewObjectProperty(resource.PropertyMap{
					"command": resource.NewArrayProperty([]resource.PropertyValue{
						resource.NewStringProperty("touch"),
						resource.NewStringProperty("./tmp/update"),
					}),
					"dir": resource.NewStringProperty("/"),
				}),
				"delete": resource.NewObjectProperty(resource.PropertyMap{
					"command": resource.NewArrayProperty([]resource.PropertyValue{
						resource.NewStringProperty("rm"),
						resource.NewStringProperty("-rf"),
						resource.NewStringProperty("create"),
						resource.NewStringProperty("update"),
					}),
				}),
				"dir": resource.NewStringProperty("/tmp"),
			},
			create: "test -f /tmp/create",
			update: "test -f /tmp/update",
			delete: "test ! -f /tmp/create && test ! -f /tmp/update",
		},
		"environment": {
			props: resource.PropertyMap{
				"create": resource.NewObjectProperty(resource.PropertyMap{
					"command": resource.NewArrayProperty([]resource.PropertyValue{
						resource.NewStringProperty("/bin/sh"),
						resource.NewStringProperty("-c"),
						resource.NewStringProperty("echo $OP > $FILE"),
					}),
					"environment": resource.NewObjectProperty(resource.PropertyMap{
						"OP": resource.NewStringProperty("create"),
					}),
				}),
				"update": resource.NewObjectProperty(resource.PropertyMap{
					"command": resource.NewArrayProperty([]resource.PropertyValue{
						resource.NewStringProperty("/bin/sh"),
						resource.NewStringProperty("-c"),
						resource.NewStringProperty("echo $OP > $FILE"),
					}),
					"environment": resource.NewObjectProperty(resource.PropertyMap{
						"OP": resource.NewStringProperty("update"),
					}),
				}),
				"delete": resource.NewObjectProperty(resource.PropertyMap{
					"command": resource.NewArrayProperty([]resource.PropertyValue{
						resource.NewStringProperty("/bin/sh"),
						resource.NewStringProperty("-c"),
						resource.NewStringProperty("rm -f $FILE"),
					}),
				}),
				"environment": resource.NewObjectProperty(resource.PropertyMap{
					"FILE": resource.NewStringProperty("/tmp/environment"),
				}),
			},
			create: "grep -q create /tmp/environment",
			update: "grep -q update /tmp/environment",
			delete: "test ! -f /tmp/environment",
		},
		"expandArgumentVars": {
			props: resource.PropertyMap{
				"create": resource.NewObjectProperty(resource.PropertyMap{
					"command": resource.NewArrayProperty([]resource.PropertyValue{
						resource.NewStringProperty("touch"),
						resource.NewStringProperty("$FILE"),
					}),
					"environment": resource.NewObjectProperty(resource.PropertyMap{
						"FILE": resource.NewStringProperty("/create"),
					}),
				}),
				"update": resource.NewObjectProperty(resource.PropertyMap{
					"command": resource.NewArrayProperty([]resource.PropertyValue{
						resource.NewStringProperty("touch"),
						resource.NewStringProperty("$FILE"),
					}),
					"environment": resource.NewObjectProperty(resource.PropertyMap{
						"FILE": resource.NewStringProperty("/update"),
					}),
				}),
				"delete": resource.NewObjectProperty(resource.PropertyMap{
					"command": resource.NewArrayProperty([]resource.PropertyValue{
						resource.NewStringProperty("rm"),
						resource.NewStringProperty("-rf"),
						resource.NewStringProperty("/create"),
						resource.NewStringProperty("/update"),
					}),
				}),
				"expandArgumentVars": resource.NewBoolProperty(true),
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
			_, err := harness.Provider.Create(p.CreateRequest{
				Urn:        MakeURN("mid:resource:Exec"),
				Properties: tc.props,
				Preview:    true,
			})
			if !assert.NoError(t, err) {
				return
			}

			t.Logf("%s: sending create request", name)
			createResponse, err := harness.Provider.Create(p.CreateRequest{
				Urn:        MakeURN("mid:resource:Exec"),
				Properties: tc.props,
			})
			if !assert.NoError(t, err) {
				return
			}

			t.Logf("%s: checking create status", name)
			if !harness.AssertCommand(t, tc.create) {
				return
			}

			t.Logf("%s: sending preview update request", name)
			_, err = harness.Provider.Update(p.UpdateRequest{
				Urn:     MakeURN("mid:resource:Exec"),
				Olds:    createResponse.Properties,
				News:    tc.props,
				Preview: true,
			})
			if !assert.NoError(t, err) {
				return
			}

			t.Logf("%s: sending update request", name)
			updateResponse, err := harness.Provider.Update(p.UpdateRequest{
				Urn:  MakeURN("mid:resource:Exec"),
				Olds: createResponse.Properties,
				News: tc.props,
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
			err = harness.Provider.Delete(p.DeleteRequest{
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

	props := resource.PropertyMap{
		"create": resource.NewObjectProperty(resource.PropertyMap{
			"command": resource.NewArrayProperty([]resource.PropertyValue{
				resource.NewStringProperty("/bin/sh"),
				resource.NewStringProperty("-c"),
				resource.NewStringProperty("echo this is create stdout\necho this is create stderr 1>&2\n"),
			}),
		}),
		"update": resource.NewObjectProperty(resource.PropertyMap{
			"command": resource.NewArrayProperty([]resource.PropertyValue{
				resource.NewStringProperty("/bin/sh"),
				resource.NewStringProperty("-c"),
				resource.NewStringProperty("echo this is update stdout\necho this is update stderr 1>&2\n"),
			}),
		}),
	}

	createResponse, err := harness.Provider.Create(p.CreateRequest{
		Urn:        MakeURN("mid:resource:Exec"),
		Properties: props,
	})
	require.NoError(t, err)

	assert.Equal(t, "this is create stdout\n", createResponse.Properties["stdout"].StringValue())
	assert.Equal(t, "this is create stderr\n", createResponse.Properties["stderr"].StringValue())

	updateResponse, err := harness.Provider.Update(p.UpdateRequest{
		Urn:  MakeURN("mid:resource:Exec"),
		Olds: createResponse.Properties,
		News: props,
	})
	require.NoError(t, err)

	assert.Equal(t, "this is update stdout\n", updateResponse.Properties["stdout"].StringValue())
	assert.Equal(t, "this is update stderr\n", updateResponse.Properties["stderr"].StringValue())
}
