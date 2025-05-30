package tests

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/assert"

	"github.com/sapslaj/mid/tests/testmachine"
)

func TestResourceApt(t *testing.T) {
	t.Parallel()

	harness := NewProviderTestHarness(t, testmachine.Config{
		Backend: testmachine.DockerBackend,
	})
	defer harness.Close()

	tests := map[string]struct {
		props  map[string]property.Value
		create string
		update string
		delete string
	}{
		"installs vim": {
			props: map[string]property.Value{
				"name": property.New("vim"),
			},
			create: "test -f /usr/bin/vim",
			delete: "test ! -f /usr/bin/vim",
		},
		"installs multiple packages": {
			props: map[string]property.Value{
				"names": property.New([]property.Value{
					property.New("curl"),
					property.New("wget"),
				}),
				"ensure": property.New("latest"),
			},
			create: "test -f /usr/bin/curl && test -f /usr/bin/wget",
			delete: "test ! -f /usr/bin/curl && test ! -f /usr/bin/wget",
		},
		"upgrade all packages": {
			props: map[string]property.Value{
				"name":        property.New("*"),
				"ensure":      property.New("latest"),
				"autoremove":  property.New(true),
				"updateCache": property.New(true),
			},
			create: "true", // no test
			delete: "true", // no test
		},
		"apt clean": {
			props: map[string]property.Value{
				"clean": property.New(true),
			},
			create: "true", // no test
			delete: "true", // no test
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			// WARN: do not use t.Parallel() here

			t.Logf("%s: sending preview create request", name)
			_, err := harness.Server.Create(p.CreateRequest{
				Urn:        MakeURN("mid:resource:Apt"),
				Properties: property.NewMap(tc.props),
				DryRun:     true,
			})
			if !assert.NoError(t, err) {
				return
			}

			t.Logf("%s: sending create request", name)
			createResponse, err := harness.Server.Create(p.CreateRequest{
				Urn:        MakeURN("mid:resource:Apt"),
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
				Urn:    MakeURN("mid:resource:Apt"),
				State:  createResponse.Properties,
				Inputs: property.NewMap(tc.props),
				DryRun: true,
			})
			if !assert.NoError(t, err) {
				return
			}

			t.Logf("%s: sending update request", name)
			updateResponse, err := harness.Server.Update(p.UpdateRequest{
				Urn:    MakeURN("mid:resource:Apt"),
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
				Urn:        MakeURN("mid:resource:Apt"),
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
